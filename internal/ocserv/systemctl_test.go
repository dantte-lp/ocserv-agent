package ocserv

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// TestNewSystemctlManager tests creating a new SystemctlManager
func TestNewSystemctlManager(t *testing.T) {
	logger := zerolog.Nop()
	manager := NewSystemctlManager("test-service", "", 10*time.Second, logger)

	if manager == nil {
		t.Fatal("Manager should not be nil")
	}

	if manager.serviceName != "test-service" {
		t.Errorf("Expected serviceName 'test-service', got '%s'", manager.serviceName)
	}

	if manager.timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", manager.timeout)
	}
}

// TestNewSystemctlManagerWithSudo tests creating a manager with sudo user
func TestNewSystemctlManagerWithSudo(t *testing.T) {
	logger := zerolog.Nop()
	manager := NewSystemctlManager("test-service", "root", 10*time.Second, logger)

	if manager == nil {
		t.Fatal("Manager should not be nil")
	}

	if manager.sudoUser != "root" {
		t.Errorf("Expected sudoUser 'root', got '%s'", manager.sudoUser)
	}
}

// TestSystemctlManagerMethods tests that methods don't panic
func TestSystemctlManagerMethods(t *testing.T) {
	logger := zerolog.Nop()
	manager := NewSystemctlManager("non-existent-service-xyz", "", 1*time.Second, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// These will fail but shouldn't panic
	t.Run("Start", func(t *testing.T) {
		err := manager.Start(ctx)
		if err == nil {
			t.Log("Start unexpectedly succeeded (service might exist)")
		} else {
			t.Logf("Start failed as expected: %v", err)
		}
	})

	t.Run("Stop", func(t *testing.T) {
		err := manager.Stop(ctx)
		if err == nil {
			t.Log("Stop unexpectedly succeeded")
		} else {
			t.Logf("Stop failed as expected: %v", err)
		}
	})

	t.Run("Restart", func(t *testing.T) {
		err := manager.Restart(ctx)
		if err == nil {
			t.Log("Restart unexpectedly succeeded")
		} else {
			t.Logf("Restart failed as expected: %v", err)
		}
	})

	t.Run("Reload", func(t *testing.T) {
		err := manager.Reload(ctx)
		if err == nil {
			t.Log("Reload unexpectedly succeeded")
		} else {
			t.Logf("Reload failed as expected: %v", err)
		}
	})

	t.Run("Status", func(t *testing.T) {
		status, err := manager.Status(ctx)
		if err != nil {
			t.Logf("Status failed as expected: %v", err)
		}
		if status != nil && status.LoadState == "not-found" {
			t.Log("Status correctly identified service as not-found")
		}
	})

	t.Run("IsActive", func(t *testing.T) {
		active, err := manager.IsActive(ctx)
		if err != nil {
			t.Logf("IsActive failed as expected: %v", err)
		}
		if !active {
			t.Log("IsActive correctly returned false")
		}
	})

	t.Run("IsEnabled", func(t *testing.T) {
		enabled, err := manager.IsEnabled(ctx)
		if err != nil {
			t.Logf("IsEnabled failed as expected: %v", err)
		}
		if !enabled {
			t.Log("IsEnabled correctly returned false")
		}
	})
}

// TestSystemctlManagerTimeout tests timeout handling
func TestSystemctlManagerTimeout(t *testing.T) {
	logger := zerolog.Nop()
	manager := NewSystemctlManager("test-service", "", 100*time.Millisecond, logger)

	// Create already-expired context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(10 * time.Millisecond) // Ensure context is expired

	err := manager.Start(ctx)
	if err == nil {
		t.Error("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "deadline") {
		t.Logf("Got error (may not be timeout-specific): %v", err)
	}
}

// TestServiceStatusParsing tests ServiceStatus struct
func TestServiceStatusParsing(t *testing.T) {
	status := &ServiceStatus{
		Active:      true,
		State:       "running",
		SubState:    "running",
		Description: "Test Service",
		MainPID:     1234,
		LoadState:   "loaded",
	}

	if !status.Active {
		t.Error("Expected Active to be true")
	}

	if status.State != "running" {
		t.Errorf("Expected State 'running', got '%s'", status.State)
	}

	if status.MainPID != 1234 {
		t.Errorf("Expected MainPID 1234, got %d", status.MainPID)
	}
}

// TestServiceStatusFields tests all ServiceStatus fields
func TestServiceStatusFields(t *testing.T) {
	tests := []struct {
		name   string
		status ServiceStatus
	}{
		{
			name: "RunningService",
			status: ServiceStatus{
				Active:      true,
				State:       "running",
				SubState:    "running",
				Description: "Running Service",
				MainPID:     1234,
				LoadState:   "loaded",
			},
		},
		{
			name: "StoppedService",
			status: ServiceStatus{
				Active:      false,
				State:       "inactive",
				SubState:    "dead",
				Description: "Stopped Service",
				MainPID:     0,
				LoadState:   "loaded",
			},
		},
		{
			name: "FailedService",
			status: ServiceStatus{
				Active:      false,
				State:       "failed",
				SubState:    "failed",
				Description: "Failed Service",
				MainPID:     0,
				LoadState:   "loaded",
			},
		},
		{
			name: "NotFoundService",
			status: ServiceStatus{
				Active:      false,
				State:       "inactive",
				SubState:    "dead",
				Description: "",
				MainPID:     0,
				LoadState:   "not-found",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.status

			// Just verify fields are accessible
			_ = s.Active
			_ = s.State
			_ = s.SubState
			_ = s.Description
			_ = s.MainPID
			_ = s.LoadState

			t.Logf("Status: Active=%v, State=%s, LoadState=%s",
				s.Active, s.State, s.LoadState)
		})
	}
}
