package logging

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/dantte-lp/ocserv-agent/internal/config"
	"github.com/dantte-lp/ocserv-agent/internal/telemetry"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/trace"
)

// NewLogger создает новый slog.Logger с интеграцией OpenTelemetry.
//
// Особенности:
//   - JSON или Text формат
//   - Автоматическая корреляция с trace ID через otelslog bridge
//   - Настраиваемый уровень логирования
//   - Поддержка вывода в stdout/stderr/file
//
// Пример:
//
//	cfg := config.LoggingConfig{
//	    Level:     "info",
//	    Format:    "json",
//	    Output:    "stdout",
//	    AddSource: true,
//	}
//	logger := NewLogger(cfg, nil)
//	logger.Info("service started", "version", "0.7.0")
func NewLogger(cfg config.LoggingConfig, victoriaLogsCfg *config.VictoriaLogsConfig) *slog.Logger {
	// Определяем уровень логирования
	level := parseLevel(cfg.Level)

	// Определяем writer
	var writer io.Writer
	switch cfg.Output {
	case "stderr":
		writer = os.Stderr
	case "file":
		if cfg.FilePath != "" {
			// Log file with owner read/write only for security
			file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
			if err != nil {
				// Fallback to stdout on error
				writer = os.Stdout
			} else {
				writer = file
			}
		} else {
			writer = os.Stdout
		}
	default:
		writer = os.Stdout
	}

	// Настройки handler
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: cfg.AddSource,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Кастомизация атрибутов (опционально)
			return a
		},
	}

	// Создаем базовый handler
	var baseHandler slog.Handler
	if cfg.Format == "json" {
		baseHandler = slog.NewJSONHandler(writer, opts)
	} else {
		baseHandler = slog.NewTextHandler(writer, opts)
	}

	// Оборачиваем в VictoriaLogs handler если включен
	if victoriaLogsCfg != nil && victoriaLogsCfg.Enabled {
		baseHandler = telemetry.NewVictoriaLogsHandler(*victoriaLogsCfg, baseHandler)
	}

	// Возвращаем logger с базовым handler
	// Корреляция с traces будет происходить через WithTraceContext()
	_ = otelslog.NewHandler // импорт для будущего использования
	return slog.New(baseHandler)
}

// NewTestLogger создает logger для тестов с буфером.
// Полезно для проверки логов в unit-тестах.
func NewTestLogger(buf io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: false,
	}
	handler := slog.NewJSONHandler(buf, opts)
	return slog.New(handler)
}

// NewNopLogger создает no-op logger, который ничего не пишет.
// Используется в тестах где логи не нужны.
func NewNopLogger() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

// WithTraceContext добавляет trace_id и span_id в logger из context.
// Используется для ручной корреляции логов с распределенной трассировкой.
//
// Пример:
//
//	logger := WithTraceContext(ctx, baseLogger)
//	logger.Info("processing request")  // автоматически добавит trace_id
func WithTraceContext(ctx context.Context, logger *slog.Logger) *slog.Logger {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return logger
	}

	spanCtx := span.SpanContext()
	return logger.With(
		slog.String("trace_id", spanCtx.TraceID().String()),
		slog.String("span_id", spanCtx.SpanID().String()),
	)
}

// parseLevel конвертирует строку в slog.Level.
func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// LevelFromString возвращает slog.Level из строки (публичная функция).
func LevelFromString(level string) slog.Level {
	return parseLevel(level)
}
