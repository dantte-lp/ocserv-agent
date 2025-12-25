package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dantte-lp/ocserv-agent/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// ExporterType определяет тип экспортера для телеметрии.
type ExporterType string

const (
	ExporterOTLP            ExporterType = "otlp"
	ExporterVictoriaLogs    ExporterType = "victoria_logs"
	ExporterVictoriaMetrics ExporterType = "victoria_metrics"
)

// InitProviders инициализирует TracerProvider, MeterProvider и VictoriaLogs/VictoriaMetrics.
// Возвращает shutdown функцию для корректного закрытия при остановке приложения.
//
// Пример использования:
//
//	shutdown, err := telemetry.InitProviders(ctx, cfg, logger)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer shutdown(context.Background())
func InitProviders(ctx context.Context, cfg config.TelemetryConfig, logger *slog.Logger) (shutdown func(context.Context) error, err error) {
	if !cfg.Enabled {
		return func(context.Context) error { return nil }, nil
	}

	var shutdownFuncs []func(context.Context) error

	// Создаем resource с метаданными сервиса
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
		resource.WithHost(),
		resource.WithProcess(),
		resource.WithOS(),
		resource.WithContainer(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Traces - всегда через OTLP (VictoriaMetrics/VictoriaLogs не поддерживают traces напрямую)
	if cfg.OTLP.Enabled {
		tracerProvider, err := initTracerProvider(ctx, cfg, res)
		if err != nil {
			return nil, fmt.Errorf("failed to init tracer provider: %w", err)
		}
		otel.SetTracerProvider(tracerProvider)
		shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)

		logger.Info("OTLP tracer provider initialized",
			"endpoint", cfg.OTLP.Endpoint,
			"protocol", cfg.OTLP.Protocol,
			"insecure", cfg.OTLP.Insecure,
		)
	}

	// Metrics - OTLP или VictoriaMetrics
	if cfg.OTLP.Enabled {
		meterProvider, err := initMeterProvider(ctx, cfg, res)
		if err != nil {
			// Cleanup tracer if meter fails
			for _, fn := range shutdownFuncs {
				_ = fn(ctx)
			}
			return nil, fmt.Errorf("failed to init meter provider: %w", err)
		}
		otel.SetMeterProvider(meterProvider)
		shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)

		logger.Info("OTLP meter provider initialized",
			"endpoint", cfg.OTLP.Endpoint,
			"protocol", cfg.OTLP.Protocol,
		)
	}

	// Prometheus HTTP server для /metrics endpoint
	if cfg.Prometheus.Enabled {
		promServer, err := NewPrometheusServer(ctx, cfg, res, logger)
		if err != nil {
			// Cleanup при ошибке
			for _, fn := range shutdownFuncs {
				_ = fn(ctx)
			}
			return nil, fmt.Errorf("failed to init prometheus server: %w", err)
		}

		if promServer != nil {
			if err := promServer.Start(); err != nil {
				// Cleanup при ошибке
				for _, fn := range shutdownFuncs {
					_ = fn(ctx)
				}
				return nil, fmt.Errorf("failed to start prometheus server: %w", err)
			}

			shutdownFuncs = append(shutdownFuncs, promServer.Shutdown)

			logger.Info("Prometheus metrics server initialized",
				"address", cfg.Prometheus.Address,
			)
		}
	}

	// VictoriaMetrics через Prometheus remote_write (DEPRECATED - используйте OTLP)
	// Оставлено для обратной совместимости
	if cfg.VictoriaMetrics.Enabled {
		logger.Warn("VictoriaMetrics direct integration is deprecated, use OTLP or Prometheus exporter instead",
			"endpoint", cfg.VictoriaMetrics.Endpoint,
		)
		// Старый код закомментирован - удалите секцию victoria_metrics из config
	}

	// Logs - VictoriaLogs handler будет настраиваться в logging package
	// Здесь только логируем информацию о конфигурации
	if cfg.VictoriaLogs.Enabled {
		logger.Info("VictoriaLogs handler configuration detected",
			"endpoint", cfg.VictoriaLogs.Endpoint,
			"batch_size", cfg.VictoriaLogs.BatchSize,
			"flush_interval", cfg.VictoriaLogs.FlushInterval,
		)
		// NOTE: Реальная инициализация VictoriaLogs handler происходит в internal/logging
	}

	// Настройка propagation для распределенной трассировки
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Shutdown функция
	return func(ctx context.Context) error {
		shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var errs []error
		for _, fn := range shutdownFuncs {
			if err := fn(shutdownCtx); err != nil {
				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			return fmt.Errorf("shutdown errors: %v", errs)
		}
		return nil
	}, nil
}

