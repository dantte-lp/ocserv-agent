package grpc

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/dantte-lp/ocserv-agent/internal/config"
	"github.com/dantte-lp/ocserv-agent/internal/storage"
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

	// Создать сессию в SessionStore
	if s.server.sessionStore != nil {
		session := &storage.VPNSession{
			SessionID:   req.SessionId,
			Username:    req.Username,
			ClientIP:    req.ClientIp,
			VpnIP:       req.VpnIp,
			DeviceID:    req.DeviceId,
			ConnectedAt: time.Now(),
			Metadata:    make(map[string]string),
		}

		// Копировать metadata из запроса
		if req.Metadata != nil {
			for k, v := range req.Metadata {
				session.Metadata[k] = v
			}
		}

		if err := s.server.sessionStore.Add(session); err != nil {
			s.logger.ErrorContext(ctx, "Failed to add session to store",
				slog.String("session_id", req.SessionId),
				slog.String("error", err.Error()),
			)
			// Не блокируем подключение из-за ошибки в SessionStore
		} else {
			s.logger.InfoContext(ctx, "Session added to store",
				slog.String("session_id", req.SessionId),
				slog.Int("total_sessions", s.server.sessionStore.Count()),
			)
		}
	}

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

	// Обновить статистику и удалить сессию из SessionStore
	if s.server.sessionStore != nil {
		// Сначала обновляем статистику если сессия существует
		if err := s.server.sessionStore.UpdateStats(req.SessionId, req.BytesIn, req.BytesOut); err != nil {
			s.logger.WarnContext(ctx, "Failed to update session stats",
				slog.String("session_id", req.SessionId),
				slog.String("error", err.Error()),
			)
		}

		// Удаляем сессию
		if err := s.server.sessionStore.Remove(req.SessionId); err != nil {
			s.logger.ErrorContext(ctx, "Failed to remove session from store",
				slog.String("session_id", req.SessionId),
				slog.String("error", err.Error()),
			)
		} else {
			s.logger.InfoContext(ctx, "Session removed from store",
				slog.String("session_id", req.SessionId),
				slog.Int("remaining_sessions", s.server.sessionStore.Count()),
			)
		}
	}

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

	var sessions []*pb.VPNSession

	// Если SessionStore включен, используем данные из него
	if s.server.sessionStore != nil {
		var storedSessions []*storage.VPNSession

		if req.UsernameFilter != "" {
			storedSessions = s.server.sessionStore.ListByUsername(req.UsernameFilter)
		} else {
			storedSessions = s.server.sessionStore.List()
		}

		// Конвертируем в proto формат
		for _, session := range storedSessions {
			pbSession := &pb.VPNSession{
				SessionId:   session.SessionID,
				Username:    session.Username,
				ClientIp:    session.ClientIP,
				VpnIp:       session.VpnIP,
				ConnectedAt: timestamppb.New(session.ConnectedAt),
				BytesIn:     session.BytesIn,
				BytesOut:    session.BytesOut,
				DeviceId:    session.DeviceID,
				Metadata:    make(map[string]string),
			}

			// Добавляем метаданные если нужны stats
			if req.IncludeStats && session.Metadata != nil {
				for k, v := range session.Metadata {
					pbSession.Metadata[k] = v
				}
			}

			sessions = append(sessions, pbSession)
		}

		s.logger.InfoContext(ctx, "Active sessions fetched from SessionStore",
			slog.Int("count", len(sessions)),
		)
	} else {
		// Fallback: получаем список пользователей через occtl
		users, err := s.server.ocservManager.Occtl().ShowUsers(ctx)
		if err != nil {
			s.logger.ErrorContext(ctx, "Failed to fetch users",
				slog.String("error", err.Error()),
			)
			return nil, errors.Wrap(err, "failed to fetch active sessions")
		}

		// Конвертируем в proto формат
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

		s.logger.InfoContext(ctx, "Active sessions fetched from occtl",
			slog.Int("count", len(sessions)),
		)
	}

	response := &pb.GetActiveSessionsResponse{
		Sessions: sessions,
		// #nosec G115 - session count is reasonable, won't overflow uint32
		TotalCount: uint32(len(sessions)),
	}

	return response, nil
}

