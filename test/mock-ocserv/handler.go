package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

// SocketHandler handles Unix socket connections
type SocketHandler struct {
	fixtures *Fixtures
	verbose  bool
}

// NewSocketHandler creates a new socket handler
func NewSocketHandler(fixtures *Fixtures, verbose bool) *SocketHandler {
	return &SocketHandler{
		fixtures: fixtures,
		verbose:  verbose,
	}
}

// HandleConnection handles a single client connection
func (h *SocketHandler) HandleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	if h.verbose {
		log.Printf("📥 New connection from %s", conn.RemoteAddr())
	}

	// Set read/write timeouts
	conn.SetReadDeadline(time.Now().Add(readTimeout))
	conn.SetWriteDeadline(time.Now().Add(writeTimeout))

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse command
		cmd, err := ParseCommand(line)
		if err != nil {
			if h.verbose {
				log.Printf("❌ Parse error: %v (input: %q)", err, line)
			}
			h.writeError(conn, fmt.Sprintf("parse error: %v", err))
			continue
		}

		if h.verbose {
			log.Printf("📨 Command: %s", cmd.String())
		}

		// Execute command
		response, err := h.executeCommand(cmd)
		if err != nil {
			if h.verbose {
				log.Printf("❌ Execute error: %v", err)
			}
			h.writeError(conn, fmt.Sprintf("execution error: %v", err))
			continue
		}

		// Send response
		if _, err := conn.Write([]byte(response + "\n")); err != nil {
			log.Printf("❌ Write error: %v", err)
			return
		}

		if h.verbose {
			log.Printf("📤 Response: %d bytes", len(response))
		}

		// Reset timeout for next read
		conn.SetReadDeadline(time.Now().Add(readTimeout))
	}

	if err := scanner.Err(); err != nil {
		log.Printf("❌ Scanner error: %v", err)
	}
}

// executeCommand executes a parsed command and returns response
func (h *SocketHandler) executeCommand(cmd *Command) (string, error) {
	// Build fixture key from command
	key := h.buildFixtureKey(cmd)

	// Check if we have a fixture for this command
	if response, ok := h.fixtures.Get(key); ok {
		return response, nil
	}

	// Try alternative keys
	altKeys := h.buildAlternativeKeys(cmd)
	for _, altKey := range altKeys {
		if response, ok := h.fixtures.Get(altKey); ok {
			if h.verbose {
				log.Printf("Using alternative key: %s", altKey)
			}
			return response, nil
		}
	}

	// No fixture found
	return "", fmt.Errorf("no fixture for command: %s", key)
}

// buildFixtureKey builds fixture filename from command
// Examples:
//   - "show -j users" -> "occtl -j show users"
//   - "show -j user lpa" -> "occtl -j show user"
//   - "show status" -> "occtl show status"
func (h *SocketHandler) buildFixtureKey(cmd *Command) string {
	parts := []string{"occtl"}

	if cmd.JSON {
		parts = append(parts, "-j")
	}

	parts = append(parts, cmd.Command...)

	// For commands with arguments (user, id, session), drop the argument
	if len(cmd.Arguments) > 0 {
		switch cmd.Command[0] {
		case "show":
			if len(cmd.Command) >= 2 {
				switch cmd.Command[1] {
				case "user", "id", "session":
					// Keep command without argument
					// "show user lpa" -> "occtl -j show user"
				default:
					// Keep arguments for other commands
					parts = append(parts, cmd.Arguments...)
				}
			}
		default:
			parts = append(parts, cmd.Arguments...)
		}
	}

	return strings.Join(parts, " ")
}

// buildAlternativeKeys generates alternative fixture keys
func (h *SocketHandler) buildAlternativeKeys(cmd *Command) []string {
	var keys []string

	// Try without -j flag
	if cmd.JSON {
		altCmd := *cmd
		altCmd.JSON = false
		keys = append(keys, h.buildFixtureKey(&altCmd))
	}

	// Try with -j flag if not present
	if !cmd.JSON {
		altCmd := *cmd
		altCmd.JSON = true
		keys = append(keys, h.buildFixtureKey(&altCmd))
	}

	return keys
}

// writeError writes JSON error response
func (h *SocketHandler) writeError(conn net.Conn, message string) {
	errorResp := map[string]string{
		"error": message,
	}
	data, _ := json.Marshal(errorResp)
	conn.Write(append(data, '\n'))
}
