package resilience

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// State represents circuit breaker state
type State int

const (
	StateClosed State = iota
	StateOpen
	StateHalfOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

// Config holds circuit breaker configuration
type Config struct {
	MaxRequests      uint32        // max requests in half-open state
	Interval         time.Duration // interval to reset failure counter
	Timeout          time.Duration // timeout in open state
	FailureThreshold uint32        // failures to open circuit
}

// DefaultConfig returns default circuit breaker config
func DefaultConfig() Config {
	return Config{
		MaxRequests:      5,
		Interval:         30 * time.Second,
		Timeout:          60 * time.Second,
		FailureThreshold: 3,
	}
}

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	config Config

	mu             sync.RWMutex
	state          State
	failures       uint32
	requests       uint32
	lastFailure    time.Time
	lastStateChange time.Time

	// Observability
	tracer       trace.Tracer
	stateGauge   metric.Int64Gauge
	requestsTotal metric.Int64Counter
	failuresTotal metric.Int64Counter
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config Config, tracer trace.Tracer, meter metric.Meter) (*CircuitBreaker, error) {
	stateGauge, err := meter.Int64Gauge("ocserv.circuit_breaker.state",
		metric.WithDescription("Circuit breaker state (0=closed, 1=open, 2=half-open)"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "create state gauge")
	}

	requestsTotal, err := meter.Int64Counter("ocserv.circuit_breaker.requests_total",
		metric.WithDescription("Total circuit breaker requests"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "create requests counter")
	}

	failuresTotal, err := meter.Int64Counter("ocserv.circuit_breaker.failures_total",
		metric.WithDescription("Total circuit breaker failures"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "create failures counter")
	}

	cb := &CircuitBreaker{
		config:        config,
		state:         StateClosed,
		tracer:        tracer,
		stateGauge:    stateGauge,
		requestsTotal: requestsTotal,
		failuresTotal: failuresTotal,
	}

	// Initialize state gauge
	cb.recordState(context.Background())

	return cb, nil
}

// Execute wraps a function call with circuit breaker logic
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	ctx, span := cb.tracer.Start(ctx, "circuit_breaker.execute",
		trace.WithAttributes(
			attribute.String("state", cb.State().String()),
		),
	)
	defer span.End()

	// Record request
	cb.requestsTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("state", cb.State().String()),
	))

	// Check if circuit is open
	if !cb.canExecute() {
		err := errors.Newf("circuit breaker is %s", cb.State())
		span.RecordError(err)
		return err
	}

	// Execute function
	err := fn(ctx)

	// Record result
	if err != nil {
		cb.recordFailure(ctx)
		span.RecordError(err)
	} else {
		cb.recordSuccess(ctx)
	}

	return err
}

// State returns current circuit breaker state
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// canExecute checks if request can be executed
func (cb *CircuitBreaker) canExecute() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	now := time.Now()

	switch cb.state {
	case StateClosed:
		// Reset counter if interval passed
		if now.Sub(cb.lastFailure) > cb.config.Interval {
			cb.failures = 0
		}
		return true

	case StateOpen:
		// Transition to half-open if timeout passed
		if now.Sub(cb.lastStateChange) > cb.config.Timeout {
			cb.setState(StateHalfOpen)
			cb.requests = 0
			return true
		}
		return false

	case StateHalfOpen:
		// Allow limited requests in half-open state
		if cb.requests < cb.config.MaxRequests {
			cb.requests++
			return true
		}
		return false

	default:
		return false
	}
}

// recordSuccess records successful execution
func (cb *CircuitBreaker) recordSuccess(ctx context.Context) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		// Transition to closed after successful requests
		if cb.requests >= cb.config.MaxRequests {
			cb.setState(StateClosed)
			cb.failures = 0
			cb.recordState(ctx)
		}
	} else if cb.state == StateClosed {
		// Reset failure counter on success
		cb.failures = 0
	}
}

// recordFailure records failed execution
func (cb *CircuitBreaker) recordFailure(ctx context.Context) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	// Record failure metric
	cb.failuresTotal.Add(ctx, 1, metric.WithAttributes(
		attribute.String("state", cb.state.String()),
	))

	if cb.state == StateHalfOpen {
		// Immediately open on failure in half-open
		cb.setState(StateOpen)
		cb.recordState(ctx)
	} else if cb.state == StateClosed && cb.failures >= cb.config.FailureThreshold {
		// Open circuit if threshold exceeded
		cb.setState(StateOpen)
		cb.recordState(ctx)
	}
}

// setState changes circuit breaker state
func (cb *CircuitBreaker) setState(state State) {
	cb.state = state
	cb.lastStateChange = time.Now()
}

// recordState records current state to metrics
func (cb *CircuitBreaker) recordState(ctx context.Context) {
	var stateValue int64
	switch cb.state {
	case StateClosed:
		stateValue = 0
	case StateOpen:
		stateValue = 1
	case StateHalfOpen:
		stateValue = 2
	}

	cb.stateGauge.Record(ctx, stateValue)
}

// Stats returns current circuit breaker statistics
func (cb *CircuitBreaker) Stats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"state":             cb.state.String(),
		"failures":          cb.failures,
		"requests":          cb.requests,
		"last_failure":      cb.lastFailure,
		"last_state_change": cb.lastStateChange,
	}
}

// Reset resets circuit breaker to closed state
func (cb *CircuitBreaker) Reset(ctx context.Context) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.setState(StateClosed)
	cb.failures = 0
	cb.requests = 0
	cb.recordState(ctx)
}

// String returns circuit breaker status string
func (cb *CircuitBreaker) String() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return fmt.Sprintf("CircuitBreaker[state=%s, failures=%d, requests=%d]",
		cb.state, cb.failures, cb.requests)
}
