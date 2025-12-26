package cache

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	vpnv1 "github.com/dantte-lp/ocserv-agent/pkg/proto/vpn/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestDecisionCache_GetSet(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultCacheConfig()
	cfg.TTL = 1 * time.Second

	cache := NewDecisionCache(cfg, logger)
	ctx := context.Background()

	req := &vpnv1.CheckPolicyRequest{
		Username:    "testuser",
		Groupname:   "testgroup",
		ClientIp:    "192.168.1.100",
		VpnIp:       "10.10.10.100",
		SessionId:   "session123",
		Device:      "vpns0",
		RequestTime: timestamppb.Now(),
	}

	resp := &vpnv1.CheckPolicyResponse{
		Allowed:    true,
		DenyReason: "",
		Routes:     []string{"10.0.0.0/8"},
		DnsServers: []string{"8.8.8.8"},
	}

	// Should miss initially
	_, found := cache.Get(ctx, req)
	if found {
		t.Error("Get() found entry that was not set")
	}

	// Set entry
	cache.Set(ctx, req, resp)

	// Should hit now
	cached, found := cache.Get(ctx, req)
	if !found {
		t.Error("Get() did not find cached entry")
	}

	if cached.Allowed != resp.Allowed {
		t.Errorf("cached.Allowed = %v, want %v", cached.Allowed, resp.Allowed)
	}
}

func TestDecisionCache_Expiry(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultCacheConfig()
	cfg.TTL = 100 * time.Millisecond

	cache := NewDecisionCache(cfg, logger)
	ctx := context.Background()

	req := &vpnv1.CheckPolicyRequest{
		Username:  "testuser",
		ClientIp:  "192.168.1.100",
		VpnIp:     "10.10.10.100",
		SessionId: "session123",
	}

	resp := &vpnv1.CheckPolicyResponse{
		Allowed: true,
	}

	cache.Set(ctx, req, resp)

	// Should be cached
	_, found := cache.Get(ctx, req)
	if !found {
		t.Error("Get() did not find fresh entry")
	}

	// Wait for expiry
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, found = cache.Get(ctx, req)
	if found {
		t.Error("Get() found expired entry")
	}
}

func TestDecisionCache_GetStale(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultCacheConfig()
	cfg.TTL = 100 * time.Millisecond
	cfg.StaleTTL = 500 * time.Millisecond

	cache := NewDecisionCache(cfg, logger)
	ctx := context.Background()

	req := &vpnv1.CheckPolicyRequest{
		Username:  "testuser",
		ClientIp:  "192.168.1.100",
		VpnIp:     "10.10.10.100",
		SessionId: "session123",
	}

	resp := &vpnv1.CheckPolicyResponse{
		Allowed: true,
	}

	cache.Set(ctx, req, resp)

	// Wait for normal expiry
	time.Sleep(150 * time.Millisecond)

	// Normal get should miss
	_, found := cache.Get(ctx, req)
	if found {
		t.Error("Get() found expired entry")
	}

	// Stale get should still hit
	stale, found := cache.GetStale(ctx, req)
	if !found {
		t.Error("GetStale() did not find stale entry")
	}

	if !stale.Allowed {
		t.Error("GetStale() returned wrong data")
	}

	// Wait for stale expiry
	time.Sleep(400 * time.Millisecond)

	// Stale get should miss now
	_, found = cache.GetStale(ctx, req)
	if found {
		t.Error("GetStale() found entry past stale TTL")
	}
}

func TestDecisionCache_Invalidate(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultCacheConfig()

	cache := NewDecisionCache(cfg, logger)
	ctx := context.Background()

	req := &vpnv1.CheckPolicyRequest{
		Username:  "testuser",
		ClientIp:  "192.168.1.100",
		VpnIp:     "10.10.10.100",
		SessionId: "session123",
	}

	resp := &vpnv1.CheckPolicyResponse{
		Allowed: true,
	}

	cache.Set(ctx, req, resp)

	// Should be cached
	_, found := cache.Get(ctx, req)
	if !found {
		t.Error("Get() did not find cached entry")
	}

	// Invalidate
	cache.Invalidate(ctx, req)

	// Should be gone
	_, found = cache.Get(ctx, req)
	if found {
		t.Error("Get() found invalidated entry")
	}
}

