package metrics

import (
	"context"
	"runtime"
	"time"

	"github.com/cockroachdb/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Metrics содержит все метрики ocserv-agent.
type Metrics struct {
	// Бизнес-метрики
	CommandsTotal   metric.Int64Counter
	CommandDuration metric.Float64Histogram
	ActiveSessions  metric.Int64UpDownCounter
	ConnectedUsers  metric.Int64UpDownCounter
	CommandErrors   metric.Int64Counter

	// gRPC метрики
	GRPCRequestsTotal   metric.Int64Counter
	GRPCRequestDuration metric.Float64Histogram

	// Runtime метрики (Go)
	GoRoutines       metric.Int64ObservableGauge
	MemoryAlloc      metric.Int64ObservableGauge
	MemoryTotalAlloc metric.Int64ObservableGauge
	MemorySys        metric.Int64ObservableGauge
	GCCount          metric.Int64ObservableCounter
}

// New создает новый набор метрик.
func New(meter metric.Meter) (*Metrics, error) {
	m := &Metrics{}

	var err error

	// Бизнес-метрики: Команды
	m.CommandsTotal, err = meter.Int64Counter(
		"ocserv_agent_commands_total",
		metric.WithDescription("Общее количество выполненных команд"),
		metric.WithUnit("{command}"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create commands_total counter")
	}

	m.CommandDuration, err = meter.Float64Histogram(
		"ocserv_agent_command_duration_seconds",
		metric.WithDescription("Длительность выполнения команд"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create command_duration histogram")
	}

	m.CommandErrors, err = meter.Int64Counter(
		"ocserv_agent_command_errors_total",
		metric.WithDescription("Количество ошибок при выполнении команд"),
		metric.WithUnit("{error}"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create command_errors counter")
	}

	// Бизнес-метрики: VPN сессии
	m.ActiveSessions, err = meter.Int64UpDownCounter(
		"ocserv_agent_active_sessions",
		metric.WithDescription("Текущее количество активных VPN сессий"),
		metric.WithUnit("{session}"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create active_sessions counter")
	}

	m.ConnectedUsers, err = meter.Int64UpDownCounter(
		"ocserv_agent_connected_users",
		metric.WithDescription("Текущее количество подключенных пользователей"),
		metric.WithUnit("{user}"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create connected_users counter")
	}

	// gRPC метрики
	m.GRPCRequestsTotal, err = meter.Int64Counter(
		"grpc_server_requests_total",
		metric.WithDescription("Общее количество gRPC запросов"),
		metric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create grpc_requests_total counter")
	}

	m.GRPCRequestDuration, err = meter.Float64Histogram(
		"grpc_server_request_duration_seconds",
		metric.WithDescription("Длительность обработки gRPC запросов"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create grpc_request_duration histogram")
	}

	// Runtime метрики (Go)
	m.GoRoutines, err = meter.Int64ObservableGauge(
		"go_goroutines",
		metric.WithDescription("Текущее количество goroutines"),
		metric.WithUnit("{goroutine}"),
		metric.WithInt64Callback(func(_ context.Context, observer metric.Int64Observer) error {
			observer.Observe(int64(runtime.NumGoroutine()))
			return nil
		}),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create go_goroutines gauge")
	}

	m.MemoryAlloc, err = meter.Int64ObservableGauge(
		"go_memory_alloc_bytes",
		metric.WithDescription("Количество выделенной памяти"),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(_ context.Context, observer metric.Int64Observer) error {
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			// #nosec G115 -- memory size unlikely to exceed int64 max (9 exabytes)
			observer.Observe(int64(ms.Alloc))
			return nil
		}),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create go_memory_alloc gauge")
	}

	m.MemorySys, err = meter.Int64ObservableGauge(
		"go_memory_sys_bytes",
		metric.WithDescription("Количество памяти полученной от OS"),
		metric.WithUnit("By"),
		metric.WithInt64Callback(func(_ context.Context, observer metric.Int64Observer) error {
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			// #nosec G115 -- memory size unlikely to exceed int64 max (9 exabytes)
			observer.Observe(int64(ms.Sys))
			return nil
		}),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create go_memory_sys gauge")
	}

	return m, nil
}

// RecordCommand записывает метрики выполнения команды.
func (m *Metrics) RecordCommand(ctx context.Context, commandType string, args []string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
		m.CommandErrors.Add(ctx, 1,
			metric.WithAttributes(
				attribute.String("command_type", commandType),
				attribute.StringSlice("args", args),
			),
		)
	}

	m.CommandsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("command_type", commandType),
			attribute.String("status", status),
		),
	)

	m.CommandDuration.Record(ctx, duration.Seconds(),
		metric.WithAttributes(
			attribute.String("command_type", commandType),
			attribute.String("status", status),
		),
	)
}

// RecordGRPCRequest записывает метрики gRPC запроса.
func (m *Metrics) RecordGRPCRequest(ctx context.Context, method string, duration time.Duration, err error) {
	status := "ok"
	if err != nil {
		status = "error"
	}

	m.GRPCRequestsTotal.Add(ctx, 1,
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("status", status),
		),
	)

	m.GRPCRequestDuration.Record(ctx, duration.Seconds(),
		metric.WithAttributes(
			attribute.String("method", method),
			attribute.String("status", status),
		),
	)
}

// UpdateActiveSessions обновляет количество активных сессий.
func (m *Metrics) UpdateActiveSessions(ctx context.Context, count int64) {
	m.ActiveSessions.Add(ctx, count)
}

// UpdateConnectedUsers обновляет количество подключенных пользователей.
func (m *Metrics) UpdateConnectedUsers(ctx context.Context, count int64) {
	m.ConnectedUsers.Add(ctx, count)
}
