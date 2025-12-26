package domain

import (
	"github.com/cockroachdb/errors"
)

// Sentinel errors для domain-логики ocserv-agent.
// Используем cockroachdb/errors для структурированных ошибок с stack traces.
var (
	// ErrCommandFailed возвращается когда выполнение команды завершилось с ошибкой.
	ErrCommandFailed = errors.New("command execution failed")

	// ErrInvalidArgument возвращается при некорректных входных данных.
	ErrInvalidArgument = errors.New("invalid argument")

	// ErrUnauthorized возвращается при недостаточных правах доступа.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrTimeout возвращается при превышении таймаута операции.
	ErrTimeout = errors.New("operation timeout")

	// ErrOcservNotRunning возвращается когда ocserv daemon не запущен.
	ErrOcservNotRunning = errors.New("ocserv is not running")

	// ErrOcservAlreadyRunning возвращается когда ocserv уже запущен.
	ErrOcservAlreadyRunning = errors.New("ocserv is already running")

	// ErrConnectionFailed возвращается при ошибке подключения к control server.
	ErrConnectionFailed = errors.New("connection to control server failed")

	// ErrCircuitBreakerOpen возвращается когда circuit breaker открыт.
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")

	// ErrConfigInvalid возвращается при некорректной конфигурации.
	ErrConfigInvalid = errors.New("invalid configuration")

	// ErrCertificateInvalid возвращается при проблемах с TLS сертификатами.
	ErrCertificateInvalid = errors.New("invalid or expired certificate")

	// ErrResourceNotFound возвращается когда запрашиваемый ресурс не найден.
	ErrResourceNotFound = errors.New("resource not found")

	// ErrAlreadyExists возвращается когда ресурс уже существует.
	ErrAlreadyExists = errors.New("resource already exists")
)

// WrapWithContext оборачивает ошибку с дополнительным контекстом.
// Использует cockroachdb/errors для сохранения stack trace.
//
// Пример:
//
//	err := occtl.Execute("show", "users")
//	if err != nil {
//	    return WrapWithContext(err, "failed to get users list",
//	        "command", "show users",
//	        "socket", "/run/ocserv/occtl.socket",
//	    )
//	}
func WrapWithContext(err error, msg string, keyvals ...interface{}) error {
	if err == nil {
		return nil
	}

	// errors.Wrap уже является обёрткой внешней ошибки
	wrapped := errors.Wrap(err, msg)

	// Добавляем key-value пары как details
	// errors.WithDetailf возвращает аннотированную ошибку
	for i := 0; i < len(keyvals)-1; i += 2 {
		if key, ok := keyvals[i].(string); ok {
			wrapped = errors.WithDetailf(wrapped, "%s: %v", key, keyvals[i+1])
		}
	}

	// Возвращаем полностью обёрнутую ошибку с контекстом
	// wrapped уже содержит обёртку через errors.Wrap()
	//nolint:wrapcheck // wrapped is already wrapped by errors.Wrap above
	return wrapped
}

// IsTemporary проверяет является ли ошибка временной (recoverable).
// Временные ошибки: таймауты, недоступность сервиса, circuit breaker.
func IsTemporary(err error) bool {
	return errors.Is(err, ErrTimeout) ||
		errors.Is(err, ErrConnectionFailed) ||
		errors.Is(err, ErrCircuitBreakerOpen) ||
		errors.Is(err, ErrOcservNotRunning)
}

// IsPermanent проверяет является ли ошибка постоянной (non-recoverable).
// Постоянные ошибки: некорректные аргументы, неавторизованный доступ, невалидная конфигурация.
func IsPermanent(err error) bool {
	return errors.Is(err, ErrInvalidArgument) ||
		errors.Is(err, ErrUnauthorized) ||
		errors.Is(err, ErrConfigInvalid) ||
		errors.Is(err, ErrCertificateInvalid)
}

// GetRootCause возвращает корневую причину ошибки.
// Используется для анализа первопричины в логировании и метриках.
// Сознательно возвращает unwrapped ошибку для получения root cause.
//
//nolint:wrapcheck // intentionally unwraps to get root cause
func GetRootCause(err error) error {
	return errors.UnwrapAll(err)
}

// FormatWithStack форматирует ошибку со stack trace для логирования.
func FormatWithStack(err error) string {
	return errors.Newf("%+v", err).Error()
}