func TestDecisionCache_InvalidateUser(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultCacheConfig()

	cache := NewDecisionCache(cfg, logger)
	ctx := context.Background()

	// Add entries for testuser
	for i := 0; i < 3; i++ {
		req := &vpnv1.CheckPolicyRequest{
			Username:  "testuser",
			ClientIp:  "192.168.1.100",
			VpnIp:     "10.10.10.100",
			SessionId: "session123",
		}
		resp := &vpnv1.CheckPolicyResponse{
			Allowed: true,
		}
		cache.Set(ctx, req, resp)
	}

	// Add entry for otheruser
	otherReq := &vpnv1.CheckPolicyRequest{
		Username:  "otheruser",
		ClientIp:  "192.168.1.200",
		VpnIp:     "10.10.10.200",
		SessionId: "session456",
	}
	otherResp := &vpnv1.CheckPolicyResponse{
		Allowed: true,
	}
	cache.Set(ctx, otherReq, otherResp)

	stats := cache.Stats()
	initialSize := stats.Size

	// Invalidate testuser
	removed := cache.InvalidateUser(ctx, "testuser")
	if removed == 0 {
		t.Error("InvalidateUser() removed 0 entries")
	}

	// otheruser should still be cached
	_, found := cache.Get(ctx, otherReq)
	if !found {
		t.Error("Get() did not find otheruser entry")
	}

	stats = cache.Stats()
	if stats.Size >= initialSize {
		t.Errorf("cache size after invalidation = %d, should be less than %d", stats.Size, initialSize)
	}
}

func TestDecisionCache_Clear(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultCacheConfig()

	cache := NewDecisionCache(cfg, logger)
	ctx := context.Background()

	// Add multiple entries
	for i := 0; i < 5; i++ {
		req := &vpnv1.CheckPolicyRequest{
			Username:  "testuser",
			ClientIp:  "192.168.1.100",
			VpnIp:     "10.10.10.100",
			SessionId: "session123",
		}
		resp := &vpnv1.CheckPolicyResponse{
			Allowed: true,
		}
		cache.Set(ctx, req, resp)
	}

	stats := cache.Stats()
	if stats.Size == 0 {
		t.Error("cache should have entries before Clear()")
	}

	cache.Clear()

	stats = cache.Stats()
	if stats.Size != 0 {
		t.Errorf("cache size after Clear() = %d, want 0", stats.Size)
	}
	if stats.Hits != 0 || stats.Misses != 0 {
		t.Error("stats should be reset after Clear()")
	}
}

func TestDecisionCache_Stats(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultCacheConfig()

	cache := NewDecisionCache(cfg, logger)
	ctx := context.Background()

	req := &vpnv1.CheckPolicyRequest{
		Username:  "testuser",
		ClientIp:  "192.168.1.100",
		VpnIp:     "10.10.10.100",
		SessionId: "session123",
	}

	resp := &vpnv1.CheckPolicyResponse{
		Allowed: true,
	}

	// Miss
	_, _ = cache.Get(ctx, req)

	// Set and hit
	cache.Set(ctx, req, resp)
	_, _ = cache.Get(ctx, req)
	_, _ = cache.Get(ctx, req)

	// Another miss
	otherReq := &vpnv1.CheckPolicyRequest{
		Username:  "otheruser",
		ClientIp:  "192.168.1.200",
		VpnIp:     "10.10.10.200",
		SessionId: "session456",
	}
	_, _ = cache.Get(ctx, otherReq)

	stats := cache.Stats()
	if stats.Hits != 2 {
		t.Errorf("hits = %d, want 2", stats.Hits)
	}
	if stats.Misses != 2 {
		t.Errorf("misses = %d, want 2", stats.Misses)
	}
	if stats.Size != 1 {
		t.Errorf("size = %d, want 1", stats.Size)
	}
	if stats.HitRate != 50.0 {
		t.Errorf("hit rate = %.2f, want 50.00", stats.HitRate)
	}
}

func TestDecisionCache_MaxSize(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultCacheConfig()
	cfg.MaxSize = 3

	cache := NewDecisionCache(cfg, logger)
	ctx := context.Background()

	// Add 5 entries (should evict oldest 2)
	for i := 0; i < 5; i++ {
		req := &vpnv1.CheckPolicyRequest{
			Username:  "testuser",
			ClientIp:  "192.168.1.100",
			VpnIp:     "10.10.10.100",
			SessionId: "session123",
		}
		resp := &vpnv1.CheckPolicyResponse{
			Allowed: true,
		}
		cache.Set(ctx, req, resp)
	}

	stats := cache.Stats()
	if stats.Size > cfg.MaxSize {
		t.Errorf("cache size = %d, want <= %d", stats.Size, cfg.MaxSize)
	}
}

func TestDecisionCache_Cleanup(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	cfg := DefaultCacheConfig()
	cfg.TTL = 50 * time.Millisecond
	cfg.StaleTTL = 100 * time.Millisecond
	cfg.CleanupInterval = 200 * time.Millisecond

	cache := NewDecisionCache(cfg, logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start cleanup goroutine
	go cache.Start(ctx)

	req := &vpnv1.CheckPolicyRequest{
		Username:  "testuser",
		ClientIp:  "192.168.1.100",
		VpnIp:     "10.10.10.100",
		SessionId: "session123",
	}

	resp := &vpnv1.CheckPolicyResponse{
		Allowed: true,
	}

	cache.Set(ctx, req, resp)

	// Wait for stale expiry and cleanup
	time.Sleep(350 * time.Millisecond)

	stats := cache.Stats()
	if stats.Size != 0 {
		t.Errorf("cache size after cleanup = %d, want 0", stats.Size)
	}
}
