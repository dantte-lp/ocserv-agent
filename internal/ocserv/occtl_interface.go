package ocserv

import "context"

// OcctlInterface определяет интерфейс для операций occtl
// Используется для моков в тестах
type OcctlInterface interface {
	// ShowUsers retrieves list of connected users
	ShowUsers(ctx context.Context) ([]User, error)

	// DisconnectUser disconnects a user by username
	DisconnectUser(ctx context.Context, username string) error

	// DisconnectID disconnects a user by session ID
	DisconnectID(ctx context.Context, id string) error

	// ShowStatus returns ocserv server status
	ShowStatus(ctx context.Context) (*ServerStatus, error)

	// ShowStats returns ocserv server statistics
	ShowStats(ctx context.Context) (*ServerStats, error)

	// Reload reloads ocserv configuration
	Reload(ctx context.Context) error
}

// Ensure OcctlManager implements OcctlInterface
var _ OcctlInterface = (*OcctlManager)(nil)
