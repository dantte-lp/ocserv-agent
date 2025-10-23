// mock-ocserv: Mock ocserv Unix socket server for integration testing
//
// This server simulates occtl Unix socket interface by:
// - Listening on a Unix socket (default: /tmp/occtl-test.socket)
// - Accepting JSON-formatted occtl commands
// - Returning realistic responses from test fixtures
//
// Usage:
//
//	go run main.go -socket /tmp/occtl-test.socket
//	go run main.go -fixtures ./test/fixtures/ocserv/occtl
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

const (
	defaultSocketPath   = "/tmp/occtl-test.socket"
	defaultFixturesPath = "../fixtures/ocserv/occtl"
	readTimeout         = 5 * time.Second
	writeTimeout        = 5 * time.Second
)

// Config holds mock server configuration
type Config struct {
	SocketPath   string
	FixturesPath string
	Verbose      bool
}

func main() {
	config := parseFlags()

	log.Printf("🚀 Starting mock-ocserv server")
	log.Printf("Socket: %s", config.SocketPath)
	log.Printf("Fixtures: %s", config.FixturesPath)

	// Load fixtures
	fixtures, err := LoadFixtures(config.FixturesPath)
	if err != nil {
		log.Fatalf("Failed to load fixtures: %v", err)
	}
	log.Printf("✅ Loaded %d fixtures", fixtures.Len())

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Remove old socket if exists
	if err := os.RemoveAll(config.SocketPath); err != nil {
		log.Fatalf("Failed to remove old socket: %v", err)
	}

	// Create Unix socket listener
	listener, err := net.Listen("unix", config.SocketPath)
	if err != nil {
		log.Fatalf("Failed to create socket: %v", err)
	}
	defer listener.Close()

	// Set socket permissions (0666 like real ocserv)
	if err := os.Chmod(config.SocketPath, 0666); err != nil {
		log.Printf("Warning: Failed to set socket permissions: %v", err)
	}

	log.Printf("✅ Listening on %s", config.SocketPath)
	log.Printf("Press Ctrl+C to stop")

	// Start accepting connections
	errChan := make(chan error, 1)
	go func() {
		handler := NewSocketHandler(fixtures, config.Verbose)
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					errChan <- fmt.Errorf("accept error: %w", err)
					return
				}
			}
			go handler.HandleConnection(ctx, conn)
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		log.Println("\n🛑 Received shutdown signal")
	case err := <-errChan:
		log.Printf("❌ Server error: %v", err)
	}

	// Cleanup
	cancel()
	listener.Close()
	os.Remove(config.SocketPath)
	log.Println("✅ Server stopped")
}

func parseFlags() *Config {
	config := &Config{}

	flag.StringVar(&config.SocketPath, "socket", defaultSocketPath,
		"Unix socket path")
	flag.StringVar(&config.FixturesPath, "fixtures", defaultFixturesPath,
		"Path to fixtures directory")
	flag.BoolVar(&config.Verbose, "verbose", false,
		"Enable verbose logging")

	flag.Parse()

	// Resolve relative paths
	if !filepath.IsAbs(config.FixturesPath) {
		if abs, err := filepath.Abs(config.FixturesPath); err == nil {
			config.FixturesPath = abs
		}
	}

	return config
}
