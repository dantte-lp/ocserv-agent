package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dantte-lp/ocserv-agent/internal/cert"
	"github.com/dantte-lp/ocserv-agent/internal/config"
	grpcserver "github.com/dantte-lp/ocserv-agent/internal/grpc"
	"github.com/rs/zerolog"
)

var (
	version = "dev" // Set by ldflags during build
)

func main() {
	// Check for subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "gencert":
			runGenCert()
			return
		case "version", "--version", "-v":
			fmt.Printf("ocserv-agent version %s\n", version)
			os.Exit(0)
		case "help", "--help", "-h":
			printUsage()
			os.Exit(0)
		}
	}

	// Parse command line flags for server mode
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version and exit")
	phase2 := flag.Bool("phase2", false, "Run with Phase 2 features (IPC + stats poller)")
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

	// Check if Phase 2 mode is enabled
	if *phase2 {
		// Import необходимо добавить в начало файла
		// Используем slog для Phase 2
		fmt.Printf("🚀 Starting ocserv-agent in Phase 2 mode (IPC + stats poller)\n")

		// Для Phase 2 используем отдельную функцию запуска
		// которая будет импортирована из main_phase2.go
		if err := runServerPhase2(cfg, nil); err != nil {
			fmt.Fprintf(os.Stderr, "Phase 2 server error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Setup logging для Phase 1 (legacy)
	logger := setupLogger(cfg.Logging)

	// Log startup with version and config
	logger.Info().
		Str("version", version).
		Str("agent_id", cfg.AgentID).
		Str("hostname", cfg.Hostname).
		Msg("Starting ocserv-agent")

	// Log loaded configuration for validation
	logger.Info().
		Str("config_file", *configPath).
		Str("log_level", cfg.Logging.Level).
		Str("log_format", cfg.Logging.Format).
		Str("log_output", cfg.Logging.Output).
		Msg("Configuration loaded")

	logger.Debug().
		Str("control_server", cfg.ControlServer.Address).
		Bool("tls_enabled", cfg.TLS.Enabled).
		Bool("tls_auto_generate", cfg.TLS.AutoGenerate).
		Str("tls_min_version", cfg.TLS.MinVersion).
		Str("ocserv_config", cfg.Ocserv.ConfigPath).
		Str("ocserv_service", cfg.Ocserv.SystemdService).
		Dur("heartbeat_interval", cfg.Health.HeartbeatInterval).
		Dur("deep_check_interval", cfg.Health.DeepCheckInterval).
		Msg("Detailed configuration")

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
		// Log to stderr since logger not ready yet
		fmt.Fprintf(os.Stderr, "Warning: invalid log level '%s', using 'info'\n", cfg.Level)
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

// printUsage prints command usage information
func printUsage() {
	fmt.Printf(`ocserv-agent - OpenConnect VPN Server Management Agent

Usage:
  ocserv-agent [flags]                Run the agent server
  ocserv-agent gencert [flags]        Generate certificates
  ocserv-agent version                Show version
  ocserv-agent help                   Show this help

Server Flags:
  -config string
        Path to configuration file (default "config.yaml")
  -version
        Show version and exit

GenCert Flags:
  -output string
        Output directory for certificates (default "/etc/ocserv-agent/certs")
  -hostname string
        Hostname for certificate (default: auto-detect)
  -self-signed
        Generate self-signed certificates (default: true)
  -ca string
        Path to CA certificate for signing (not implemented yet)

Examples:
  # Run agent server with default config
  ocserv-agent

  # Run with custom config
  ocserv-agent -config /etc/ocserv-agent/config.yaml

  # Generate self-signed certificates
  ocserv-agent gencert -output /etc/ocserv-agent/certs

  # Generate with custom hostname
  ocserv-agent gencert -hostname vpn.example.com -output /etc/ocserv-agent/certs

For more information, visit: https://github.com/dantte-lp/ocserv-agent
`)
}

// runGenCert handles the 'gencert' subcommand
func runGenCert() {
	// Create flagset for gencert subcommand
	gencertCmd := flag.NewFlagSet("gencert", flag.ExitOnError)
	outputDir := gencertCmd.String("output", "/etc/ocserv-agent/certs", "Output directory for certificates")
	hostname := gencertCmd.String("hostname", "", "Hostname for certificate (auto-detect if empty)")
	selfSigned := gencertCmd.Bool("self-signed", true, "Generate self-signed certificates")
	caPath := gencertCmd.String("ca", "", "Path to CA certificate (not implemented)")

	// Parse flags
	if err := gencertCmd.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Auto-detect hostname if not provided
	if *hostname == "" {
		h, err := os.Hostname()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to auto-detect hostname: %v\n", err)
			os.Exit(1)
		}
		*hostname = h
	}

	// Check if not self-signed and CA provided
	if !*selfSigned && *caPath != "" {
		fmt.Fprintf(os.Stderr, "Error: CA-signed certificate generation not yet implemented\n")
		fmt.Fprintf(os.Stderr, "Use -self-signed flag to generate self-signed certificates\n")
		os.Exit(1)
	}

	// Generate self-signed certificates
	if *selfSigned {
		fmt.Printf("🔐 Generating self-signed certificates...\n")
		fmt.Printf("   Hostname:        %s\n", *hostname)
		fmt.Printf("   Output dir:      %s\n", *outputDir)
		fmt.Printf("\n")

		info, err := cert.GenerateSelfSignedCerts(*outputDir, *hostname)
		if err != nil {
			fmt.Fprintf(os.Stderr, "❌ Failed to generate certificates: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Certificates generated successfully!\n\n")
		fmt.Printf("Certificate Information:\n")
		fmt.Printf("   CA Fingerprint:   %s\n", info.CAFingerprint)
		fmt.Printf("   Cert Fingerprint: %s\n", info.CertFingerprint)
		fmt.Printf("   Subject:          %s\n", info.Subject)
		fmt.Printf("   Valid From:       %s\n", info.ValidFrom.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("   Valid Until:      %s\n", info.ValidUntil.Format("2006-01-02 15:04:05 MST"))
		fmt.Printf("\n")
		fmt.Printf("Files created:\n")
		fmt.Printf("   %s/ca.crt       - CA certificate\n", *outputDir)
		fmt.Printf("   %s/agent.crt    - Agent certificate\n", *outputDir)
		fmt.Printf("   %s/agent.key    - Agent private key\n", *outputDir)
		fmt.Printf("\n")
		fmt.Printf("⚠️  These are self-signed certificates for autonomous operation.\n")
		fmt.Printf("   To connect to a control server, you'll need CA-signed certificates.\n")
		fmt.Printf("\n")
		fmt.Printf("💡 Tip: Update your config.yaml to use these certificates:\n")
		fmt.Printf("   tls:\n")
		fmt.Printf("     enabled: true\n")
		fmt.Printf("     auto_generate: false  # Disable auto-gen since certs exist\n")
		fmt.Printf("     cert_file: %s/agent.crt\n", *outputDir)
		fmt.Printf("     key_file: %s/agent.key\n", *outputDir)
		fmt.Printf("     ca_file: %s/ca.crt\n", *outputDir)
		fmt.Printf("\n")
	}
}
