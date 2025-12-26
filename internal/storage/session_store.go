package storage

import (
	"context"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
)

// VPNSession представляет активную VPN сессию
type VPNSession struct {
	SessionID    string            // Уникальный ID сессии
	Username     string            // Имя пользователя
	ClientIP     string            // IP адрес клиента
	VpnIP        string            // VPN IP адрес
	DeviceID     string            // ID устройства
	ConnectedAt  time.Time         // Время подключения
	LastActivity time.Time         // Время последней активности
	BytesIn      uint64            // Принято байт
	BytesOut     uint64            // Отправлено байт
	Metadata     map[string]string // Дополнительные метаданные
	ExpiresAt    *time.Time        // Время истечения (для TTL)
}

// SessionStore хранит активные VPN сессии в памяти
type SessionStore struct {
	sessions map[string]*VPNSession // sessionID -> session
	mu       sync.RWMutex           // Mutex для thread-safe доступа
	ttl      time.Duration          // TTL для сессий (0 = без TTL)
}

// NewSessionStore создает новый in-memory session store
func NewSessionStore(ttl time.Duration) *SessionStore {
	store := &SessionStore{
		sessions: make(map[string]*VPNSession),
		ttl:      ttl,
	}

	// Запустить cleanup goroutine если TTL установлен
	if ttl > 0 {
		go store.cleanupExpiredSessions(context.Background())
	}

	return store
}

