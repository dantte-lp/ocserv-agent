package grpc

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dantte-lp/ocserv-agent/internal/config"
	"github.com/dantte-lp/ocserv-agent/internal/storage"
	pb "github.com/dantte-lp/ocserv-agent/pkg/proto/agent/v1"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVPNService_SessionStoreIntegration tests SessionStore integration without occtl dependency
func TestVPNService_SessionStoreIntegration(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Ocserv: config.OcservConfig{
			ConfigPerUserDir: tmpDir + "/per-user",
		},
	}

	logger := zerolog.Nop()
	sessionStore := storage.NewSessionStore(24 * time.Hour)

	server := &Server{
		config:       cfg,
		logger:       logger,
		sessionStore: sessionStore,
	}

	vpnService := NewVPNService(server, nil)
	ctx := context.Background()

	// Test: NotifyConnect adds session to store
	t.Run("NotifyConnect adds session", func(t *testing.T) {
		req := &pb.NotifyConnectRequest{
			SessionId: "test-session-1",
			Username:  "alice",
			ClientIp:  "203.0.113.1",
			VpnIp:     "10.10.10.1",
			DeviceId:  "device-1",
			Metadata: map[string]string{
				"user_agent": "TestClient/1.0",
			},
		}

		resp, err := vpnService.NotifyConnect(ctx, req)
		require.NoError(t, err)
		assert.True(t, resp.Allowed)

		// Verify session in store
		assert.Equal(t, 1, sessionStore.Count())
		session, err := sessionStore.Get("test-session-1")
		require.NoError(t, err)
		assert.Equal(t, "alice", session.Username)
		assert.Equal(t, "10.10.10.1", session.VpnIP)
		assert.Equal(t, "TestClient/1.0", session.Metadata["user_agent"])
	})

	// Test: GetActiveSessions from SessionStore
	t.Run("GetActiveSessions from SessionStore", func(t *testing.T) {
		// Add another session
		req2 := &pb.NotifyConnectRequest{
			SessionId: "test-session-2",
			Username:  "bob",
			ClientIp:  "203.0.113.2",
			VpnIp:     "10.10.10.2",
		}
		_, err := vpnService.NotifyConnect(ctx, req2)
		require.NoError(t, err)

		// Get all sessions
		getResp, err := vpnService.GetActiveSessions(ctx, &pb.GetActiveSessionsRequest{})
		require.NoError(t, err)
		assert.Equal(t, uint32(2), getResp.TotalCount)

		// Get sessions for specific user
		getResp2, err := vpnService.GetActiveSessions(ctx, &pb.GetActiveSessionsRequest{
			UsernameFilter: "alice",
		})
		require.NoError(t, err)
		assert.Equal(t, uint32(1), getResp2.TotalCount)
		assert.Equal(t, "alice", getResp2.Sessions[0].Username)
	})

	// Test: NotifyDisconnect removes session
	t.Run("NotifyDisconnect removes session", func(t *testing.T) {
		req := &pb.NotifyDisconnectRequest{
			SessionId:        "test-session-1",
			Username:         "alice",
			DisconnectReason: "test",
			BytesIn:          1024,
			BytesOut:         2048,
		}

		resp, err := vpnService.NotifyDisconnect(ctx, req)
		require.NoError(t, err)
		assert.True(t, resp.Acknowledged)

		// Verify session removed
		assert.Equal(t, 1, sessionStore.Count())
		_, err = sessionStore.Get("test-session-1")
		assert.Error(t, err)
	})
}

// TestVPNService_ConcurrentSessions tests concurrent session operations
func TestVPNService_ConcurrentSessions(t *testing.T) {
	cfg := &config.Config{}
	logger := zerolog.Nop()
	sessionStore := storage.NewSessionStore(0) // No TTL

	server := &Server{
		config:       cfg,
		logger:       logger,
		sessionStore: sessionStore,
	}

	vpnService := NewVPNService(server, nil)
	ctx := context.Background()

	const numSessions = 50
	errChan := make(chan error, numSessions*2)

	// Concurrent connects
	for i := 0; i < numSessions; i++ {
		go func(id int) {
			req := &pb.NotifyConnectRequest{
				SessionId: fmt.Sprintf("session-%d", id),
				Username:  fmt.Sprintf("user-%d", id),
				ClientIp:  "203.0.113.100",
				VpnIp:     fmt.Sprintf("10.10.10.%d", id),
			}
			_, err := vpnService.NotifyConnect(ctx, req)
			errChan <- err
		}(i)
	}

	// Wait for all connects
	for i := 0; i < numSessions; i++ {
		err := <-errChan
		assert.NoError(t, err)
	}

	assert.Equal(t, numSessions, sessionStore.Count())

	// Concurrent disconnects
	for i := 0; i < numSessions; i++ {
		go func(id int) {
			req := &pb.NotifyDisconnectRequest{
				SessionId: fmt.Sprintf("session-%d", id),
				Username:  fmt.Sprintf("user-%d", id),
			}
			_, err := vpnService.NotifyDisconnect(ctx, req)
			errChan <- err
		}(i)
	}

	// Wait for all disconnects
	for i := 0; i < numSessions; i++ {
		err := <-errChan
		assert.NoError(t, err)
	}

	assert.Equal(t, 0, sessionStore.Count())
}
