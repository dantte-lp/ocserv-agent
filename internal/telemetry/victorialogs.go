package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/dantte-lp/ocserv-agent/internal/config"
)

// VictoriaLogsHandler реализует slog.Handler для отправки логов в VictoriaLogs.
// Использует JSON Lines (NDJSON) формат через HTTP API.
type VictoriaLogsHandler struct {
	config config.VictoriaLogsConfig
	client *http.Client
	buffer []logEntry
	mu     sync.Mutex
	ticker *time.Ticker
	done   chan struct{}
	wg     sync.WaitGroup
	inner  slog.Handler // Fallback handler для локальных логов
	attrs  []slog.Attr  // Accumulated attributes
	groups []string     // Group path
}

// logEntry представляет одну запись лога для VictoriaLogs.
type logEntry struct {
	Time    string                 `json:"_time"`
	Message string                 `json:"_msg"`
	Level   string                 `json:"level"`
	Fields  map[string]interface{} `json:"-"`
}

// MarshalJSON кастомная сериализация для включения динамических полей.
func (e logEntry) MarshalJSON() ([]byte, error) {
	// Создаем временный map со всеми полями
	m := make(map[string]interface{})
	m["_time"] = e.Time
	m["_msg"] = e.Message
	m["level"] = e.Level

	// Добавляем все динамические поля
	for k, v := range e.Fields {
		m[k] = v
	}

	data, err := json.Marshal(m)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal log entry")
	}
	return data, nil
}

// NewVictoriaLogsHandler создает новый handler для VictoriaLogs.
func NewVictoriaLogsHandler(cfg config.VictoriaLogsConfig, inner slog.Handler) *VictoriaLogsHandler {
	h := &VictoriaLogsHandler{
		config: cfg,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		buffer: make([]logEntry, 0, cfg.BatchSize),
		ticker: time.NewTicker(cfg.FlushInterval),
		done:   make(chan struct{}),
		inner:  inner,
		attrs:  make([]slog.Attr, 0),
		groups: make([]string, 0),
	}

	// Запускаем фоновый процесс периодического flush
	h.wg.Add(1)
	go h.flushLoop()

	return h
}

// Enabled проверяет, должен ли логироваться данный уровень.
func (h *VictoriaLogsHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle обрабатывает запись лога.
func (h *VictoriaLogsHandler) Handle(ctx context.Context, r slog.Record) error {
	// Сначала пишем в fallback handler
	if err := h.inner.Handle(ctx, r); err != nil {
		return errors.Wrap(err, "fallback handler failed")
	}

	// Если VictoriaLogs не включен, пропускаем
	if !h.config.Enabled {
		return nil
	}

	// Создаем запись лога
	entry := logEntry{
		Time:    r.Time.UTC().Format(time.RFC3339Nano),
		Message: r.Message,
		Level:   r.Level.String(),
		Fields:  make(map[string]interface{}),
	}

	// Добавляем глобальные метки из конфигурации
	for k, v := range h.config.Labels {
		entry.Fields[k] = v
	}

	// Добавляем accumulated attributes
	for _, attr := range h.attrs {
		addAttrToFields(entry.Fields, h.groups, attr)
	}

	// Добавляем атрибуты из записи
	r.Attrs(func(a slog.Attr) bool {
		addAttrToFields(entry.Fields, h.groups, a)
		return true
	})

	// Добавляем в буфер
	h.mu.Lock()
	h.buffer = append(h.buffer, entry)
	shouldFlush := len(h.buffer) >= h.config.BatchSize
	h.mu.Unlock()

	// Flush если буфер заполнен
	if shouldFlush {
		return h.flush()
	}

	return nil
}

// WithAttrs возвращает новый handler с дополнительными атрибутами.
func (h *VictoriaLogsHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &VictoriaLogsHandler{
		config: h.config,
		client: h.client,
		buffer: h.buffer,
		ticker: h.ticker,
		done:   h.done,
		inner:  h.inner.WithAttrs(attrs),
		attrs:  newAttrs,
		groups: h.groups,
	}
}

// WithGroup возвращает новый handler с группой.
func (h *VictoriaLogsHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}

	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name

	return &VictoriaLogsHandler{
		config: h.config,
		client: h.client,
		buffer: h.buffer,
		ticker: h.ticker,
		done:   h.done,
		inner:  h.inner.WithGroup(name),
		attrs:  h.attrs,
		groups: newGroups,
	}
}

// flush отправляет буфер в VictoriaLogs.
func (h *VictoriaLogsHandler) flush() error {
	h.mu.Lock()
	if len(h.buffer) == 0 {
		h.mu.Unlock()
		return nil
	}

	// Копируем буфер для отправки
	entries := make([]logEntry, len(h.buffer))
	copy(entries, h.buffer)
	h.buffer = h.buffer[:0] // Очищаем буфер
	h.mu.Unlock()

	// Сериализуем в NDJSON
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	for _, entry := range entries {
		if err := encoder.Encode(entry); err != nil {
			return fmt.Errorf("failed to encode log entry: %w", err)
		}
	}

	// Создаем HTTP запрос
	req, err := http.NewRequest(http.MethodPost, h.config.Endpoint, &buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-ndjson")

	// Добавляем Basic Auth если настроено
	if h.config.Username != "" && h.config.Password != "" {
		req.SetBasicAuth(h.config.Username, h.config.Password)
	}

	// Отправляем запрос
	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send logs to VictoriaLogs: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close response body", "error", err)
		}
	}()

	// Проверяем статус ответа
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("VictoriaLogs returned error status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// flushLoop периодически отправляет логи в VictoriaLogs.
func (h *VictoriaLogsHandler) flushLoop() {
	defer h.wg.Done()

	for {
		select {
		case <-h.ticker.C:
			if err := h.flush(); err != nil {
				// Используем fallback handler для ошибок отправки
				if handleErr := h.inner.Handle(context.Background(), slog.NewRecord(
					time.Now(),
					slog.LevelError,
					"Failed to flush logs to VictoriaLogs",
					0,
				)); handleErr != nil {
					// Нет возможности логировать ошибку логирования
					_ = handleErr
				}
			}
		case <-h.done:
			// Финальный flush перед закрытием
			_ = h.flush()
			return
		}
	}
}

// Close корректно завершает работу handler.
func (h *VictoriaLogsHandler) Close() error {
	close(h.done)
	h.ticker.Stop()
	h.wg.Wait()
	return nil
}

// addAttrToFields добавляет атрибут в map с учетом групп.
func addAttrToFields(fields map[string]interface{}, groups []string, attr slog.Attr) {
	key := attr.Key
	if len(groups) > 0 {
		key = joinGroups(groups) + "." + key
	}

	switch attr.Value.Kind() {
	case slog.KindGroup:
		// Рекурсивно добавляем атрибуты группы
		for _, groupAttr := range attr.Value.Group() {
			addAttrToFields(fields, append(groups, attr.Key), groupAttr)
		}
	default:
		fields[key] = attr.Value.Any()
	}
}

// joinGroups объединяет группы в путь через точку.
func joinGroups(groups []string) string {
	if len(groups) == 0 {
		return ""
	}
	result := groups[0]
	for i := 1; i < len(groups); i++ {
		result += "." + groups[i]
	}
	return result
}
