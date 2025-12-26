package ipc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/dantte-lp/ocserv-agent/internal/resilience"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// PortalClient defines the interface for communicating with the portal
type PortalClient interface {
	// CheckPolicy validates user access policy
	CheckPolicy(ctx context.Context, username, groupName, clientIP string) (bool, string, error)
}

// CacheEntry represents a cached policy decision
type CacheEntry struct {
	Allowed    bool
	DenyReason string
}

// DecisionCache defines the interface for caching policy decisions
type DecisionCache interface {
	// Get retrieves a cached decision (returns *CacheEntry)
	Get(ctx context.Context, key string) (entry interface{}, found bool, err error)
	// Set stores a decision in cache
	Set(ctx context.Context, key string, allowed bool, denyReason string) error
}

// Handler processes IPC authentication requests
type Handler struct {
	logger        *slog.Logger
	tracer        trace.Tracer
	protocol      *Protocol
	portalClient  PortalClient
	decisionCache DecisionCache
	failMode      string // open, close, stale
	timeout       time.Duration

	// Metrics
	requestsTotal   metric.Int64Counter
	requestDuration metric.Float64Histogram
	errorsTotal     metric.Int64Counter
}

// HandlerConfig configures the IPC handler
type HandlerConfig struct {
	Logger        *slog.Logger
	Tracer        trace.Tracer
	Meter         metric.Meter
	PortalClient  PortalClient
	DecisionCache DecisionCache
	FailMode      string // open, close, stale
	Timeout       time.Duration
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
	if cfg.FailMode == "" {
		cfg.FailMode = "stale"
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
		decisionCache:   cfg.DecisionCache,
		failMode:        cfg.FailMode,
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

	// For connect events, check cache first (if available)
	cacheKey := fmt.Sprintf("%s:%s:%s", req.Username, req.GroupName, req.IPReal)

	if h.decisionCache != nil {
		entry, found, err := h.decisionCache.Get(ctx, cacheKey)
		if err == nil && found {
			h.logger.DebugContext(ctx, "using cached decision",
				slog.String("username", req.Username),
				slog.Bool("from_cache", true),
			)

			// Convert to resilience.CacheEntry
			if ce, ok := entry.(*resilience.CacheEntry); ok {
				return AuthResponse{
					Allowed: ce.Allowed,
					Error:   ce.DenyReason,
				}
			}
		}
	}

	// Check with portal
	allowed, message, err := h.portalClient.CheckPolicy(ctx, req.Username, req.GroupName, req.IPReal)
	if err != nil {
		h.logger.ErrorContext(ctx, "portal check failed",
			slog.String("username", req.Username),
			slog.String("error", err.Error()),
		)
		h.errorsTotal.Add(ctx, 1, metric.WithAttributes(
			attribute.String("error_type", "portal"),
		))

		// Apply fail mode policy
		return h.applyFailMode(ctx, req, err)
	}

	// Cache the decision if cache is available
	if h.decisionCache != nil {
		if err := h.decisionCache.Set(ctx, cacheKey, allowed, message); err != nil {
			h.logger.WarnContext(ctx, "failed to cache decision",
				slog.String("error", err.Error()),
			)
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

// applyFailMode applies the configured fail mode when portal is unavailable
func (h *Handler) applyFailMode(ctx context.Context, req *AuthRequest, portalErr error) AuthResponse {
	switch h.failMode {
	case "open":
		// Fail open: allow all connections
		h.logger.WarnContext(ctx, "portal unavailable, failing open (allowing)",
			slog.String("username", req.Username),
			slog.String("error", portalErr.Error()),
		)
		return AuthResponse{
			Allowed: true,
			Message: "portal unavailable, access granted (fail-open mode)",
		}

	case "close":
		// Fail close: deny all connections
		h.logger.WarnContext(ctx, "portal unavailable, failing close (denying)",
			slog.String("username", req.Username),
			slog.String("error", portalErr.Error()),
		)
		return AuthResponse{
			Allowed: false,
			Error:   fmt.Sprintf("portal unavailable: %v", portalErr),
		}

	case "stale":
		// Fail stale: use cached decision if available
		cacheKey := fmt.Sprintf("%s:%s:%s", req.Username, req.GroupName, req.IPReal)

		if h.decisionCache != nil {
			entry, found, err := h.decisionCache.Get(ctx, cacheKey)
			if err == nil && found {
				if ce, ok := entry.(*resilience.CacheEntry); ok {
					h.logger.WarnContext(ctx, "portal unavailable, using stale cache",
						slog.String("username", req.Username),
						slog.Bool("allowed", ce.Allowed),
					)

					return AuthResponse{
						Allowed: ce.Allowed,
						Error:   ce.DenyReason,
					}
				}
			}
		}

		// No cache available, fail close
		h.logger.WarnContext(ctx, "portal unavailable, no cache, denying",
			slog.String("username", req.Username),
		)
		return AuthResponse{
			Allowed: false,
			Error:   fmt.Sprintf("portal unavailable and no cache: %v", portalErr),
		}

	default:
		// Default to fail close
		return AuthResponse{
			Allowed: false,
			Error:   fmt.Sprintf("portal check failed: %v", portalErr),
		}
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
