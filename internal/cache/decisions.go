package cache

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/zeebo/xxh3"
	vpnv1 "github.com/dantte-lp/ocserv-agent/pkg/proto/vpn/v1"
)

// DecisionCache caches authorization decisions with TTL
type DecisionCache struct {
	mu      sync.RWMutex
	entries map[uint64]*CacheEntry
	logger  *slog.Logger
	config  CacheConfig

	// Stats
	hits   uint64
	misses uint64
	stale  uint64
}

// CacheEntry represents a cached authorization decision
type CacheEntry struct {
	Response  *vpnv1.CheckPolicyResponse
	ExpiresAt time.Time
	CreatedAt time.Time
	Key       string
}

// CacheConfig configures the decision cache
type CacheConfig struct {
	// TTL is the time-to-live for cache entries
	TTL time.Duration

	// StaleTTL is the extended TTL for stale entries (used when portal is down)
	StaleTTL time.Duration

	// MaxSize is the maximum number of entries in cache (0 = unlimited)
	MaxSize int

	// CleanupInterval is how often to clean expired entries
	CleanupInterval time.Duration
}

// DefaultCacheConfig returns default cache configuration
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		TTL:             5 * time.Minute,
		StaleTTL:        30 * time.Minute,
		MaxSize:         10000,
		CleanupInterval: 10 * time.Minute,
	}
}

// NewDecisionCache creates a new decision cache
func NewDecisionCache(cfg CacheConfig, logger *slog.Logger) *DecisionCache {
	if cfg.TTL == 0 {
		cfg.TTL = 5 * time.Minute
	}
	if cfg.StaleTTL == 0 {
		cfg.StaleTTL = 30 * time.Minute
	}
	if cfg.MaxSize == 0 {
		cfg.MaxSize = 10000
	}
	if cfg.CleanupInterval == 0 {
		cfg.CleanupInterval = 10 * time.Minute
	}

	cache := &DecisionCache{
		entries: make(map[uint64]*CacheEntry),
		logger:  logger,
		config:  cfg,
	}

	return cache
}

// Start starts the background cleanup goroutine
func (c *DecisionCache) Start(ctx context.Context) {
	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			c.logger.Info("decision cache cleanup stopped")
			return
		case <-ticker.C:
			c.cleanup()
		}
	}
}

// Get retrieves a cached decision
// Returns (response, found)
func (c *DecisionCache) Get(ctx context.Context, req *vpnv1.CheckPolicyRequest) (*vpnv1.CheckPolicyResponse, bool) {
	key := c.generateKey(req)

	c.mu.RLock()
	entry, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		c.incrementMisses()
		return nil, false
	}

	now := time.Now()
	if now.After(entry.ExpiresAt) {
		c.incrementMisses()
		c.logger.Debug("cache entry expired",
			slog.String("username", req.Username),
			slog.String("client_ip", req.ClientIp),
		)
		return nil, false
	}

	c.incrementHits()
	c.logger.Debug("cache hit",
		slog.String("username", req.Username),
		slog.String("client_ip", req.ClientIp),
		slog.Duration("age", now.Sub(entry.CreatedAt)),
	)

	return entry.Response, true
}

// Set stores a decision in cache
func (c *DecisionCache) Set(ctx context.Context, req *vpnv1.CheckPolicyRequest, resp *vpnv1.CheckPolicyResponse) {
	key := c.generateKey(req)

	// Check cache size limit
	c.mu.Lock()
	if c.config.MaxSize > 0 && len(c.entries) >= c.config.MaxSize {
		// Evict oldest entry
		c.evictOldest()
	}

	entry := &CacheEntry{
		Response:  resp,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(c.config.TTL),
		Key:       c.formatKey(req),
	}

	c.entries[key] = entry
	c.mu.Unlock()

	c.logger.Debug("cache set",
		slog.String("username", req.Username),
		slog.String("client_ip", req.ClientIp),
		slog.Duration("ttl", c.config.TTL),
	)
}

// GetStale retrieves a stale cache entry (for fallback when portal is down)
// Returns (response, found)
func (c *DecisionCache) GetStale(ctx context.Context, req *vpnv1.CheckPolicyRequest) (*vpnv1.CheckPolicyResponse, bool) {
	key := c.generateKey(req)

	c.mu.RLock()
	entry, exists := c.entries[key]
	c.mu.RUnlock()

	if !exists {
		return nil, false
	}

	now := time.Now()
	staleExpiry := entry.CreatedAt.Add(c.config.StaleTTL)
	if now.After(staleExpiry) {
		c.logger.Debug("stale cache entry expired",
			slog.String("username", req.Username),
			slog.String("client_ip", req.ClientIp),
			slog.Duration("age", now.Sub(entry.CreatedAt)),
		)
		return nil, false
	}

	c.incrementStale()
	c.logger.Warn("using stale cache entry",
		slog.String("username", req.Username),
		slog.String("client_ip", req.ClientIp),
		slog.Duration("age", now.Sub(entry.CreatedAt)),
		slog.Bool("expired", now.After(entry.ExpiresAt)),
	)

	return entry.Response, true
}

