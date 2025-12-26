package ipc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// PortalClient defines the interface for communicating with the portal
type PortalClient interface {
	// CheckPolicy validates user access policy
	CheckPolicy(ctx context.Context, username, groupName, clientIP string) (bool, string, error)
}

// Handler processes IPC authentication requests
type Handler struct {
	logger       *slog.Logger
	tracer       trace.Tracer
	protocol     *Protocol
	portalClient PortalClient
	timeout      time.Duration

	// Metrics
	requestsTotal   metric.Int64Counter
	requestDuration metric.Float64Histogram
	errorsTotal     metric.Int64Counter
}

// HandlerConfig configures the IPC handler
type HandlerConfig struct {
	Logger       *slog.Logger
	Tracer       trace.Tracer
	Meter        metric.Meter
	PortalClient PortalClient
	Timeout      time.Duration
}

// NewHandler creates a new IPC request handler
func NewHandler(cfg *HandlerConfig) (*Handler, error) {
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if cfg.Tracer == nil {
		return nil, fmt.Errorf("tracer is required")
	}
	if cfg.Meter == nil {
		return nil, fmt.Errorf("meter is required")
	}
	if cfg.PortalClient == nil {
		return nil, fmt.Errorf("portal client is required")
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}

	// Initialize metrics
	requestsTotal, err := cfg.Meter.Int64Counter(
		"ipc.requests.total",
		metric.WithDescription("Total number of IPC requests"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create requests counter: %w", err)
	}

	requestDuration, err := cfg.Meter.Float64Histogram(
		"ipc.request.duration",
		metric.WithDescription("IPC request processing duration"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, fmt.Errorf("create duration histogram: %w", err)
	}

	errorsTotal, err := cfg.Meter.Int64Counter(
		"ipc.errors.total",
		metric.WithDescription("Total number of IPC errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, fmt.Errorf("create errors counter: %w", err)
	}

	return &Handler{
		logger:          cfg.Logger,
		tracer:          cfg.Tracer,
		protocol:        NewProtocol(),
		portalClient:    cfg.PortalClient,
		timeout:         cfg.Timeout,
		requestsTotal:   requestsTotal,
		requestDuration: requestDuration,
		errorsTotal:     errorsTotal,
	}, nil
}

// Handle processes a single IPC connection
func (h *Handler) Handle(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	start := time.Now()
	var req AuthRequest
	var resp AuthResponse

	// Create span for tracing
	ctx, span := h.tracer.Start(ctx, "ipc.handle",
		trace.WithAttributes(
			attribute.String("remote_addr", conn.RemoteAddr().String()),
		),
	)
	defer span.End()

	// Set connection deadline
	deadline := time.Now().Add(h.timeout)
	if err := conn.SetDeadline(deadline); err != nil {
		h.logger.ErrorContext(ctx, "failed to set connection deadline",
			slog.String("error", err.Error()),
		)
		h.errorsTotal.Add(ctx, 1, metric.WithAttributes(
			attribute.String("error_type", "deadline"),
		))
		return
	}

	// Read request
	if err := h.protocol.ReadMessage(conn, &req); err != nil {
		h.logger.ErrorContext(ctx, "failed to read request",
			slog.String("error", err.Error()),
		)
		h.errorsTotal.Add(ctx, 1, metric.WithAttributes(
			attribute.String("error_type", "read"),
		))
		resp = AuthResponse{
			Allowed: false,
			Error:   "failed to read request",
		}
		_ = h.protocol.WriteMessage(conn, &resp)
		return
	}

	// Add request attributes to span
	span.SetAttributes(
		attribute.String("reason", req.Reason),
		attribute.String("username", req.Username),
		attribute.String("group", req.GroupName),
		attribute.String("client_ip", req.IPReal),
		attribute.String("session_id", req.SessionID),
	)

	h.logger.InfoContext(ctx, "processing auth request",
		slog.String("reason", req.Reason),
		slog.String("username", req.Username),
		slog.String("group", req.GroupName),
		slog.String("client_ip", req.IPReal),
		slog.String("vpn_ip", req.IPRemote),
		slog.String("session_id", req.SessionID),
	)

	// Increment request counter
	h.requestsTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("reason", req.Reason),
	))

	// Process request
	resp = h.processRequest(ctx, &req)

	// Record duration
	duration := time.Since(start).Seconds()
	h.requestDuration.Record(ctx, duration, metric.WithAttributes(
		attribute.String("reason", req.Reason),
		attribute.Bool("allowed", resp.Allowed),
	))

	// Write response
	if err := h.protocol.WriteMessage(conn, &resp); err != nil {
		h.logger.ErrorContext(ctx, "failed to write response",
			slog.String("error", err.Error()),
		)
		h.errorsTotal.Add(ctx, 1, metric.WithAttributes(
			attribute.String("error_type", "write"),
		))
		return
	}

	h.logger.InfoContext(ctx, "request processed",
		slog.String("username", req.Username),
		slog.Bool("allowed", resp.Allowed),
		slog.Float64("duration_ms", duration*1000),
	)
}

// processRequest performs the actual authentication logic
func (h *Handler) processRequest(ctx context.Context, req *AuthRequest) AuthResponse {
	// Validate request
	if err := h.validateRequest(req); err != nil {
		return AuthResponse{
			Allowed: false,
			Error:   fmt.Sprintf("validation failed: %v", err),
		}
	}

	// For disconnect events, always allow (just logging)
	if req.Reason == "disconnect" || req.Reason == "host-update" {
		return AuthResponse{
			Allowed: true,
			Message: fmt.Sprintf("reason=%s logged", req.Reason),
		}
	}

	// For connect events, check with portal
	allowed, message, err := h.portalClient.CheckPolicy(ctx, req.Username, req.GroupName, req.IPReal)
	if err != nil {
		h.logger.ErrorContext(ctx, "portal check failed",
			slog.String("username", req.Username),
			slog.String("error", err.Error()),
		)
		h.errorsTotal.Add(ctx, 1, metric.WithAttributes(
			attribute.String("error_type", "portal"),
		))
		return AuthResponse{
			Allowed: false,
			Error:   fmt.Sprintf("portal check failed: %v", err),
		}
	}

	if !allowed {
		h.logger.WarnContext(ctx, "access denied by portal",
			slog.String("username", req.Username),
			slog.String("reason", message),
		)
	}

	return AuthResponse{
		Allowed: allowed,
		Error:   message,
	}
}

// validateRequest performs basic validation of the request
func (h *Handler) validateRequest(req *AuthRequest) error {
	if req.Username == "" {
		return fmt.Errorf("username is required")
	}
	if req.Reason == "" {
		return fmt.Errorf("reason is required")
	}
	if req.Reason != "connect" && req.Reason != "disconnect" && req.Reason != "host-update" {
		return fmt.Errorf("invalid reason: %s", req.Reason)
	}
	return nil
}
