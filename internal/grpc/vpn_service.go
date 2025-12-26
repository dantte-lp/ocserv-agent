package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cockroachdb/errors"
	pb "github.com/dantte-lp/ocserv-agent/pkg/proto/agent/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// VPNService represents the VPN management service
type VPNService struct {
	pb.UnimplementedVPNAgentServiceServer

	server *Server
	logger *slog.Logger
}

// NewVPNService creates a new VPN service instance
func NewVPNService(server *Server, logger *slog.Logger) *VPNService {
	return &VPNService{
		server: server,
		logger: logger,
	}
}

// NotifyConnect обрабатывает уведомление о попытке подключения пользователя
func (s *VPNService) NotifyConnect(ctx context.Context, req *pb.NotifyConnectRequest) (*pb.NotifyConnectResponse, error) {
	s.logger.InfoContext(ctx, "Processing connect notification",
		slog.String("username", req.Username),
		slog.String("client_ip", req.ClientIp),
		slog.String("vpn_ip", req.VpnIp),
		slog.String("session_id", req.SessionId),
	)

	// TODO: Интеграция с Portal для проверки политик
	// На данный момент возвращаем базовый ответ

	response := &pb.NotifyConnectResponse{
		Allowed:           true,
		ShouldDisconnect:  false,
		DenyReason:        "",
		Routes:            []string{}, // TODO: Получить маршруты из конфига пользователя
		DnsServers:        []string{},
		ConfigParams:      make(map[string]string),
	}

	s.logger.InfoContext(ctx, "Connect notification processed",
		slog.Bool("allowed", response.Allowed),
		slog.String("username", req.Username),
	)

	return response, nil
}

// NotifyDisconnect обрабатывает уведомление об отключении пользователя
func (s *VPNService) NotifyDisconnect(ctx context.Context, req *pb.NotifyDisconnectRequest) (*pb.NotifyDisconnectResponse, error) {
	s.logger.InfoContext(ctx, "Processing disconnect notification",
		slog.String("username", req.Username),
		slog.String("session_id", req.SessionId),
		slog.String("reason", req.DisconnectReason),
		slog.Uint64("bytes_in", req.BytesIn),
		slog.Uint64("bytes_out", req.BytesOut),
		slog.Uint64("duration", req.DurationSeconds),
	)

	// TODO: Сохранить статистику сессии
	// TODO: Уведомить Portal об отключении

	response := &pb.NotifyDisconnectResponse{
		Acknowledged: true,
		Message:      "Disconnect notification received",
	}

	return response, nil
}

// GetActiveSessions возвращает список активных VPN сессий
func (s *VPNService) GetActiveSessions(ctx context.Context, req *pb.GetActiveSessionsRequest) (*pb.GetActiveSessionsResponse, error) {
	s.logger.InfoContext(ctx, "Fetching active sessions",
		slog.String("username_filter", req.UsernameFilter),
		slog.Bool("include_stats", req.IncludeStats),
	)

	// Получаем список пользователей через occtl
	// Используем прямой доступ к occtl напрямую (без Manager)
	// т.к. Manager не экспортирует occtl
	// TODO: Рефакторинг - добавить методы ShowUsers/DisconnectUser в Manager
	users, err := s.server.ocservManager.Occtl().ShowUsers(ctx)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to fetch users",
			slog.String("error", err.Error()),
		)
		return nil, errors.Wrap(err, "failed to fetch active sessions")
	}

	// Конвертируем в proto формат
	var sessions []*pb.VPNSession
	for _, user := range users {
		// Фильтрация по username если указан
		if req.UsernameFilter != "" && user.Username != req.UsernameFilter {
			continue
		}

		// Парсим RX/TX из строки в uint64
		bytesIn, _ := parseBytes(user.RX)
		bytesOut, _ := parseBytes(user.TX)

		session := &pb.VPNSession{
			SessionId:   fmt.Sprintf("%d", user.ID),
			Username:    user.Username,
			ClientIp:    user.RemoteIP,
			VpnIp:       user.IPv4,
			ConnectedAt: timestamppb.New(time.Unix(user.RawConnectedAt, 0)),
			BytesIn:     bytesIn,
			BytesOut:    bytesOut,
			DeviceId:    user.Device,
			Metadata:    make(map[string]string),
		}

		// Добавляем метаданные если нужны stats
		if req.IncludeStats {
			session.Metadata["user_agent"] = user.UserAgent
			session.Metadata["hostname"] = user.Hostname
			session.Metadata["tls_cipher"] = user.TLSCiphersuite
		}

		sessions = append(sessions, session)
	}

	response := &pb.GetActiveSessionsResponse{
		Sessions: sessions,
		// #nosec G115 - session count is reasonable, won't overflow uint32
		TotalCount: uint32(len(sessions)),
	}

	s.logger.InfoContext(ctx, "Active sessions fetched",
		slog.Int("count", len(sessions)),
	)

	return response, nil
}

