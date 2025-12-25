package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/dantte-lp/ocserv-agent/internal/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

// PrometheusServer представляет HTTP сервер для /metrics endpoint.
type PrometheusServer struct {
	config   config.PrometheusConfig
	exporter *prometheus.Exporter
	server   *http.Server
	logger   *slog.Logger
}

// NewPrometheusServer создает новый Prometheus HTTP server.
func NewPrometheusServer(ctx context.Context, cfg config.TelemetryConfig, res *resource.Resource, logger *slog.Logger) (*PrometheusServer, error) {
	if !cfg.Prometheus.Enabled {
		return nil, nil
	}

	// Создаем Prometheus exporter
	exporter, err := prometheus.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus exporter: %w", err)
	}

	// Создаем MeterProvider с Prometheus exporter
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(exporter),
		metric.WithResource(res),
	)

	// Устанавливаем как глобальный MeterProvider (или используем отдельно)
	otel.SetMeterProvider(meterProvider)

	// Создаем HTTP мультиплексор
	mux := http.NewServeMux()

	// Prometheus exporter implements prometheus.Collector interface
	// Use promhttp.Handler() для HTTP endpoint
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Создаем HTTP сервер
	server := &http.Server{
		Addr:    cfg.Prometheus.Address,
		Handler: mux,
	}

	return &PrometheusServer{
		config:   cfg.Prometheus,
		exporter: exporter,
		server:   server,
		logger:   logger,
	}, nil
}

// Start запускает Prometheus HTTP server в отдельной горутине.
func (s *PrometheusServer) Start() error {
	if s == nil || !s.config.Enabled {
		return nil
	}

	s.logger.Info("Starting Prometheus metrics server",
		"address", s.config.Address,
	)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Prometheus server error",
				"error", err,
			)
		}
	}()

	return nil
}

// Shutdown корректно завершает работу сервера.
func (s *PrometheusServer) Shutdown(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}

	s.logger.Info("Shutting down Prometheus metrics server")
	return s.server.Shutdown(ctx)
}
