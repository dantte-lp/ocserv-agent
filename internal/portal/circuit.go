package portal

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// CircuitState represents the state of the circuit breaker
type CircuitState int

const (
	// StateClosed - circuit is closed, requests are allowed
	StateClosed CircuitState = iota
	// StateOpen - circuit is open, requests are rejected
	StateOpen
	// StateHalfOpen - circuit is half-open, testing if service recovered
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when circuit is open
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitBreakerConfig configures circuit breaker behavior
type CircuitBreakerConfig struct {
	// MaxRequests is the maximum number of requests allowed to pass through
	// when the circuit is half-open (default: 1)
	MaxRequests uint32

	// Interval is the cyclic period of the closed state for the circuit breaker
	// to clear the internal counts. If interval is 0, the circuit breaker doesn't clear
	// the internal counts during the closed state (default: 60s)
	Interval time.Duration

	// Timeout is the period of the open state after which the state becomes half-open (default: 30s)
	Timeout time.Duration

	// FailureThreshold is the maximum number of failures allowed in closed state
	// before the circuit opens (default: 5)
	FailureThreshold uint32

	// OnStateChange is called when the circuit breaker changes state
	OnStateChange func(from, to CircuitState)
}

// DefaultCircuitBreakerConfig returns default circuit breaker configuration
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxRequests:      1,
		Interval:         60 * time.Second,
		Timeout:          30 * time.Second,
		FailureThreshold: 5,
	}
}

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	config CircuitBreakerConfig
	logger *slog.Logger

	mu           sync.RWMutex
	state        CircuitState
	generation   uint64
	lastFailTime time.Time
	lastSuccTime time.Time
	counts       Counts
	expiry       time.Time
}

// Counts holds counters for circuit breaker
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(cfg CircuitBreakerConfig, logger *slog.Logger) *CircuitBreaker {
	if cfg.MaxRequests == 0 {
		cfg.MaxRequests = 1
	}
	if cfg.Interval == 0 {
		cfg.Interval = 60 * time.Second
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.FailureThreshold == 0 {
		cfg.FailureThreshold = 5
	}

	return &CircuitBreaker{
		config: cfg,
		logger: logger,
		state:  StateClosed,
	}
}

// Execute runs the given function if the circuit breaker allows it
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	generation, err := cb.beforeRequest()
	if err != nil {
		return err
	}

	defer func() {
		if r := recover(); r != nil {
			cb.afterRequest(generation, false)
			panic(r)
		}
	}()

	err = fn(ctx)
	cb.afterRequest(generation, err == nil)
	return err
}

// State returns current circuit breaker state
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Counts returns current counts
func (cb *CircuitBreaker) Counts() Counts {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.counts
}

// beforeRequest checks if request is allowed
func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		return generation, ErrCircuitOpen
	}

	if state == StateHalfOpen {
		if cb.counts.Requests >= cb.config.MaxRequests {
			return generation, ErrCircuitOpen
		}
	}

	cb.counts.Requests++
	return generation, nil
}

// afterRequest records the result of a request
func (cb *CircuitBreaker) afterRequest(generation uint64, success bool) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()
	state, currentGen := cb.currentState(now)
	if generation != currentGen {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

// currentState returns the current state and generation
func (cb *CircuitBreaker) currentState(now time.Time) (CircuitState, uint64) {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

// onSuccess handles successful request
func (cb *CircuitBreaker) onSuccess(state CircuitState, now time.Time) {
	cb.counts.TotalSuccesses++
	cb.counts.ConsecutiveSuccesses++
	cb.counts.ConsecutiveFailures = 0
	cb.lastSuccTime = now

	switch state {
	case StateHalfOpen:
		if cb.counts.ConsecutiveSuccesses >= cb.config.MaxRequests {
			cb.setState(StateClosed, now)
			cb.logger.Info("circuit breaker recovered",
				slog.String("state", "closed"),
				slog.Uint64("consecutive_successes", uint64(cb.counts.ConsecutiveSuccesses)),
			)
		}
	}
}

// onFailure handles failed request
func (cb *CircuitBreaker) onFailure(state CircuitState, now time.Time) {
	cb.counts.TotalFailures++
	cb.counts.ConsecutiveFailures++
	cb.counts.ConsecutiveSuccesses = 0
	cb.lastFailTime = now

	switch state {
	case StateClosed:
		if cb.counts.ConsecutiveFailures >= cb.config.FailureThreshold {
			cb.setState(StateOpen, now)
			cb.logger.Warn("circuit breaker opened",
				slog.String("state", "open"),
				slog.Uint64("consecutive_failures", uint64(cb.counts.ConsecutiveFailures)),
				slog.Duration("timeout", cb.config.Timeout),
			)
		}
	case StateHalfOpen:
		cb.setState(StateOpen, now)
		cb.logger.Warn("circuit breaker re-opened",
			slog.String("state", "open"),
			slog.String("reason", "failure_in_half_open"),
		)
	}
}

// setState changes the state of the circuit breaker
func (cb *CircuitBreaker) setState(state CircuitState, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.toNewGeneration(now)

	if cb.config.OnStateChange != nil {
		cb.config.OnStateChange(prev, state)
	}

	switch state {
	case StateOpen:
		cb.expiry = now.Add(cb.config.Timeout)
	case StateHalfOpen:
		cb.expiry = time.Time{}
	}
}

// toNewGeneration resets counts and increments generation
func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts = Counts{}

	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.config.Interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.config.Interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.config.Timeout)
	default:
		cb.expiry = zero
	}
}

// Reset resets the circuit breaker to initial state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.generation = 0
	cb.counts = Counts{}
	cb.expiry = time.Time{}

	cb.logger.Info("circuit breaker reset")
}

// Stats returns current statistics
func (cb *CircuitBreaker) Stats() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return fmt.Sprintf("state=%s generation=%d requests=%d successes=%d failures=%d consecutive_failures=%d",
		cb.state,
		cb.generation,
		cb.counts.Requests,
		cb.counts.TotalSuccesses,
		cb.counts.TotalFailures,
		cb.counts.ConsecutiveFailures,
	)
}
