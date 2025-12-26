package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dantte-lp/ocserv-agent/internal/config"
	"github.com/dantte-lp/ocserv-agent/internal/ipc"
	"github.com/dantte-lp/ocserv-agent/internal/logging"
	"github.com/dantte-lp/ocserv-agent/internal/ocserv"
	"github.com/dantte-lp/ocserv-agent/internal/portal"
	"github.com/dantte-lp/ocserv-agent/internal/stats"
	"github.com/dantte-lp/ocserv-agent/internal/telemetry"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
)

// runServerPhase2 запускает агент с поддержкой IPC server и stats poller (Фаза 2)
func runServerPhase2(cfg *config.Config, _ *slog.Logger) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Инициализация logger (должна быть до инициализации OpenTelemetry)
	logger := logging.NewLogger(cfg.Logging, &cfg.Telemetry.VictoriaLogs, &cfg.Telemetry.OTLP)

	// Инициализация OpenTelemetry
	logger.InfoContext(ctx, "initializing telemetry")
	shutdown, err := telemetry.InitProviders(ctx, cfg.Telemetry, logger)
	if err != nil {
		return fmt.Errorf("telemetry init: %w", err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := shutdown(shutdownCtx); err != nil {
			logger.ErrorContext(shutdownCtx, "telemetry shutdown error",
				slog.String("error", err.Error()),
			)
		}
	}()

	// Получаем tracer и meter из глобального otel
	tracerProvider := otel.GetTracerProvider()
	tracer := tracerProvider.Tracer(cfg.Telemetry.ServiceName)
	meterProvider := otel.GetMeterProvider()
	meter := meterProvider.Meter(cfg.Telemetry.ServiceName)

	// Создаем occtl manager для работы с ocserv
	// Используем zerolog для совместимости с существующим кодом ocserv
	zlogger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	occtlMgr := ocserv.NewOcctlManager(
		cfg.Ocserv.CtlSocket,
		cfg.Security.SudoUser,
		cfg.Security.MaxCommandTimeout,
		zlogger,
	)

	// Создаем portal client
	logger.InfoContext(ctx, "connecting to portal",
		slog.String("address", cfg.Portal.Address),
		slog.Bool("tls", !cfg.Portal.Insecure),
	)

	portalClient, err := portal.NewClient(
		ctx,
		&portal.Config{
			Address:  cfg.Portal.Address,
			TLSCert:  cfg.Portal.TLSCert,
			TLSKey:   cfg.Portal.TLSKey,
			TLSCA:    cfg.Portal.TLSCA,
			Timeout:  cfg.Portal.Timeout,
			Insecure: cfg.Portal.Insecure,
		},
		logger,
		tracer,
		tracerProvider,
		meterProvider,
	)
	if err != nil {
		return fmt.Errorf("create portal client: %w", err)
	}
	defer portalClient.Close()

	// Создаем IPC handler
	logger.InfoContext(ctx, "creating IPC handler")
	ipcHandler, err := ipc.NewHandler(&ipc.HandlerConfig{
		Logger:       logger,
		Tracer:       tracer,
		Meter:        meter,
		PortalClient: portalClient,
		Timeout:      cfg.IPC.Timeout,
	})
	if err != nil {
		return fmt.Errorf("create IPC handler: %w", err)
	}

	// Создаем IPC server
	logger.InfoContext(ctx, "creating IPC server",
		slog.String("socket", cfg.IPC.SocketPath),
	)
	ipcServer, err := ipc.NewServer(&ipc.ServerConfig{
		SocketPath: cfg.IPC.SocketPath,
		Handler:    ipcHandler,
		Logger:     logger,
		Tracer:     tracer,
		Meter:      meter,
	})
	if err != nil {
		return fmt.Errorf("create IPC server: %w", err)
	}

	// Запускаем IPC server
	if err := ipcServer.Start(ctx); err != nil {
		return fmt.Errorf("start IPC server: %w", err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := ipcServer.Stop(shutdownCtx); err != nil {
			logger.ErrorContext(shutdownCtx, "IPC server shutdown error",
				slog.String("error", err.Error()),
			)
		}
	}()

	// Создаем stats poller
	logger.InfoContext(ctx, "creating stats poller",
		slog.Duration("interval", cfg.Health.MetricsInterval),
	)
	statsPoller, err := stats.NewPoller(&stats.PollerConfig{
		OcctlManager: occtlMgr,
		Logger:       logger,
		Tracer:       tracer,
		Meter:        meter,
		Interval:     cfg.Health.MetricsInterval,
	})
	if err != nil {
		return fmt.Errorf("create stats poller: %w", err)
	}

	// Регистрируем callback для событий сессий
	statsPoller.RegisterCallback(func(ctx context.Context, event stats.SessionEvent) {
		logger.InfoContext(ctx, "session event",
			slog.String("type", string(event.Type)),
			slog.String("username", event.Session.Username),
			slog.String("client_ip", event.Session.ClientIP),
			slog.String("vpn_ip", event.Session.VPNIP),
		)

		// Отправляем события в portal
		switch event.Type {
		case stats.SessionConnected:
			if err := portalClient.ReportConnect(
				ctx,
				fmt.Sprintf("%d", event.Session.ID),
				event.Session.Username,
				event.Session.GroupName,
				event.Session.ClientIP,
				event.Session.VPNIP,
				"", // device
			); err != nil {
				logger.ErrorContext(ctx, "failed to report connect",
					slog.String("error", err.Error()),
				)
			}

		case stats.SessionDisconnected:
			duration := time.Since(event.Session.ConnectedAt)
			if err := portalClient.ReportDisconnect(
				ctx,
				fmt.Sprintf("%d", event.Session.ID),
				event.Session.Username,
				duration,
				event.Session.BytesRX,
				event.Session.BytesTX,
			); err != nil {
				logger.ErrorContext(ctx, "failed to report disconnect",
					slog.String("error", err.Error()),
				)
			}
		}
	})

	// Запускаем stats poller
	if err := statsPoller.Start(ctx); err != nil {
		return fmt.Errorf("start stats poller: %w", err)
	}
	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := statsPoller.Stop(shutdownCtx); err != nil {
			logger.ErrorContext(shutdownCtx, "stats poller shutdown error",
				slog.String("error", err.Error()),
			)
		}
	}()

	logger.InfoContext(ctx, "agent started successfully",
		slog.String("ipc_socket", cfg.IPC.SocketPath),
		slog.String("portal_address", cfg.Portal.Address),
		slog.Duration("stats_interval", cfg.Health.MetricsInterval),
	)

	// Ожидание сигналов завершения
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	sig := <-sigCh
	logger.InfoContext(ctx, "received shutdown signal",
		slog.String("signal", sig.String()),
	)

	// Graceful shutdown обрабатывается через defer'ы выше
	logger.InfoContext(ctx, "shutdown complete")
	return nil
}
