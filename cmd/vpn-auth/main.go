// vpn-auth - минималистичный CLI для connect-script ocserv
// Используется как connect/disconnect скрипт для быстрой авторизации через IPC
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"
)

const (
	// socketPath - путь к Unix socket агента
	socketPath = "/var/run/ocserv-agent.sock"

	// socketTimeout - максимальное время ожидания ответа от агента
	// Должно быть меньше таймаута connect-script ocserv (~5-7 секунд)
	socketTimeout = 3 * time.Second
)

// AuthRequest - запрос авторизации к агенту
type AuthRequest struct {
	Reason    string `json:"reason"`     // connect, disconnect, host-update
	Username  string `json:"username"`   // Из CN сертификата
	GroupName string `json:"groupname"`  // Из OU сертификата
	IPReal    string `json:"ip_real"`    // IP клиента
	IPRemote  string `json:"ip_remote"`  // VPN IP
	Device    string `json:"device"`     // tun/tap устройство
	SessionID string `json:"session_id"` // ID сессии ocserv
}

// AuthResponse - ответ от агента
type AuthResponse struct {
	Allowed bool   `json:"allowed"` // Разрешено ли подключение
	Error   string `json:"error,omitempty"`
}

func main() {
	// Чтение переменных окружения, установленных ocserv
	req := AuthRequest{
		Reason:    os.Getenv("REASON"),
		Username:  os.Getenv("USERNAME"),
		GroupName: os.Getenv("GROUPNAME"),
		IPReal:    os.Getenv("IP_REAL"),
		IPRemote:  os.Getenv("IP_REMOTE"),
		Device:    os.Getenv("DEVICE"),
		SessionID: os.Getenv("ID"),
	}

	// Валидация обязательных полей
	if req.Username == "" {
		fmt.Fprintf(os.Stderr, "ERROR: USERNAME not set\n")
		os.Exit(1)
	}

	// Только для connect нужна авторизация
	// Для disconnect просто логируем
	if req.Reason != "connect" {
		fmt.Fprintf(os.Stderr, "INFO: reason=%s user=%s - skipping authorization\n",
			req.Reason, req.Username)
		os.Exit(0)
	}

	// Отправка запроса к агенту
	resp, err := sendRequest(&req)
	if err != nil {
		// Fail-open режим: если агент недоступен, разрешаем подключение
		// Для production можно изменить на fail-close (os.Exit(1))
		fmt.Fprintf(os.Stderr, "WARN: agent unavailable: %v - allowing (fail-open mode)\n", err)
		os.Exit(0)
	}

	// Проверка решения агента
	if !resp.Allowed {
		fmt.Fprintf(os.Stderr, "DENY: user=%s reason=%s\n", req.Username, resp.Error)
		os.Exit(1)
	}

	// Разрешено
	fmt.Fprintf(os.Stderr, "ALLOW: user=%s ip=%s\n", req.Username, req.IPReal)
	os.Exit(0)
}

// sendRequest отправляет запрос к агенту через Unix socket
// Использует length-prefixed JSON протокол
func sendRequest(req *AuthRequest) (*AuthResponse, error) {
	// Подключение к Unix socket
	conn, err := net.DialTimeout("unix", socketPath, socketTimeout)
	if err != nil {
		return nil, fmt.Errorf("dial socket: %w", err)
	}
	defer conn.Close()

	// Установка deadline для всей операции
	if err := conn.SetDeadline(time.Now().Add(socketTimeout)); err != nil {
		return nil, fmt.Errorf("set deadline: %w", err)
	}

	// Сериализация запроса
	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Отправка: length (4 bytes big-endian) + JSON payload
	if err := binary.Write(conn, binary.BigEndian, uint32(len(data))); err != nil {
		return nil, fmt.Errorf("write length: %w", err)
	}

	if _, err := conn.Write(data); err != nil {
		return nil, fmt.Errorf("write data: %w", err)
	}

	// Чтение ответа: length (4 bytes) + JSON
	var respLen uint32
	if err := binary.Read(conn, binary.BigEndian, &respLen); err != nil {
		return nil, fmt.Errorf("read response length: %w", err)
	}

	// Защита от слишком больших ответов
	const maxResponseSize = 1024 * 1024 // 1MB
	if respLen > maxResponseSize {
		return nil, fmt.Errorf("response too large: %d bytes", respLen)
	}

	// Чтение данных ответа
	respData := make([]byte, respLen)
	if _, err := conn.Read(respData); err != nil {
		return nil, fmt.Errorf("read response data: %w", err)
	}

	// Десериализация ответа
	var resp AuthResponse
	if err := json.Unmarshal(respData, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &resp, nil
}
