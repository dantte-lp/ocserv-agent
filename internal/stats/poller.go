package stats

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/dantte-lp/ocserv-agent/internal/ocserv"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// SessionInfo represents a VPN session for tracking
type SessionInfo struct {
	ID          int
	Username    string
	GroupName   string
	ClientIP    string
	VPNIP       string
	ConnectedAt time.Time
	BytesRX     uint64
	BytesTX     uint64
}

// SessionEventType defines the type of session event
type SessionEventType string

const (
	SessionConnected    SessionEventType = "connected"
	SessionDisconnected SessionEventType = "disconnected"
	SessionUpdated      SessionEventType = "updated"
)

// SessionEvent represents a session state change
type SessionEvent struct {
	Type    SessionEventType
	Session SessionInfo
}

// SessionCallback is called when session events occur
type SessionCallback func(ctx context.Context, event SessionEvent)

// Poller polls ocserv for active sessions and metrics
type Poller struct {
	occtl    *ocserv.OcctlManager
	logger   *slog.Logger
	tracer   trace.Tracer
	metrics  *Metrics
	interval time.Duration

	// Session tracking
	sessions  map[int]*SessionInfo
	callbacks []SessionCallback
	mu        sync.RWMutex

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// PollerConfig configures the stats poller
type PollerConfig struct {
	OcctlManager *ocserv.OcctlManager
	Logger       *slog.Logger
	Tracer       trace.Tracer
	Meter        metric.Meter
	Interval     time.Duration
}

// NewPoller creates a new stats poller
func NewPoller(cfg *PollerConfig) (*Poller, error) {
	if cfg.OcctlManager == nil {
		return nil, ErrOcctlManagerRequired
	}
	if cfg.Logger == nil {
		return nil, ErrLoggerRequired
	}
	if cfg.Tracer == nil {
		return nil, ErrTracerRequired
	}
	if cfg.Meter == nil {
		return nil, ErrMeterRequired
	}
	if cfg.Interval == 0 {
		cfg.Interval = 10 * time.Second
	}

	// Initialize metrics
	metrics, err := NewMetrics(cfg.Meter)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Poller{
		occtl:     cfg.OcctlManager,
		logger:    cfg.Logger,
		tracer:    cfg.Tracer,
		metrics:   metrics,
		interval:  cfg.Interval,
		sessions:  make(map[int]*SessionInfo),
		callbacks: make([]SessionCallback, 0),
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

// Start starts the polling loop
func (p *Poller) Start(ctx context.Context) error {
	ctx, span := p.tracer.Start(ctx, "stats.poller.start",
		trace.WithAttributes(
			attribute.String("interval", p.interval.String()),
		),
	)
	defer span.End()

	p.logger.InfoContext(ctx, "starting stats poller",
		slog.Duration("interval", p.interval),
	)

	// Start polling loop
	p.wg.Add(1)
	go p.pollLoop()

	return nil
}

// Stop stops the polling loop
func (p *Poller) Stop(ctx context.Context) error {
	ctx, span := p.tracer.Start(ctx, "stats.poller.stop")
	defer span.End()

	p.logger.InfoContext(ctx, "stopping stats poller")

	// Cancel context to signal shutdown
	p.cancel()

	// Wait for polling loop to finish with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		p.logger.InfoContext(ctx, "stats poller stopped gracefully")
	case <-time.After(10 * time.Second):
		p.logger.WarnContext(ctx, "stats poller shutdown timeout")
	}

	return nil
}

// RegisterCallback registers a callback for session events
func (p *Poller) RegisterCallback(cb SessionCallback) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.callbacks = append(p.callbacks, cb)
}

// GetActiveSessions returns a snapshot of active sessions
func (p *Poller) GetActiveSessions() []SessionInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()

	sessions := make([]SessionInfo, 0, len(p.sessions))
	for _, s := range p.sessions {
		sessions = append(sessions, *s)
	}
	return sessions
}

// pollLoop is the main polling loop
func (p *Poller) pollLoop() {
	defer p.wg.Done()

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	// Initial poll
	p.poll()

	for {
		select {
		case <-p.ctx.Done():
			return
		case <-ticker.C:
			p.poll()
		}
	}
}

