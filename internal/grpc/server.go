package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/dantte-lp/ocserv-agent/internal/config"
	"github.com/dantte-lp/ocserv-agent/internal/ocserv"
	"github.com/dantte-lp/ocserv-agent/internal/storage"
	pb "github.com/dantte-lp/ocserv-agent/pkg/proto/agent/v1"
	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

// Server represents the gRPC server
type Server struct {
	pb.UnimplementedAgentServiceServer

	config          *config.Config
	logger          zerolog.Logger
	slogger         *slog.Logger // modern structured logger
	server          *grpc.Server
	ocservManager   *ocserv.Manager
	configGenerator *config.Generator
	sessionStore    *storage.SessionStore // In-memory session storage
}

// New creates a new gRPC server instance
func New(cfg *config.Config, logger zerolog.Logger) (*Server, error) {
	s := &Server{
		config:  cfg,
		logger:  logger,
		slogger: slog.Default(),
	}

	// Create ocserv manager
	s.ocservManager = ocserv.NewManager(cfg, logger)

	// Create config generator if directories are configured
	if cfg.Ocserv.ConfigPerUserDir != "" {
		generator, err := config.NewGenerator(
			cfg.Ocserv.ConfigPerUserDir,
			cfg.Ocserv.ConfigPerGroupDir,
			cfg.Ocserv.BackupDir,
		)
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to create config generator")
		} else {
			s.configGenerator = generator
		}
	}

	// Create session store with 24h TTL
	s.sessionStore = storage.NewSessionStore(24 * time.Hour)

	// Create gRPC server with TLS
	grpcServer, err := s.createGRPCServer()
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC server: %w", err)
	}

	s.server = grpcServer

	// Register AgentService
	pb.RegisterAgentServiceServer(s.server, s)

	// Register VPNAgentService
	vpnService := NewVPNService(s, slog.Default())
	pb.RegisterVPNAgentServiceServer(s.server, vpnService)

	// Register reflection service (for grpcurl and other tools)
	reflection.Register(s.server)

	return s, nil
}

// createGRPCServer creates a gRPC server with mTLS if enabled
func (s *Server) createGRPCServer() (*grpc.Server, error) {
	var opts []grpc.ServerOption

	// Add TLS credentials if enabled
	if s.config.TLS.Enabled {
		tlsCreds, err := s.loadTLSCredentials()
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.Creds(tlsCreds))
		s.logger.Info().Msg("mTLS enabled for gRPC server")
	} else {
		s.logger.Warn().Msg("TLS is disabled - running in insecure mode")
	}

	// Add interceptors
	opts = append(opts,
		grpc.ChainUnaryInterceptor(
			s.loggingInterceptor(),
			s.recoveryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			s.streamLoggingInterceptor(),
		),
	)

	return grpc.NewServer(opts...), nil
}

// loadTLSCredentials loads mTLS credentials
func (s *Server) loadTLSCredentials() (credentials.TransportCredentials, error) {
	// Load CA certificate
	caCert, err := os.ReadFile(s.config.TLS.CAFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	// Load server certificate and key
	serverCert, err := tls.LoadX509KeyPair(s.config.TLS.CertFile, s.config.TLS.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	// Configure TLS with secure defaults
	// MinVersion is guaranteed to be >= TLS 1.2 by config validation
	minVersion := s.getTLSVersion()

	// G402: Ensure minimum TLS version is 1.2 or higher
	// This is validated in internal/config/validation.go:102-114
	// Default is TLS 1.3, fallback is also TLS 1.3
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
		MinVersion:   minVersion, // #nosec G402 - validated by config validation
		CipherSuites: []uint16{
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
	}

	return credentials.NewTLS(tlsConfig), nil
}

// getTLSVersion returns the TLS version from config
// Returns TLS 1.3 by default for maximum security
func (s *Server) getTLSVersion() uint16 {
	switch s.config.TLS.MinVersion {
	case "TLS1.3":
		return tls.VersionTLS13
	case "TLS1.2":
		return tls.VersionTLS12
	default:
		// Default to TLS 1.3 for any invalid/empty value
		// Validation in config package ensures only TLS1.2/TLS1.3 are accepted
		return tls.VersionTLS13
	}
}

// Serve starts the gRPC server
func (s *Server) Serve(address string) error {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}

	s.logger.Info().
		Str("address", address).
		Bool("tls_enabled", s.config.TLS.Enabled).
		Msg("Starting gRPC server")

	if err := s.server.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}

	return nil
}

// GracefulStop gracefully stops the gRPC server
func (s *Server) GracefulStop() {
	s.logger.Info().Msg("Gracefully stopping gRPC server")
	s.server.GracefulStop()
}

// Stop forcefully stops the gRPC server
func (s *Server) Stop() {
	s.logger.Warn().Msg("Forcefully stopping gRPC server")
	s.server.Stop()
}

// loggingInterceptor logs all unary RPC calls
func (s *Server) loggingInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		s.logger.Info().
			Str("method", info.FullMethod).
			Msg("RPC call")

		resp, err := handler(ctx, req)

		if err != nil {
			s.logger.Error().
				Err(err).
				Str("method", info.FullMethod).
				Msg("RPC call failed")
		}

		return resp, err
	}
}

// recoveryInterceptor recovers from panics in RPC handlers
func (s *Server) recoveryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error().
					Interface("panic", r).
					Str("method", info.FullMethod).
					Msg("Recovered from panic in RPC handler")
				err = fmt.Errorf("internal server error")
			}
		}()

		return handler(ctx, req)
	}
}

// streamLoggingInterceptor logs all streaming RPC calls
func (s *Server) streamLoggingInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		s.logger.Info().
			Str("method", info.FullMethod).
			Bool("is_client_stream", info.IsClientStream).
			Bool("is_server_stream", info.IsServerStream).
			Msg("Stream RPC call")

		err := handler(srv, ss)

		if err != nil {
			s.logger.Error().
				Err(err).
				Str("method", info.FullMethod).
				Msg("Stream RPC call failed")
		}

		return err
	}
}