// initTracerProvider создает TracerProvider с OTLP exporter (gRPC или HTTP).
func initTracerProvider(ctx context.Context, cfg config.TelemetryConfig, res *resource.Resource) (*trace.TracerProvider, error) {
	var exporter trace.SpanExporter
	var err error

	// Выбираем транспорт на основе cfg.OTLP.Protocol
	protocol := strings.ToLower(cfg.OTLP.Protocol)
	switch protocol {
	case "http", "http/protobuf":
		// HTTP транспорт
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(cfg.OTLP.Endpoint),
			otlptracehttp.WithTimeout(cfg.OTLP.Timeout),
		}
		if cfg.OTLP.Insecure {
			opts = append(opts, otlptracehttp.WithInsecure())
		}
		exporter, err = otlptracehttp.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP trace exporter: %w", err)
		}

	case "grpc", "":
		// gRPC транспорт (default)
		opts := []otlptracegrpc.Option{
			otlptracegrpc.WithEndpoint(cfg.OTLP.Endpoint),
			otlptracegrpc.WithTimeout(cfg.OTLP.Timeout),
		}
		if cfg.OTLP.Insecure {
			opts = append(opts, otlptracegrpc.WithInsecure())
		}
		exporter, err = otlptracegrpc.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create gRPC trace exporter: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported OTLP protocol: %s (supported: grpc, http)", cfg.OTLP.Protocol)
	}

	// Определяем sampler
	sampler := trace.ParentBased(trace.TraceIDRatioBased(cfg.SampleRate))
	if cfg.SampleRate >= 1.0 {
		sampler = trace.AlwaysSample()
	}

	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(sampler),
	)

	return tp, nil
}

// initMeterProvider создает MeterProvider с OTLP exporter (gRPC или HTTP).
func initMeterProvider(ctx context.Context, cfg config.TelemetryConfig, res *resource.Resource) (*metric.MeterProvider, error) {
	var exporter metric.Exporter
	var err error

	// Выбираем транспорт на основе cfg.OTLP.Protocol
	protocol := strings.ToLower(cfg.OTLP.Protocol)
	switch protocol {
	case "http", "http/protobuf":
		// HTTP транспорт
		opts := []otlpmetrichttp.Option{
			otlpmetrichttp.WithEndpoint(cfg.OTLP.Endpoint),
			otlpmetrichttp.WithTimeout(cfg.OTLP.Timeout),
		}
		if cfg.OTLP.Insecure {
			opts = append(opts, otlpmetrichttp.WithInsecure())
		}
		exporter, err = otlpmetrichttp.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create HTTP metric exporter: %w", err)
		}

	case "grpc", "":
		// gRPC транспорт (default)
		opts := []otlpmetricgrpc.Option{
			otlpmetricgrpc.WithEndpoint(cfg.OTLP.Endpoint),
			otlpmetricgrpc.WithTimeout(cfg.OTLP.Timeout),
		}
		if cfg.OTLP.Insecure {
			opts = append(opts, otlpmetricgrpc.WithInsecure())
		}
		exporter, err = otlpmetricgrpc.New(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create gRPC metric exporter: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported OTLP protocol: %s (supported: grpc, http)", cfg.OTLP.Protocol)
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter,
			metric.WithInterval(30*time.Second),
		)),
		metric.WithResource(res),
	)

	return mp, nil
}
