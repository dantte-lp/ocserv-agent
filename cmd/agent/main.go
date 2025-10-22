package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dantte-lp/ocserv-agent/internal/config"
	grpcserver "github.com/dantte-lp/ocserv-agent/internal/grpc"
	"github.com/rs/zerolog"
)

var (
	version = "dev" // Set by ldflags during build
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("ocserv-agent version %s\n", version)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Setup logging
	logger := setupLogger(cfg.Logging)
	logger.Info().
		Str("version", version).
		Str("agent_id", cfg.AgentID).
		Str("hostname", cfg.Hostname).
		Msg("Starting ocserv-agent")

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create gRPC server
	grpcServer, err := grpcserver.New(cfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create gRPC server")
	}

	// Start gRPC server in goroutine
	serverErr := make(chan error, 1)
	go func() {
		address := fmt.Sprintf(":%d", 9090) // TODO: Make port configurable
		if err := grpcServer.Serve(address); err != nil {
			serverErr <- err
		}
	}()

	// Wait for interrupt signal or server error
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)

	select {
	case err := <-serverErr:
		logger.Fatal().Err(err).Msg("gRPC server failed")

	case sig := <-sigCh:
		logger.Info().
			Str("signal", sig.String()).
			Msg("Received shutdown signal")

		// Handle SIGHUP for config reload
		if sig == syscall.SIGHUP {
			logger.Info().Msg("Config reload not yet implemented, ignoring SIGHUP")
			// TODO: Implement config reload
		} else {
			// Graceful shutdown
			logger.Info().Msg("Initiating graceful shutdown...")

			// Create shutdown context with timeout
			shutdownCtx, shutdownCancel := context.WithTimeout(ctx, 30*time.Second)
			defer shutdownCancel()

			// Stop gRPC server gracefully
			stopped := make(chan struct{})
			go func() {
				grpcServer.GracefulStop()
				close(stopped)
			}()

			// Wait for graceful stop or timeout
			select {
			case <-stopped:
				logger.Info().Msg("Server stopped gracefully")
			case <-shutdownCtx.Done():
				logger.Warn().Msg("Graceful shutdown timeout, forcing stop")
				grpcServer.Stop()
			}

			logger.Info().Msg("Shutdown complete")
		}
	}
}

// setupLogger configures zerolog based on config
func setupLogger(cfg config.LoggingConfig) zerolog.Logger {
	// Set log level
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Configure output
	var output *os.File
	if cfg.Output == "file" && cfg.FilePath != "" {
		// TODO: Implement file rotation with lumberjack
		file, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file: %v, falling back to stdout\n", err)
			output = os.Stdout
		} else {
			output = file
		}
	} else {
		output = os.Stdout
	}

	// Create logger
	var logger zerolog.Logger
	if cfg.Format == "json" {
		logger = zerolog.New(output).With().Timestamp().Logger()
	} else {
		// Pretty console output
		logger = zerolog.New(zerolog.ConsoleWriter{Out: output}).With().Timestamp().Logger()
	}

	return logger
}
