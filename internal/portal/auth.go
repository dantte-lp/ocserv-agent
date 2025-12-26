package portal

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	vpnv1 "github.com/dantte-lp/ocserv-agent/pkg/proto/vpn/v1"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// CheckPolicy validates user access policy via portal
func (c *Client) CheckPolicy(ctx context.Context, username, groupName, clientIP string) (bool, string, error) {
	ctx, span := c.tracer.Start(ctx, "portal.check_policy",
		trace.WithAttributes(
			attribute.String("username", username),
			attribute.String("group", groupName),
			attribute.String("client_ip", clientIP),
		),
	)
	defer span.End()

	// Create auth service client
	authClient := vpnv1.NewAuthServiceClient(c.conn)

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// Prepare request
	req := &vpnv1.CheckPolicyRequest{
		Username:    username,
		Groupname:   groupName,
		ClientIp:    clientIP,
		RequestTime: timestamppb.Now(),
	}

	c.logger.InfoContext(ctx, "checking policy via portal",
		"username", username,
		"group", groupName,
		"client_ip", clientIP,
	)

	// Call portal gRPC service
	resp, err := authClient.CheckPolicy(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "policy check failed")
		return false, "", errors.Wrap(err, "grpc CheckPolicy")
	}

	// Record response
	span.SetAttributes(
		attribute.Bool("allowed", resp.Allowed),
		attribute.String("deny_reason", resp.DenyReason),
		attribute.Bool("should_disconnect", resp.ShouldDisconnect),
	)

	if !resp.Allowed {
		c.logger.WarnContext(ctx, "access denied by portal",
			"username", username,
			"reason", resp.DenyReason,
		)
		return false, resp.DenyReason, nil
	}

	c.logger.InfoContext(ctx, "access allowed by portal",
		"username", username,
		"routes_count", len(resp.Routes),
		"dns_count", len(resp.DnsServers),
	)

	// Return success (routes will be handled by caller if needed)
	return true, "", nil
}

// ValidateSession validates an active VPN session
func (c *Client) ValidateSession(ctx context.Context, sessionID, username string) (bool, error) {
	ctx, span := c.tracer.Start(ctx, "portal.validate_session",
		trace.WithAttributes(
			attribute.String("session_id", sessionID),
			attribute.String("username", username),
		),
	)
	defer span.End()

	// Create auth service client
	authClient := vpnv1.NewAuthServiceClient(c.conn)

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// Prepare request
	req := &vpnv1.ValidateSessionRequest{
		Username:    username,
		SessionId:   sessionID,
		RequestTime: timestamppb.Now(),
	}

	c.logger.InfoContext(ctx, "validating session via portal",
		"session_id", sessionID,
		"username", username,
	)

	// Call portal gRPC service
	resp, err := authClient.ValidateSession(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "session validation failed")
		return false, errors.Wrap(err, "grpc ValidateSession")
	}

	// Record response
	span.SetAttributes(
		attribute.Bool("valid", resp.Valid),
		attribute.String("invalid_reason", resp.InvalidReason),
		attribute.Bool("force_disconnect", resp.ForceDisconnect),
	)

	if !resp.Valid {
		c.logger.WarnContext(ctx, "session invalid",
			"session_id", sessionID,
			"reason", resp.InvalidReason,
			"force_disconnect", resp.ForceDisconnect,
		)
	}

	return resp.Valid, nil
}

// ReportConnect reports a new connection to portal
func (c *Client) ReportConnect(ctx context.Context, sessionID, username, groupName, clientIP, vpnIP, device string) error {
	ctx, span := c.tracer.Start(ctx, "portal.report_connect",
		trace.WithAttributes(
			attribute.String("session_id", sessionID),
			attribute.String("username", username),
		),
	)
	defer span.End()

	// Create event service client
	eventClient := vpnv1.NewEventServiceClient(c.conn)

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// Prepare request
	req := &vpnv1.ReportConnectRequest{
		Username:    username,
		SessionId:   sessionID,
		ClientIp:    clientIP,
		VpnIp:       vpnIP,
		Device:      device,
		ConnectedAt: timestamppb.Now(),
		Groupname:   groupName,
	}

	c.logger.InfoContext(ctx, "reporting connection to portal",
		"session_id", sessionID,
		"username", username,
	)

	// Call portal gRPC service
	resp, err := eventClient.ReportConnect(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "report connect failed")
		return errors.Wrap(err, "grpc ReportConnect")
	}

	c.logger.InfoContext(ctx, "connection reported",
		"session_id", sessionID,
		"success", resp.Success,
		"message", resp.Message,
	)

	return nil
}

// ReportDisconnect reports a disconnection to portal
func (c *Client) ReportDisconnect(ctx context.Context, sessionID, username string, duration time.Duration, bytesRX, bytesTX uint64) error {
	ctx, span := c.tracer.Start(ctx, "portal.report_disconnect",
		trace.WithAttributes(
			attribute.String("session_id", sessionID),
			attribute.String("username", username),
		),
	)
	defer span.End()

	// Create event service client
	eventClient := vpnv1.NewEventServiceClient(c.conn)

	// Apply timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.Timeout)
	defer cancel()

	// Prepare request
	req := &vpnv1.ReportDisconnectRequest{
		Username:       username,
		SessionId:      sessionID,
		DisconnectedAt: timestamppb.Now(),
		Reason:         vpnv1.DisconnectReason_DISCONNECT_REASON_USER_INITIATED,
		Stats: &vpnv1.SessionStats{
			DurationSeconds: int64(duration.Seconds()),
			BytesReceived:   bytesRX,
			BytesSent:       bytesTX,
		},
	}

	c.logger.InfoContext(ctx, "reporting disconnection to portal",
		"session_id", sessionID,
		"username", username,
		"duration", duration,
	)

	// Call portal gRPC service
	resp, err := eventClient.ReportDisconnect(ctx, req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "report disconnect failed")
		return errors.Wrap(err, "grpc ReportDisconnect")
	}

	c.logger.InfoContext(ctx, "disconnection reported",
		"session_id", sessionID,
		"success", resp.Success,
		"message", resp.Message,
	)

	return nil
}
