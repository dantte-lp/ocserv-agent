package ipc

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
)

const (
	// MaxMessageSize defines the maximum allowed message size (1MB)
	MaxMessageSize = 1024 * 1024

	// ProtocolVersion defines the current protocol version
	ProtocolVersion = 1
)

// Protocol implements length-prefixed JSON protocol for IPC communication
type Protocol struct{}

// NewProtocol creates a new protocol handler
func NewProtocol() *Protocol {
	return &Protocol{}
}

// ReadMessage reads a length-prefixed JSON message from the connection
func (p *Protocol) ReadMessage(conn net.Conn, v interface{}) error {
	// Read message length (4 bytes, big-endian)
	var msgLen uint32
	if err := binary.Read(conn, binary.BigEndian, &msgLen); err != nil {
		if err == io.EOF {
			return io.EOF
		}
		return fmt.Errorf("read message length: %w", err)
	}

	// Validate message size
	if msgLen == 0 {
		return fmt.Errorf("invalid message length: 0")
	}
	if msgLen > MaxMessageSize {
		return fmt.Errorf("message too large: %d bytes (max: %d)", msgLen, MaxMessageSize)
	}

	// Read message data
	data := make([]byte, msgLen)
	if _, err := io.ReadFull(conn, data); err != nil {
		return fmt.Errorf("read message data: %w", err)
	}

	// Unmarshal JSON
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("unmarshal message: %w", err)
	}

	return nil
}

// WriteMessage writes a length-prefixed JSON message to the connection
func (p *Protocol) WriteMessage(conn net.Conn, v interface{}) error {
	// Marshal to JSON
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	// Validate size
	if len(data) > MaxMessageSize {
		return fmt.Errorf("message too large: %d bytes (max: %d)", len(data), MaxMessageSize)
	}

	// Write length prefix
	// #nosec G115 - len(data) validated against MaxMessageSize above
	if err := binary.Write(conn, binary.BigEndian, uint32(len(data))); err != nil {
		return fmt.Errorf("write message length: %w", err)
	}

	// Write data
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("write message data: %w", err)
	}

	return nil
}
