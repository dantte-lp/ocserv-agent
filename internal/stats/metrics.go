package stats

import (
	"context"
	"time"

	"github.com/cockroachdb/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Common errors
var (
	ErrOcctlManagerRequired = errors.New("occtl manager is required")
	ErrLoggerRequired       = errors.New("logger is required")
	ErrTracerRequired       = errors.New("tracer is required")
	ErrMeterRequired        = errors.New("meter is required")
)

// Metrics holds OpenTelemetry metrics for stats
type Metrics struct {
	// Session metrics
	activeSessions metric.Int64Gauge
	sessionsTotal  metric.Int64Counter

	// Traffic metrics
	trafficBytesRX metric.Int64Counter
	trafficBytesTX metric.Int64Counter

	// Polling metrics
	pollDuration metric.Float64Histogram
	pollErrors   metric.Int64Counter
}

// NewMetrics creates and registers OpenTelemetry metrics
func NewMetrics(meter metric.Meter) (*Metrics, error) {
	activeSessions, err := meter.Int64Gauge(
		"ocserv.sessions.active",
		metric.WithDescription("Number of active VPN sessions"),
		metric.WithUnit("{session}"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "create active sessions gauge")
	}

	sessionsTotal, err := meter.Int64Counter(
		"ocserv.sessions.total",
		metric.WithDescription("Total number of VPN sessions"),
		metric.WithUnit("{session}"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "create sessions counter")
	}

	trafficBytesRX, err := meter.Int64Counter(
		"ocserv.traffic.bytes.rx",
		metric.WithDescription("Total bytes received from clients"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "create RX traffic counter")
	}

	trafficBytesTX, err := meter.Int64Counter(
		"ocserv.traffic.bytes.tx",
		metric.WithDescription("Total bytes transmitted to clients"),
		metric.WithUnit("By"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "create TX traffic counter")
	}

	pollDuration, err := meter.Float64Histogram(
		"ocserv.stats.poll.duration",
		metric.WithDescription("Duration of stats polling operations"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5),
	)
	if err != nil {
		return nil, errors.Wrap(err, "create poll duration histogram")
	}

	pollErrors, err := meter.Int64Counter(
		"ocserv.stats.poll.errors",
		metric.WithDescription("Number of stats polling errors"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "create poll errors counter")
	}

	return &Metrics{
		activeSessions: activeSessions,
		sessionsTotal:  sessionsTotal,
		trafficBytesRX: trafficBytesRX,
		trafficBytesTX: trafficBytesTX,
		pollDuration:   pollDuration,
		pollErrors:     pollErrors,
	}, nil
}

// RecordActiveSessions records the number of active sessions
func (m *Metrics) RecordActiveSessions(ctx context.Context, count int) {
	m.activeSessions.Record(ctx, int64(count))
}

// RecordSessionConnected increments the total session counter
func (m *Metrics) RecordSessionConnected(ctx context.Context, username, groupName string) {
	m.sessionsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("username", username),
			attribute.String("group", groupName),
			attribute.String("event", "connected"),
		),
	)
}

// RecordSessionDisconnected records a session disconnection
func (m *Metrics) RecordSessionDisconnected(ctx context.Context, username, groupName string, duration time.Duration) {
	m.sessionsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("username", username),
			attribute.String("group", groupName),
			attribute.String("event", "disconnected"),
		),
	)
}

// RecordUserTraffic records traffic for a specific user
func (m *Metrics) RecordUserTraffic(ctx context.Context, username string, bytesRX, bytesTX uint64) {
	attrs := metric.WithAttributes(
		attribute.String("username", username),
	)

	m.trafficBytesRX.Add(ctx, int64(bytesRX), attrs)
	m.trafficBytesTX.Add(ctx, int64(bytesTX), attrs)
}

// RecordPollDuration records the duration of a poll operation
func (m *Metrics) RecordPollDuration(ctx context.Context, duration time.Duration) {
	m.pollDuration.Record(ctx, duration.Seconds())
}

// RecordPollError records a polling error
func (m *Metrics) RecordPollError(ctx context.Context, errorType string) {
	m.pollErrors.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("error_type", errorType),
		),
	)
}