// Add добавляет новую сессию в store
func (s *SessionStore) Add(session *VPNSession) error {
	if session == nil {
		return errors.New("session cannot be nil")
	}

	if session.SessionID == "" {
		return errors.New("session ID cannot be empty")
	}

	if session.Username == "" {
		return errors.New("username cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Установить LastActivity если не установлено
	if session.LastActivity.IsZero() {
		session.LastActivity = time.Now()
	}

	// Установить ConnectedAt если не установлено
	if session.ConnectedAt.IsZero() {
		session.ConnectedAt = time.Now()
	}

	// Установить ExpiresAt если TTL установлен
	if s.ttl > 0 {
		expiresAt := time.Now().Add(s.ttl)
		session.ExpiresAt = &expiresAt
	}

	// Инициализировать Metadata если nil
	if session.Metadata == nil {
		session.Metadata = make(map[string]string)
	}

	s.sessions[session.SessionID] = session

	return nil
}

// Get возвращает сессию по ID
func (s *SessionStore) Get(sessionID string) (*VPNSession, error) {
	if sessionID == "" {
		return nil, errors.New("session ID cannot be empty")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return nil, errors.Newf("session not found: %s", sessionID)
	}

	// Проверить TTL
	if session.ExpiresAt != nil && time.Now().After(*session.ExpiresAt) {
		return nil, errors.Newf("session expired: %s", sessionID)
	}

	return session, nil
}

// Update обновляет существующую сессию
func (s *SessionStore) Update(sessionID string, updateFn func(*VPNSession) error) error {
	if sessionID == "" {
		return errors.New("session ID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.sessions[sessionID]
	if !exists {
		return errors.Newf("session not found: %s", sessionID)
	}

	// Выполнить update функцию
	if err := updateFn(session); err != nil {
		return errors.Wrap(err, "update function failed")
	}

	// Обновить LastActivity
	session.LastActivity = time.Now()

	// Обновить ExpiresAt если TTL установлен
	if s.ttl > 0 {
		expiresAt := time.Now().Add(s.ttl)
		session.ExpiresAt = &expiresAt
	}

	return nil
}

// Remove удаляет сессию по ID
func (s *SessionStore) Remove(sessionID string) error {
	if sessionID == "" {
		return errors.New("session ID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.sessions[sessionID]; !exists {
		return errors.Newf("session not found: %s", sessionID)
	}

	delete(s.sessions, sessionID)

	return nil
}

// List возвращает все активные сессии
func (s *SessionStore) List() []*VPNSession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*VPNSession, 0, len(s.sessions))
	now := time.Now()

	for _, session := range s.sessions {
		// Пропустить истёкшие сессии
		if session.ExpiresAt != nil && now.After(*session.ExpiresAt) {
			continue
		}
		sessions = append(sessions, session)
	}

	return sessions
}

// ListByUsername возвращает все сессии для указанного пользователя
func (s *SessionStore) ListByUsername(username string) []*VPNSession {
	if username == "" {
		return []*VPNSession{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*VPNSession, 0)
	now := time.Now()

	for _, session := range s.sessions {
		// Пропустить истёкшие сессии
		if session.ExpiresAt != nil && now.After(*session.ExpiresAt) {
			continue
		}

		if session.Username == username {
			sessions = append(sessions, session)
		}
	}

	return sessions
}

// Count возвращает количество активных сессий
func (s *SessionStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	now := time.Now()

	for _, session := range s.sessions {
		// Пропустить истёкшие сессии
		if session.ExpiresAt != nil && now.After(*session.ExpiresAt) {
			continue
		}
		count++
	}

	return count
}

// CountByUsername возвращает количество сессий для пользователя
func (s *SessionStore) CountByUsername(username string) int {
	return len(s.ListByUsername(username))
}

// Clear удаляет все сессии
func (s *SessionStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.sessions = make(map[string]*VPNSession)
}

// RemoveByUsername удаляет все сессии пользователя
func (s *SessionStore) RemoveByUsername(username string) int {
	if username == "" {
		return 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	count := 0
	for sessionID, session := range s.sessions {
		if session.Username == username {
			delete(s.sessions, sessionID)
			count++
		}
	}

	return count
}

// UpdateStats обновляет статистику сессии (bytes in/out)
func (s *SessionStore) UpdateStats(sessionID string, bytesIn, bytesOut uint64) error {
	return s.Update(sessionID, func(session *VPNSession) error {
		session.BytesIn = bytesIn
		session.BytesOut = bytesOut
		return nil
	})
}

// cleanupExpiredSessions периодически удаляет истёкшие сессии
func (s *SessionStore) cleanupExpiredSessions(ctx context.Context) {
	ticker := time.NewTicker(s.ttl / 2) // Cleanup каждые ttl/2
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.removeExpiredSessions()
		}
	}
}

// removeExpiredSessions удаляет истёкшие сессии
func (s *SessionStore) removeExpiredSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for sessionID, session := range s.sessions {
		if session.ExpiresAt != nil && now.After(*session.ExpiresAt) {
			delete(s.sessions, sessionID)
		}
	}
}

// GetStats возвращает статистику по всем сессиям
func (s *SessionStore) GetStats() SessionStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := SessionStats{
		TotalSessions: 0,
		UserSessions:  make(map[string]int),
	}

	now := time.Now()

	for _, session := range s.sessions {
		// Пропустить истёкшие сессии
		if session.ExpiresAt != nil && now.After(*session.ExpiresAt) {
			continue
		}

		stats.TotalSessions++
		stats.TotalBytesIn += session.BytesIn
		stats.TotalBytesOut += session.BytesOut

		// Подсчитать сессии по пользователям
		stats.UserSessions[session.Username]++
	}

	return stats
}

// SessionStats содержит статистику по сессиям
type SessionStats struct {
	TotalSessions  int            // Общее количество сессий
	TotalBytesIn   uint64         // Общее количество принятых байт
	TotalBytesOut  uint64         // Общее количество отправленных байт
	UserSessions   map[string]int // Количество сессий по пользователям
}

// Exists проверяет существование сессии
func (s *SessionStore) Exists(sessionID string) bool {
	_, err := s.Get(sessionID)
	return err == nil
}

// GetOrCreate возвращает существующую сессию или создает новую
func (s *SessionStore) GetOrCreate(session *VPNSession) (*VPNSession, bool, error) {
	if session == nil || session.SessionID == "" {
		return nil, false, errors.New("invalid session")
	}

	// Попытка получить существующую сессию
	existing, err := s.Get(session.SessionID)
	if err == nil {
		return existing, false, nil
	}

	// Создать новую сессию
	if err := s.Add(session); err != nil {
		return nil, false, errors.Wrap(err, "failed to add session")
	}

	return session, true, nil
}
