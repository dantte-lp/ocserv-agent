package logging

import (
	"context"
	"io"
	"log/slog"
	"os"

	"github.com/dantte-lp/ocserv-agent/internal/config"
	"github.com/dantte-lp/ocserv-agent/internal/telemetry"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/trace"
)

// NewLogger создает новый slog.Logger с интеграцией OpenTelemetry.
//
// Особенности:
//   - JSON или Text формат
//   - Автоматическая корреляция с trace ID через otelslog bridge (если OTLP logs включен)
//   - Multi-handler: локальный + OTLP + VictoriaLogs (опционально)
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
//	otlpCfg := config.OTLPConfig{
//	    Enabled:     true,
//	    LogsEnabled: true,
//	}
//	logger := NewLogger(cfg, nil, &otlpCfg)
//	logger.Info("service started", "version", "0.7.0")
func NewLogger(cfg config.LoggingConfig, victoriaLogsCfg *config.VictoriaLogsConfig, otlpCfg *config.OTLPConfig) *slog.Logger {
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

	// Создаем базовый handler для локального вывода
	var localHandler slog.Handler
	if cfg.Format == "json" {
		localHandler = slog.NewJSONHandler(writer, opts)
	} else {
		localHandler = slog.NewTextHandler(writer, opts)
	}

	// Оборачиваем в VictoriaLogs handler если включен
	if victoriaLogsCfg != nil && victoriaLogsCfg.Enabled {
		localHandler = telemetry.NewVictoriaLogsHandler(*victoriaLogsCfg, localHandler)
	}

	// Проверяем, включен ли OTLP logs exporter
	if otlpCfg != nil && otlpCfg.Enabled && otlpCfg.LogsEnabled {
		// Получаем глобальный LoggerProvider
		loggerProvider := global.GetLoggerProvider()

		// Создаем otelslog handler для отправки логов через OTLP
		otlpHandler := otelslog.NewHandler("ocserv-agent",
			otelslog.WithLoggerProvider(loggerProvider),
		)

		// Создаем multi-handler: локальный + OTLP
		multiHandler := NewMultiHandler(localHandler, otlpHandler)
		return slog.New(multiHandler)
	}

	// Возвращаем logger с локальным handler
	return slog.New(localHandler)
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

// MultiHandler реализует slog.Handler для отправки логов в несколько handler'ов одновременно.
// Используется для параллельной записи в локальный лог и OTLP exporter.
type MultiHandler struct {
	handlers []slog.Handler
}

// NewMultiHandler создает новый MultiHandler с несколькими обработчиками.
func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{
		handlers: handlers,
	}
}

// Enabled проверяет, активен ли хотя бы один handler для данного уровня.
func (m *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle отправляет запись во все handler'ы.
func (m *MultiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, h := range m.handlers {
		// Клонируем record для каждого handler, чтобы избежать race conditions
		if err := h.Handle(ctx, record.Clone()); err != nil {
			// Продолжаем обработку даже если один из handler'ов вернул ошибку
			// Можно добавить логирование ошибок, но это может привести к циклу
			continue
		}
	}
	return nil
}

// WithAttrs добавляет атрибуты ко всем handler'ам.
func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: newHandlers}
}

// WithGroup добавляет группу ко всем handler'ам.
func (m *MultiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		newHandlers[i] = h.WithGroup(name)
	}
	return &MultiHandler{handlers: newHandlers}
}
