package ipc

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	vpnv1 "github.com/dantte-lp/ocserv-agent/pkg/proto/vpn/v1"
)

func TestParseFailMode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    FailMode
		wantErr bool
	}{
		{
			name:    "fail-open",
			input:   "fail-open",
			want:    FailOpen,
			wantErr: false,
		},
		{
			name:    "open short form",
			input:   "open",
			want:    FailOpen,
			wantErr: false,
		},
		{
			name:    "fail-close",
			input:   "fail-close",
			want:    FailClose,
			wantErr: false,
		},
		{
			name:    "close short form",
			input:   "close",
			want:    FailClose,
			wantErr: false,
		},
		{
			name:    "fail-stale",
			input:   "fail-stale",
			want:    FailStale,
			wantErr: false,
		},
		{
			name:    "stale short form",
			input:   "stale",
			want:    FailStale,
			wantErr: false,
		},
		{
			name:    "invalid",
			input:   "invalid",
			want:    FailClose,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFailMode(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFailMode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseFailMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFailModeString(t *testing.T) {
	tests := []struct {
		name string
		mode FailMode
		want string
	}{
		{
			name: "fail-open",
			mode: FailOpen,
			want: "fail-open",
		},
		{
			name: "fail-close",
			mode: FailClose,
			want: "fail-close",
		},
		{
			name: "fail-stale",
			mode: FailStale,
			want: "fail-stale",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("FailMode.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFailModeHandler_HandleFailure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	ctx := context.Background()
	testErr := errors.New("portal unavailable")

	req := &vpnv1.CheckPolicyRequest{
		Username: "testuser",
		ClientIp: "192.168.1.100",
		VpnIp:    "10.10.10.100",
	}

	tests := []struct {
		name            string
		mode            FailMode
		wantAllowed     bool
		wantUseResponse bool
		wantErr         bool
	}{
		{
			name:            "fail-open allows connection",
			mode:            FailOpen,
			wantAllowed:     true,
			wantUseResponse: true,
			wantErr:         false,
		},
		{
			name:            "fail-close denies connection",
			mode:            FailClose,
			wantAllowed:     false,
			wantUseResponse: true,
			wantErr:         false,
		},
		{
			name:            "fail-stale signals stale cache",
			mode:            FailStale,
			wantAllowed:     false,
			wantUseResponse: false,
			wantErr:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewFailModeHandler(tt.mode, logger)
			resp, useResp, err := handler.HandleFailure(ctx, req, testErr)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleFailure() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if useResp != tt.wantUseResponse {
				t.Errorf("HandleFailure() useResponse = %v, want %v", useResp, tt.wantUseResponse)
			}

			if useResp && resp != nil {
				if resp.Allowed != tt.wantAllowed {
					t.Errorf("HandleFailure() allowed = %v, want %v", resp.Allowed, tt.wantAllowed)
				}

				// Check metadata
				if resp.Metadata == nil {
					t.Error("HandleFailure() metadata is nil")
				} else {
					if tt.mode == FailOpen {
						if resp.Metadata["fail_mode"] != "open" {
							t.Errorf("HandleFailure() fail_mode metadata = %v, want 'open'", resp.Metadata["fail_mode"])
						}
					} else if tt.mode == FailClose {
						if resp.Metadata["fail_mode"] != "close" {
							t.Errorf("HandleFailure() fail_mode metadata = %v, want 'close'", resp.Metadata["fail_mode"])
						}
					}
				}
			}
		})
	}
}

func TestFailModeHandler_SetMode(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	handler := NewFailModeHandler(FailClose, logger)

	if handler.Mode() != FailClose {
		t.Errorf("initial mode = %v, want FailClose", handler.Mode())
	}

	handler.SetMode(FailOpen)
	if handler.Mode() != FailOpen {
		t.Errorf("mode after SetMode = %v, want FailOpen", handler.Mode())
	}

	if !handler.IsFailOpen() {
		t.Error("IsFailOpen() should return true")
	}

	handler.SetMode(FailStale)
	if !handler.IsFailStale() {
		t.Error("IsFailStale() should return true")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		mode        FailMode
		environment string
		wantErr     bool
	}{
		{
			name:        "fail-open in production - ERROR",
			mode:        FailOpen,
			environment: "production",
			wantErr:     true,
		},
		{
			name:        "fail-close in production - OK",
			mode:        FailClose,
			environment: "production",
			wantErr:     false,
		},
		{
			name:        "fail-stale in production - OK",
			mode:        FailStale,
			environment: "production",
			wantErr:     false,
		},
		{
			name:        "fail-open in development - OK",
			mode:        FailOpen,
			environment: "development",
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.mode, tt.environment)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRecommendedFailMode(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		want        FailMode
	}{
		{
			name:        "production",
			environment: "production",
			want:        FailStale,
		},
		{
			name:        "staging",
			environment: "staging",
			want:        FailStale,
		},
		{
			name:        "development",
			environment: "development",
			want:        FailOpen,
		},
		{
			name:        "unknown",
			environment: "unknown",
			want:        FailClose,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RecommendedFailMode(tt.environment); got != tt.want {
				t.Errorf("RecommendedFailMode() = %v, want %v", got, tt.want)
			}
		})
	}
}
