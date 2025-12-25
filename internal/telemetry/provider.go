package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/dantte-lp/ocserv-agent/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
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
		)
	}

	if cfg.VictoriaMetrics.Enabled {
		vmExporter := NewVictoriaMetricsExporter(cfg.VictoriaMetrics)
		vmExporter.Start(ctx)
		shutdownFuncs = append(shutdownFuncs, func(ctx context.Context) error {
			return vmExporter.Close()
		})

		logger.Info("VictoriaMetrics exporter initialized",
			"endpoint", cfg.VictoriaMetrics.Endpoint,
			"push_interval", cfg.VictoriaMetrics.PushInterval,
		)
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

// initTracerProvider создает TracerProvider с OTLP exporter.
func initTracerProvider(ctx context.Context, cfg config.TelemetryConfig, res *resource.Resource) (*trace.TracerProvider, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OTLP.Endpoint),
	}
	if cfg.OTLP.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
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

// initMeterProvider создает MeterProvider с OTLP exporter.
func initMeterProvider(ctx context.Context, cfg config.TelemetryConfig, res *resource.Resource) (*metric.MeterProvider, error) {
	opts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(cfg.OTLP.Endpoint),
	}
	if cfg.OTLP.Insecure {
		opts = append(opts, otlpmetricgrpc.WithInsecure())
	}

	exporter, err := otlpmetricgrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter,
			metric.WithInterval(30*time.Second),
		)),
		metric.WithResource(res),
	)

	return mp, nil
}
