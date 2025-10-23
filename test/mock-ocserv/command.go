package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Command represents a parsed occtl command
type Command struct {
	JSON      bool     // -j flag
	Command   []string // Main command parts (e.g., ["show", "users"])
	Arguments []string // Command arguments (e.g., username, session ID)
}

// String returns human-readable command representation
func (c *Command) String() string {
	parts := []string{}
	if c.JSON {
		parts = append(parts, "-j")
	}
	parts = append(parts, c.Command...)
	if len(c.Arguments) > 0 {
		parts = append(parts, fmt.Sprintf("[%s]", strings.Join(c.Arguments, ", ")))
	}
	return strings.Join(parts, " ")
}

// CommandRequest represents JSON request from ocserv
type CommandRequest struct {
	Command []string `json:"command"`
}

// ParseCommand parses occtl command from JSON or plain text
//
// Supports two formats:
// 1. JSON format (from ocserv socket): {"command": ["show", "-j", "users"]}
// 2. Plain text format (for testing): "show -j users"
func ParseCommand(input string) (*Command, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("empty command")
	}

	// Try JSON format first
	if strings.HasPrefix(input, "{") {
		var req CommandRequest
		if err := json.Unmarshal([]byte(input), &req); err == nil {
			return parseCommandParts(req.Command)
		}
	}

	// Fall back to plain text format
	parts := strings.Fields(input)
	return parseCommandParts(parts)
}

// parseCommandParts parses command parts into Command struct
func parseCommandParts(parts []string) (*Command, error) {
	if len(parts) == 0 {
		return nil, fmt.Errorf("no command parts")
	}

	cmd := &Command{
		JSON:      false,
		Command:   []string{},
		Arguments: []string{},
	}

	// Parse flags and command
	commandStarted := false
	for _, part := range parts {
		if part == "-j" {
			cmd.JSON = true
			continue
		}

		// First non-flag part starts the command
		if !commandStarted {
			cmd.Command = append(cmd.Command, part)
			commandStarted = true
			continue
		}

		// Determine if this is part of command or an argument
		if isCommandKeyword(part) {
			cmd.Command = append(cmd.Command, part)
		} else {
			// This is an argument (username, session ID, etc.)
			cmd.Arguments = append(cmd.Arguments, part)
			// All remaining parts are arguments
			break
		}
	}

	if len(cmd.Command) == 0 {
		return nil, fmt.Errorf("no command specified")
	}

	return cmd, nil
}

// isCommandKeyword checks if a word is part of command structure
func isCommandKeyword(word string) bool {
	keywords := []string{
		// Primary commands
		"show", "disconnect", "unban", "reload", "stop",

		// Show subcommands
		"status", "users", "user", "id", "sessions", "session",
		"iroutes", "events", "bans", "cookies",

		// Show modifiers
		"all", "valid", "ip", "ban", "points",

		// Other
		"now",
	}

	for _, kw := range keywords {
		if word == kw {
			return true
		}
	}

	return false
}
