package grpc

import (
	"context"
	"log/slog"
	"testing"

	pb "github.com/dantte-lp/ocserv-agent/pkg/proto/agent/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestVPNService_NotifyConnect(t *testing.T) {
	// Создаём тестовый server (с минимальной конфигурацией)
	// TODO: Добавить mock для ocservManager

	server := &Server{}
	vpnService := &VPNService{
		server: server,
		logger: slog.Default(),
	}

	req := &pb.NotifyConnectRequest{
		Username:    "testuser",
		ClientIp:    "192.168.1.100",
		VpnIp:       "10.0.0.1",
		SessionId:   "session-123",
		DeviceId:    "device-456",
		ConnectTime: timestamppb.Now(),
	}

	resp, err := vpnService.NotifyConnect(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Allowed)
	assert.False(t, resp.ShouldDisconnect)
}

func TestVPNService_NotifyDisconnect(t *testing.T) {
	server := &Server{}
	vpnService := &VPNService{
		server: server,
		logger: slog.Default(),
	}

	req := &pb.NotifyDisconnectRequest{
		Username:         "testuser",
		SessionId:        "session-123",
		DisconnectTime:   timestamppb.Now(),
		DisconnectReason: "user logout",
		BytesIn:          1024000,
		BytesOut:         2048000,
		DurationSeconds:  3600,
	}

	resp, err := vpnService.NotifyDisconnect(context.Background(), req)

	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Acknowledged)
}

func TestVPNService_DisconnectUser_EmptyUsername(t *testing.T) {
	vpnService := &VPNService{
		logger: slog.Default(),
	}

	req := &pb.DisconnectUserRequest{
		Username: "",
		Reason:   "test",
	}

	resp, err := vpnService.DisconnectUser(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "username is required")
}

func TestVPNService_UpdateUserRoutes_EmptyUsername(t *testing.T) {
	vpnService := &VPNService{
		logger: slog.Default(),
	}

	req := &pb.UpdateUserRoutesRequest{
		Username: "",
		Routes:   []string{"10.0.0.0/8"},
	}

	resp, err := vpnService.UpdateUserRoutes(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "username is required")
}

func TestParseBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected uint64
		wantErr  bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: 0,
			wantErr:  false,
		},
		{
			name:     "parse megabytes",
			input:    "1.5M",
			expected: 1572864, // 1.5 * 1024 * 1024
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBytes(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expected, got)
		})
	}
}
