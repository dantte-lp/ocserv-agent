package ipc

import (
	"context"
	"fmt"
	"log/slog"

	vpnv1 "github.com/dantte-lp/ocserv-agent/pkg/proto/vpn/v1"
)

// FailMode defines behavior when portal is unavailable
type FailMode int

const (
	// FailOpen - allow connections when portal is unavailable
	// SECURITY WARNING: Use only for development/testing
	FailOpen FailMode = iota

	// FailClose - deny connections when portal is unavailable (default)
	// RECOMMENDED for production environments
	FailClose

	// FailStale - use stale cache when portal is unavailable
	// Balances security and availability
	FailStale
)

func (m FailMode) String() string {
	switch m {
	case FailOpen:
		return "fail-open"
	case FailClose:
		return "fail-close"
	case FailStale:
		return "fail-stale"
	default:
		return "unknown"
	}
}

// ParseFailMode parses fail mode from string
func ParseFailMode(s string) (FailMode, error) {
	switch s {
	case "fail-open", "open":
		return FailOpen, nil
	case "fail-close", "close":
		return FailClose, nil
	case "fail-stale", "stale":
		return FailStale, nil
	default:
		return FailClose, fmt.Errorf("invalid fail mode: %s (valid: fail-open, fail-close, fail-stale)", s)
	}
}

// FailModeHandler handles failures according to configured fail mode
type FailModeHandler struct {
	mode   FailMode
	logger *slog.Logger
}

// NewFailModeHandler creates a new fail mode handler
func NewFailModeHandler(mode FailMode, logger *slog.Logger) *FailModeHandler {
	return &FailModeHandler{
		mode:   mode,
		logger: logger,
	}
}

// HandleFailure handles a failure according to fail mode
// Returns (response, shouldUseResponse, error)
func (h *FailModeHandler) HandleFailure(ctx context.Context, req *vpnv1.CheckPolicyRequest, originalErr error) (*vpnv1.CheckPolicyResponse, bool, error) {
	switch h.mode {
	case FailOpen:
		return h.handleFailOpen(ctx, req, originalErr)
	case FailClose:
		return h.handleFailClose(ctx, req, originalErr)
	case FailStale:
		return h.handleFailStale(ctx, req, originalErr)
	default:
		return h.handleFailClose(ctx, req, originalErr)
	}
}

// handleFailOpen allows connection when portal is down
func (h *FailModeHandler) handleFailOpen(ctx context.Context, req *vpnv1.CheckPolicyRequest, originalErr error) (*vpnv1.CheckPolicyResponse, bool, error) {
	h.logger.WarnContext(ctx, "portal unavailable, allowing connection (fail-open mode)",
		slog.String("username", req.Username),
		slog.String("client_ip", req.ClientIp),
		slog.String("error", originalErr.Error()),
	)

	// Create permissive response
	resp := &vpnv1.CheckPolicyResponse{
		Allowed:    true,
		DenyReason: "",
		Routes:     []string{}, // Use default routes from ocserv config
		DnsServers: []string{}, // Use default DNS from ocserv config
		Metadata: map[string]string{
			"fallback":     "true",
			"fail_mode":    "open",
			"portal_error": originalErr.Error(),
		},
	}

	return resp, true, nil
}

// handleFailClose denies connection when portal is down
func (h *FailModeHandler) handleFailClose(ctx context.Context, req *vpnv1.CheckPolicyRequest, originalErr error) (*vpnv1.CheckPolicyResponse, bool, error) {
	h.logger.WarnContext(ctx, "portal unavailable, denying connection (fail-close mode)",
		slog.String("username", req.Username),
		slog.String("client_ip", req.ClientIp),
		slog.String("error", originalErr.Error()),
	)

	// Create deny response
	resp := &vpnv1.CheckPolicyResponse{
		Allowed:    false,
		DenyReason: "authorization service temporarily unavailable",
		Metadata: map[string]string{
			"fail_mode":    "close",
			"portal_error": originalErr.Error(),
		},
	}

	return resp, true, nil
}

// handleFailStale returns error to signal caller to use stale cache
func (h *FailModeHandler) handleFailStale(ctx context.Context, req *vpnv1.CheckPolicyRequest, originalErr error) (*vpnv1.CheckPolicyResponse, bool, error) {
	h.logger.InfoContext(ctx, "portal unavailable, signaling stale cache usage (fail-stale mode)",
		slog.String("username", req.Username),
		slog.String("client_ip", req.ClientIp),
		slog.String("error", originalErr.Error()),
	)

	// Return nil response to signal caller should check stale cache
	return nil, false, fmt.Errorf("portal unavailable, check stale cache: %w", originalErr)
}

// IsFailOpen returns true if mode is fail-open
func (h *FailModeHandler) IsFailOpen() bool {
	return h.mode == FailOpen
}

// IsFailClose returns true if mode is fail-close
func (h *FailModeHandler) IsFailClose() bool {
	return h.mode == FailClose
}

// IsFailStale returns true if mode is fail-stale
func (h *FailModeHandler) IsFailStale() bool {
	return h.mode == FailStale
}

// Mode returns current fail mode
func (h *FailModeHandler) Mode() FailMode {
	return h.mode
}

// SetMode changes fail mode at runtime
func (h *FailModeHandler) SetMode(mode FailMode) {
	if h.mode != mode {
		h.logger.Info("fail mode changed",
			slog.String("from", h.mode.String()),
			slog.String("to", mode.String()),
		)
		h.mode = mode
	}
}

// ValidateConfig validates fail mode configuration
func ValidateConfig(mode FailMode, environment string) error {
	// In production, warn about fail-open
	if mode == FailOpen && environment == "production" {
		return fmt.Errorf("fail-open mode is NOT recommended for production (security risk)")
	}

	// Recommend fail-stale for production
	if mode == FailClose && environment == "production" {
		// This is just a warning, not an error
		// fail-close is acceptable but fail-stale is better
	}

	return nil
}

// RecommendedFailMode returns recommended fail mode for environment
func RecommendedFailMode(environment string) FailMode {
	switch environment {
	case "production":
		return FailStale // Best balance of security and availability
	case "staging":
		return FailStale
	case "development":
		return FailOpen // Convenient for development
	default:
		return FailClose // Safe default
	}
}
