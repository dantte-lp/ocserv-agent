package ipc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Server implements a Unix socket server for IPC communication
type Server struct {
	socketPath string
	listener   net.Listener
	handler    *Handler
	logger     *slog.Logger
	tracer     trace.Tracer

	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc

	// Metrics
	activeConnections metric.Int64UpDownCounter
	connectionsTotal  metric.Int64Counter
}

// ServerConfig configures the IPC server
type ServerConfig struct {
	SocketPath string
	Handler    *Handler
	Logger     *slog.Logger
	Tracer     trace.Tracer
	Meter      metric.Meter
}

// NewServer creates a new IPC server
func NewServer(cfg *ServerConfig) (*Server, error) {
	if cfg.SocketPath == "" {
		return nil, fmt.Errorf("socket path is required")
	}
	if cfg.Handler == nil {
		return nil, fmt.Errorf("handler is required")
	}
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if cfg.Tracer == nil {
		return nil, fmt.Errorf("tracer is required")
	}
	if cfg.Meter == nil {
		return nil, fmt.Errorf("meter is required")
	}

	// Initialize metrics
	activeConnections, err := cfg.Meter.Int64UpDownCounter(
		"ipc.connections.active",
		metric.WithDescription("Number of active IPC connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create active connections counter: %w", err)
	}

	connectionsTotal, err := cfg.Meter.Int64Counter(
		"ipc.connections.total",
		metric.WithDescription("Total number of IPC connections"),
		metric.WithUnit("{connection}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create connections counter: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		socketPath:        cfg.SocketPath,
		handler:           cfg.Handler,
		logger:            cfg.Logger,
		tracer:            cfg.Tracer,
		ctx:               ctx,
		cancel:            cancel,
		activeConnections: activeConnections,
		connectionsTotal:  connectionsTotal,
	}, nil
}

// Start starts the IPC server
func (s *Server) Start(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "ipc.server.start",
		trace.WithAttributes(
			attribute.String("socket_path", s.socketPath),
		),
	)
	defer span.End()

	// Remove existing socket file if it exists
	if err := os.RemoveAll(s.socketPath); err != nil {
		return fmt.Errorf("remove existing socket: %w", err)
	}

	// Create Unix socket listener
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listen on socket: %w", err)
	}
	s.listener = listener

	// Set socket permissions to 0666 (readable/writable by all)
	// This allows ocserv running as root to communicate with the agent
	if err := os.Chmod(s.socketPath, 0666); err != nil {
		listener.Close()
		return fmt.Errorf("chmod socket: %w", err)
	}

	s.logger.InfoContext(ctx, "IPC server started",
		slog.String("socket", s.socketPath),
	)

	// Start accepting connections in background
	s.wg.Add(1)
	go s.acceptLoop()

	return nil
}

// Stop gracefully stops the IPC server
func (s *Server) Stop(ctx context.Context) error {
	ctx, span := s.tracer.Start(ctx, "ipc.server.stop")
	defer span.End()

	s.logger.InfoContext(ctx, "stopping IPC server")

	// Cancel context to signal shutdown
	s.cancel()

	// Close listener to stop accepting new connections
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			s.logger.ErrorContext(ctx, "error closing listener",
				slog.String("error", err.Error()),
			)
		}
	}

	// Wait for all connections to finish with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		s.logger.InfoContext(ctx, "IPC server stopped gracefully")
	case <-time.After(10 * time.Second):
		s.logger.WarnContext(ctx, "IPC server shutdown timeout, forcing close")
	}

	// Remove socket file
	if err := os.RemoveAll(s.socketPath); err != nil {
		s.logger.ErrorContext(ctx, "error removing socket file",
			slog.String("error", err.Error()),
		)
	}

	return nil
}

// acceptLoop accepts incoming connections
func (s *Server) acceptLoop() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				// Server is shutting down
				return
			default:
				s.logger.Error("failed to accept connection",
					slog.String("error", err.Error()),
				)
				continue
			}
		}

		// Increment connection counter
		s.connectionsTotal.Add(s.ctx, 1)

		// Handle connection in goroutine
		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection handles a single connection
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()

	// Track active connections
	s.activeConnections.Add(s.ctx, 1)
	defer s.activeConnections.Add(s.ctx, -1)

	// Create span for connection
	ctx, span := s.tracer.Start(s.ctx, "ipc.connection",
		trace.WithAttributes(
			attribute.String("remote_addr", conn.RemoteAddr().String()),
		),
	)
	defer span.End()

	// Delegate to handler
	s.handler.Handle(ctx, conn)
}
