package logging

import (
	"context"
	"log/slog"
)

type contextKey string

const loggerKey contextKey = "logger"

// ToContext сохраняет logger в context для передачи между функциями.
//
// Пример:
//
//	logger := NewLogger(cfg)
//	ctx := ToContext(context.Background(), logger)
//	processRequest(ctx) // внутри можно достать logger через FromContext
func ToContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext извлекает logger из context.
// Если logger не найден, возвращает default logger.
//
// Пример:
//
//	func processRequest(ctx context.Context) {
//	    logger := FromContext(ctx)
//	    logger.Info("processing started")
//	}
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	// Fallback to default logger
	return slog.Default()
}

// WithFields добавляет дополнительные поля к logger из context.
// Возвращает новый context с обновленным logger.
//
// Пример:
//
//	ctx = WithFields(ctx, "request_id", reqID, "user_id", userID)
//	logger := FromContext(ctx)
//	logger.Info("user authenticated") // автоматически включит request_id и user_id
func WithFields(ctx context.Context, args ...any) context.Context {
	logger := FromContext(ctx)
	newLogger := logger.With(args...)
	return ToContext(ctx, newLogger)
}

// Debug логирует debug-сообщение используя logger из context.
func Debug(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).DebugContext(ctx, msg, args...)
}

// Info логирует info-сообщение используя logger из context.
func Info(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).InfoContext(ctx, msg, args...)
}

// Warn логирует warning-сообщение используя logger из context.
func Warn(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).WarnContext(ctx, msg, args...)
}

// Error логирует error-сообщение используя logger из context.
func Error(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).ErrorContext(ctx, msg, args...)
}

// WithError добавляет ошибку как поле к logger в context.
//
// Пример:
//
//	if err != nil {
//	    ctx = WithError(ctx, err)
//	    Error(ctx, "failed to connect")
//	    return err
//	}
func WithError(ctx context.Context, err error) context.Context {
	if err == nil {
		return ctx
	}
	return WithFields(ctx, "error", err.Error())
}

// WithRequestID добавляет request_id к logger в context.
// Полезно для корреляции логов одного запроса.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return WithFields(ctx, "request_id", requestID)
}

// WithAgentID добавляет agent_id к logger в context.
func WithAgentID(ctx context.Context, agentID string) context.Context {
	return WithFields(ctx, "agent_id", agentID)
}

// WithCommand добавляет информацию о команде к logger в context.
func WithCommand(ctx context.Context, commandType string, args ...string) context.Context {
	return WithFields(ctx,
		"command_type", commandType,
		"command_args", args,
	)
}