// poll performs a single poll operation
func (p *Poller) poll() {
	ctx, span := p.tracer.Start(p.ctx, "stats.poller.poll")
	defer span.End()

	start := time.Now()

	// Get current users from occtl
	users, err := p.occtl.ShowUsers(ctx)
	if err != nil {
		p.logger.ErrorContext(ctx, "failed to get users from occtl",
			slog.String("error", err.Error()),
		)
		p.metrics.RecordPollError(ctx, "show_users")
		return
	}

	// Record poll duration
	duration := time.Since(start)
	p.metrics.RecordPollDuration(ctx, duration)

	// Process users and detect changes
	p.reconcileSessions(ctx, users)

	// Update metrics
	p.updateMetrics(ctx, users)
}

// reconcileSessions compares current users with tracked sessions
func (p *Poller) reconcileSessions(ctx context.Context, users []ocserv.User) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create map of current user IDs
	currentIDs := make(map[int]bool)

	// Process current users
	for _, user := range users {
		currentIDs[user.ID] = true

		// Check if this is a new session
		if existing, ok := p.sessions[user.ID]; !ok {
			// New session
			session := p.userToSession(user)
			p.sessions[user.ID] = &session

			p.logger.InfoContext(ctx, "new session detected",
				slog.Int("id", user.ID),
				slog.String("username", user.Username),
				slog.String("client_ip", user.RemoteIP),
				slog.String("vpn_ip", user.IPv4),
			)

			// Emit event
			p.emitEvent(ctx, SessionEvent{
				Type:    SessionConnected,
				Session: session,
			})
		} else {
			// Existing session - check for updates
			session := p.userToSession(user)

			// Check if traffic stats changed significantly
			if session.BytesRX != existing.BytesRX || session.BytesTX != existing.BytesTX {
				p.sessions[user.ID] = &session

				// Emit update event
				p.emitEvent(ctx, SessionEvent{
					Type:    SessionUpdated,
					Session: session,
				})
			}
		}
	}

	// Find disconnected sessions
	for id, session := range p.sessions {
		if !currentIDs[id] {
			p.logger.InfoContext(ctx, "session disconnected",
				slog.Int("id", id),
				slog.String("username", session.Username),
				slog.Duration("duration", time.Since(session.ConnectedAt)),
			)

			// Emit event
			p.emitEvent(ctx, SessionEvent{
				Type:    SessionDisconnected,
				Session: *session,
			})

			// Remove from tracking
			delete(p.sessions, id)
		}
	}
}

// userToSession converts ocserv.User to SessionInfo
func (p *Poller) userToSession(user ocserv.User) SessionInfo {
	return SessionInfo{
		ID:          user.ID,
		Username:    user.Username,
		GroupName:   user.Groupname,
		ClientIP:    user.RemoteIP,
		VPNIP:       user.IPv4,
		ConnectedAt: time.Unix(user.RawConnectedAt, 0),
		BytesRX:     parseBytes(user.RX),
		BytesTX:     parseBytes(user.TX),
	}
}

// emitEvent calls all registered callbacks
func (p *Poller) emitEvent(ctx context.Context, event SessionEvent) {
	for _, cb := range p.callbacks {
		go func(callback SessionCallback) {
			defer func() {
				if r := recover(); r != nil {
					p.logger.ErrorContext(ctx, "callback panic",
						slog.Any("panic", r),
					)
				}
			}()
			callback(ctx, event)
		}(cb)
	}
}

// updateMetrics updates OpenTelemetry metrics
func (p *Poller) updateMetrics(ctx context.Context, users []ocserv.User) {
	// Record active sessions count
	p.metrics.RecordActiveSessions(ctx, len(users))

	// Aggregate traffic by user
	userTraffic := make(map[string]struct{ rx, tx uint64 })
	for _, user := range users {
		key := user.Username
		traffic := userTraffic[key]
		traffic.rx += parseBytes(user.RX)
		traffic.tx += parseBytes(user.TX)
		userTraffic[key] = traffic
	}

	// Record per-user traffic
	for username, traffic := range userTraffic {
		p.metrics.RecordUserTraffic(ctx, username, traffic.rx, traffic.tx)
	}
}

// parseBytes parses byte count from occtl output
// Format: "123456" (bytes as string)
func parseBytes(s string) uint64 {
	// occtl returns bytes as string, sometimes with units
	// For now, assume simple integer string
	var bytes uint64
	fmt.Sscanf(s, "%d", &bytes)
	return bytes
}
