package resilience

import (
	"context"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// CacheConfig holds cache configuration
type CacheConfig struct {
	TTL       time.Duration // normal TTL for cached entries
	StaleTTL  time.Duration // extended TTL for stale entries
	MaxSize   int           // maximum cache size
}

// DefaultCacheConfig returns default cache config
func DefaultCacheConfig() CacheConfig {
	return CacheConfig{
		TTL:      5 * time.Minute,
		StaleTTL: 30 * time.Minute,
		MaxSize:  10000,
	}
}

// CacheEntry represents a cached decision
type CacheEntry struct {
	Allowed    bool
	DenyReason string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	StaleAt    time.Time // when entry becomes stale but can still be used
	AccessCount int64
	LastAccess time.Time
}

// IsValid checks if entry is still valid (not expired)
func (e *CacheEntry) IsValid() bool {
	return time.Now().Before(e.ExpiresAt)
}

// IsStale checks if entry is stale (past normal TTL but within stale TTL)
func (e *CacheEntry) IsStale() bool {
	now := time.Now()
	return now.After(e.ExpiresAt) && now.Before(e.StaleAt)
}

// DecisionCache implements a TTL-based cache with stale entries support
type DecisionCache struct {
	config CacheConfig

	mu      sync.RWMutex
	entries map[string]*CacheEntry

	// Observability
	tracer      trace.Tracer
	hitsTotal   metric.Int64Counter
	missesTotal metric.Int64Counter
	staleHits   metric.Int64Counter
	sizeGauge   metric.Int64Gauge
}

// NewDecisionCache creates a new decision cache
func NewDecisionCache(config CacheConfig, tracer trace.Tracer, meter metric.Meter) (*DecisionCache, error) {
	hitsTotal, err := meter.Int64Counter("ocserv.cache.hits_total",
		metric.WithDescription("Total cache hits"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "create hits counter")
	}

	missesTotal, err := meter.Int64Counter("ocserv.cache.misses_total",
		metric.WithDescription("Total cache misses"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "create misses counter")
	}

	staleHits, err := meter.Int64Counter("ocserv.cache.stale_hits_total",
		metric.WithDescription("Total stale cache hits"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "create stale hits counter")
	}

	sizeGauge, err := meter.Int64Gauge("ocserv.cache.size",
		metric.WithDescription("Current cache size"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "create size gauge")
	}

	dc := &DecisionCache{
		config:      config,
		entries:     make(map[string]*CacheEntry),
		tracer:      tracer,
		hitsTotal:   hitsTotal,
		missesTotal: missesTotal,
		staleHits:   staleHits,
		sizeGauge:   sizeGauge,
	}

	// Start cleanup goroutine
	go dc.cleanupLoop()

	return dc, nil
}

// Get retrieves a cached entry
func (dc *DecisionCache) Get(ctx context.Context, key string) (*CacheEntry, bool, error) {
	_, span := dc.tracer.Start(ctx, "cache.get",
		trace.WithAttributes(
			attribute.String("key", key),
		),
	)
	defer span.End()

	dc.mu.RLock()
	entry, exists := dc.entries[key]
	dc.mu.RUnlock()

	if !exists {
		dc.missesTotal.Add(ctx, 1)
		span.SetAttributes(attribute.Bool("hit", false))
		return nil, false, nil
	}

	// Update access stats
	dc.mu.Lock()
	entry.AccessCount++
	entry.LastAccess = time.Now()
	dc.mu.Unlock()

	// Check if entry is valid
	if entry.IsValid() {
		dc.hitsTotal.Add(ctx, 1, metric.WithAttributes(
			attribute.Bool("stale", false),
		))
		span.SetAttributes(
			attribute.Bool("hit", true),
			attribute.Bool("stale", false),
		)
		return entry, true, nil
	}

	// Check if entry is stale but usable
	if entry.IsStale() {
		dc.staleHits.Add(ctx, 1)
		span.SetAttributes(
			attribute.Bool("hit", true),
			attribute.Bool("stale", true),
		)
		return entry, true, nil
	}

	// Entry expired completely
	dc.mu.Lock()
	delete(dc.entries, key)
	dc.mu.Unlock()

	dc.missesTotal.Add(ctx, 1)
	span.SetAttributes(attribute.Bool("hit", false))
	return nil, false, nil
}

// Set stores an entry in cache
func (dc *DecisionCache) Set(ctx context.Context, key string, allowed bool, denyReason string) error {
	_, span := dc.tracer.Start(ctx, "cache.set",
		trace.WithAttributes(
			attribute.String("key", key),
			attribute.Bool("allowed", allowed),
		),
	)
	defer span.End()

	dc.mu.Lock()
	defer dc.mu.Unlock()

	// Check cache size limit
	if len(dc.entries) >= dc.config.MaxSize {
		// Evict oldest entry
		dc.evictOldest()
	}

	now := time.Now()
	entry := &CacheEntry{
		Allowed:     allowed,
		DenyReason:  denyReason,
		CreatedAt:   now,
		ExpiresAt:   now.Add(dc.config.TTL),
		StaleAt:     now.Add(dc.config.StaleTTL),
		AccessCount: 0,
		LastAccess:  now,
	}

	dc.entries[key] = entry

	// Update size gauge
	dc.sizeGauge.Record(ctx, int64(len(dc.entries)))

	return nil
}

// Delete removes an entry from cache
func (dc *DecisionCache) Delete(ctx context.Context, key string) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	delete(dc.entries, key)

	// Update size gauge
	dc.sizeGauge.Record(ctx, int64(len(dc.entries)))
}

// Clear removes all entries from cache
func (dc *DecisionCache) Clear(ctx context.Context) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	dc.entries = make(map[string]*CacheEntry)

	// Update size gauge
	dc.sizeGauge.Record(ctx, 0)
}

// evictOldest evicts the oldest entry based on last access time
func (dc *DecisionCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range dc.entries {
		if oldestKey == "" || entry.LastAccess.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.LastAccess
		}
	}

	if oldestKey != "" {
		delete(dc.entries, oldestKey)
	}
}

// cleanupLoop periodically removes expired entries
func (dc *DecisionCache) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		dc.cleanup()
	}
}

// cleanup removes all expired entries
func (dc *DecisionCache) cleanup() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	now := time.Now()
	for key, entry := range dc.entries {
		// Remove entries past stale TTL
		if now.After(entry.StaleAt) {
			delete(dc.entries, key)
		}
	}
}

// Stats returns cache statistics
func (dc *DecisionCache) Stats() map[string]interface{} {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	var validEntries, staleEntries, expiredEntries int
	now := time.Now()

	for _, entry := range dc.entries {
		if entry.IsValid() {
			validEntries++
		} else if entry.IsStale() {
			staleEntries++
		} else {
			expiredEntries++
		}
	}

	return map[string]interface{}{
		"total_entries":   len(dc.entries),
		"valid_entries":   validEntries,
		"stale_entries":   staleEntries,
		"expired_entries": expiredEntries,
		"max_size":        dc.config.MaxSize,
	}
}

// Size returns current cache size
func (dc *DecisionCache) Size() int {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return len(dc.entries)
}