// Invalidate removes an entry from cache
func (c *DecisionCache) Invalidate(ctx context.Context, req *vpnv1.CheckPolicyRequest) {
	key := c.generateKey(req)

	c.mu.Lock()
	delete(c.entries, key)
	c.mu.Unlock()

	c.logger.Debug("cache invalidated",
		slog.String("username", req.Username),
		slog.String("client_ip", req.ClientIp),
	)
}

// InvalidateUser removes all entries for a specific user
func (c *DecisionCache) InvalidateUser(ctx context.Context, username string) int {
	c.mu.Lock()
	defer c.mu.Unlock()

	count := 0
	for hash, entry := range c.entries {
		if entry.Response != nil && containsUsername(entry.Key, username) {
			delete(c.entries, hash)
			count++
		}
	}

	c.logger.Info("cache invalidated for user",
		slog.String("username", username),
		slog.Int("entries_removed", count),
	)

	return count
}

// Clear removes all entries from cache
func (c *DecisionCache) Clear() {
	c.mu.Lock()
	c.entries = make(map[uint64]*CacheEntry)
	c.hits = 0
	c.misses = 0
	c.stale = 0
	c.mu.Unlock()

	c.logger.Info("cache cleared")
}

// Stats returns cache statistics
func (c *DecisionCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return CacheStats{
		Size:    len(c.entries),
		Hits:    c.hits,
		Misses:  c.misses,
		Stale:   c.stale,
		HitRate: c.calculateHitRate(),
	}
}

// CacheStats holds cache statistics
type CacheStats struct {
	Size    int
	Hits    uint64
	Misses  uint64
	Stale   uint64
	HitRate float64
}

// cleanup removes expired entries
func (c *DecisionCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	staleExpiry := now.Add(-c.config.StaleTTL)
	removed := 0

	for hash, entry := range c.entries {
		if entry.CreatedAt.Before(staleExpiry) {
			delete(c.entries, hash)
			removed++
		}
	}

	if removed > 0 {
		c.logger.Info("cache cleanup completed",
			slog.Int("removed", removed),
			slog.Int("remaining", len(c.entries)),
		)
	}
}

// evictOldest removes the oldest entry from cache
func (c *DecisionCache) evictOldest() {
	var oldestHash uint64
	var oldestTime time.Time

	for hash, entry := range c.entries {
		if oldestTime.IsZero() || entry.CreatedAt.Before(oldestTime) {
			oldestHash = hash
			oldestTime = entry.CreatedAt
		}
	}

	if oldestHash != 0 {
		delete(c.entries, oldestHash)
		c.logger.Debug("evicted oldest cache entry",
			slog.Time("created_at", oldestTime),
		)
	}
}

// generateKey generates a hash key for the cache entry
func (c *DecisionCache) generateKey(req *vpnv1.CheckPolicyRequest) uint64 {
	keyStr := c.formatKey(req)
	return xxh3.HashString(keyStr)
}

// formatKey formats a cache key string
func (c *DecisionCache) formatKey(req *vpnv1.CheckPolicyRequest) string {
	return fmt.Sprintf("%s:%s:%s:%s",
		req.Username,
		req.Groupname,
		req.ClientIp,
		req.VpnIp,
	)
}

// containsUsername checks if a key contains a specific username
func containsUsername(key, username string) bool {
	// Simple substring check since username is the first part of the key
	return len(key) >= len(username) && key[:len(username)] == username
}

// incrementHits increments hit counter
func (c *DecisionCache) incrementHits() {
	c.mu.Lock()
	c.hits++
	c.mu.Unlock()
}

// incrementMisses increments miss counter
func (c *DecisionCache) incrementMisses() {
	c.mu.Lock()
	c.misses++
	c.mu.Unlock()
}

// incrementStale increments stale counter
func (c *DecisionCache) incrementStale() {
	c.mu.Lock()
	c.stale++
	c.mu.Unlock()
}

// calculateHitRate calculates cache hit rate
func (c *DecisionCache) calculateHitRate() float64 {
	total := c.hits + c.misses
	if total == 0 {
		return 0
	}
	return float64(c.hits) / float64(total) * 100
}