// parseBytes парсит строку с размером в байтах (например "1.5M", "200K", "3.2G")
// Возвращает значение в байтах
func parseBytes(s string) (uint64, error) {
	if s == "" || s == "0" || s == "-" {
		return 0, nil
	}

	// Удаляем пробелы
	s = strings.TrimSpace(s)

	// Найти позицию первого не-цифрового символа (кроме точки)
	var value float64
	var unit string

	// Разделить на число и единицу измерения
	i := 0
	for i < len(s) && (s[i] >= '0' && s[i] <= '9' || s[i] == '.') {
		i++
	}

	if i == 0 {
		return 0, errors.Newf("invalid byte string: %s", s)
	}

	// Парсинг числового значения
	valueStr := s[:i]
	var err error
	value, err = strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to parse value: %s", valueStr)
	}

	// Парсинг единицы измерения
	unit = strings.ToUpper(strings.TrimSpace(s[i:]))

	// Конвертация в байты
	var multiplier uint64
	switch unit {
	case "", "B":
		multiplier = 1
	case "K", "KB":
		multiplier = 1024
	case "M", "MB":
		multiplier = 1024 * 1024
	case "G", "GB":
		multiplier = 1024 * 1024 * 1024
	case "T", "TB":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, errors.Newf("unknown unit: %s", unit)
	}

	// #nosec G115 - value is parsed from occtl output, bounded by network limits
	result := uint64(value * float64(multiplier))
	return result, nil
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

	// Создаем конфигурацию для пользователя
	userConfig := &config.PerUserConfig{
		Username:         req.Username,
		Routes:           req.Routes,
		DNS:              req.DnsServers,
		CustomDirectives: make(map[string]string),
	}

	// Добавляем custom директивы из запроса
	if req.ConfigParams != nil {
		for k, v := range req.ConfigParams {
			userConfig.CustomDirectives[k] = v
		}
	}

	// Генерируем конфигурационный файл
	if err := s.server.configGenerator.GenerateUserConfig(userConfig); err != nil {
		s.logger.ErrorContext(ctx, "Failed to generate user config",
			slog.String("username", req.Username),
			slog.String("error", err.Error()),
		)
		return &pb.UpdateUserRoutesResponse{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to generate config: %v", err),
		}, nil
	}

	configPath := fmt.Sprintf("%s/%s", s.server.config.Ocserv.ConfigPerUserDir, req.Username)

	// Проверяем, подключен ли пользователь
	userReconnected := false
	if req.ReloadIfConnected && s.server.sessionStore != nil {
		userSessions := s.server.sessionStore.ListByUsername(req.Username)
		if len(userSessions) > 0 {
			// Отключаем пользователя для применения новой конфигурации
			if err := s.server.ocservManager.Occtl().DisconnectUser(ctx, req.Username); err != nil {
				s.logger.WarnContext(ctx, "Failed to disconnect user for reconnect",
					slog.String("username", req.Username),
					slog.String("error", err.Error()),
				)
			} else {
				userReconnected = true
				// Удаляем сессии из SessionStore
				s.server.sessionStore.RemoveByUsername(req.Username)
				s.logger.InfoContext(ctx, "User disconnected for config reload",
					slog.String("username", req.Username),
					slog.Int("sessions_removed", len(userSessions)),
				)
			}
		}
	}

	response := &pb.UpdateUserRoutesResponse{
		Success:         true,
		ConfigPath:      configPath,
		UserReconnected: userReconnected,
		ErrorMessage:    "",
	}

	s.logger.InfoContext(ctx, "User routes updated successfully",
		slog.String("username", req.Username),
		slog.String("config_path", configPath),
		slog.Bool("user_reconnected", userReconnected),
	)

	return response, nil
}