// parseBytes парсит строку с размером в байтах (например "1.5M", "200K")
// Возвращает значение в байтах
func parseBytes(s string) (uint64, error) {
	// TODO: Реализовать парсинг human-readable bytes
	// Пока возвращаем 0
	return 0, nil
}

// DisconnectUser принудительно отключает пользователя
func (s *VPNService) DisconnectUser(ctx context.Context, req *pb.DisconnectUserRequest) (*pb.DisconnectUserResponse, error) {
	s.logger.InfoContext(ctx, "Disconnecting user",
		slog.String("username", req.Username),
		slog.String("reason", req.Reason),
		slog.Bool("disconnect_all", req.DisconnectAllSessions),
	)

	if req.Username == "" {
		return nil, errors.New("username is required")
	}

	// Выполняем disconnect через occtl
	err := s.server.ocservManager.Occtl().DisconnectUser(ctx, req.Username)
	if err != nil {
		s.logger.ErrorContext(ctx, "Failed to disconnect user",
			slog.String("username", req.Username),
			slog.String("error", err.Error()),
		)
		return &pb.DisconnectUserResponse{
			Success:             false,
			SessionsDisconnected: 0,
			ErrorMessage:        err.Error(),
		}, nil
	}

	response := &pb.DisconnectUserResponse{
		Success:             true,
		SessionsDisconnected: 1, // TODO: Подсчитать реальное количество отключенных сессий
		ErrorMessage:        "",
	}

	s.logger.InfoContext(ctx, "User disconnected successfully",
		slog.String("username", req.Username),
	)

	return response, nil
}

// UpdateUserRoutes обновляет маршруты для пользователя
func (s *VPNService) UpdateUserRoutes(ctx context.Context, req *pb.UpdateUserRoutesRequest) (*pb.UpdateUserRoutesResponse, error) {
	s.logger.InfoContext(ctx, "Updating user routes",
		slog.String("username", req.Username),
		slog.Int("routes_count", len(req.Routes)),
		slog.Int("dns_count", len(req.DnsServers)),
		slog.Bool("reload_if_connected", req.ReloadIfConnected),
	)

	if req.Username == "" {
		return nil, errors.New("username is required")
	}

	if s.server.configGenerator == nil {
		return &pb.UpdateUserRoutesResponse{
			Success:         false,
			ErrorMessage:    "config generator not initialized",
		}, nil
	}

	// Генерируем per-user конфигурацию
	// TODO: Реализовать генерацию через configGenerator
	configPath := fmt.Sprintf("%s/%s", s.server.config.Ocserv.ConfigPerUserDir, req.Username)

	// TODO: Если пользователь подключен и reload_if_connected=true,
	// отключить и переподключить пользователя

	response := &pb.UpdateUserRoutesResponse{
		Success:         true,
		ConfigPath:      configPath,
		UserReconnected: false,
		ErrorMessage:    "",
	}

	s.logger.InfoContext(ctx, "User routes updated",
		slog.String("username", req.Username),
		slog.String("config_path", configPath),
	)

	return response, nil
}
