// +build e2e

package e2e_test

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/dantte-lp/ocserv-agent/pkg/proto/agent/v1"
)

const (
	// Параметры нагрузочного тестирования
	concurrentConnections = 100
	requestsPerConnection = 10
)

// LoadTestSuite содержит нагрузочные тесты
type LoadTestSuite struct {
	suite.Suite
	ctx        context.Context
	grpcClient pb.VPNAgentServiceClient
	grpcConn   *grpc.ClientConn
}

// LatencyStats содержит статистику по latency
type LatencyStats struct {
	samples  []time.Duration
	p50      time.Duration
	p95      time.Duration
	p99      time.Duration
	mean     time.Duration
	min      time.Duration
	max      time.Duration
	totalOps int64
}

// SetupSuite выполняется один раз перед всеми тестами
func (s *LoadTestSuite) SetupSuite() {
	s.ctx = context.Background()

	// Подключение к gRPC серверу агента
	s.T().Log("Connecting to agent gRPC server for load testing...")
	conn, err := grpc.Dial(
		agentGRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithTimeout(10*time.Second),
	)
	require.NoError(s.T(), err, "Failed to connect to agent gRPC server")
	s.grpcConn = conn
	s.grpcClient = pb.NewVPNAgentServiceClient(conn)

	s.T().Logf("Connected to agent at %s", agentGRPCAddr)
}

// TearDownSuite выполняется один раз после всех тестов
func (s *LoadTestSuite) TearDownSuite() {
	if s.grpcConn != nil {
		_ = s.grpcConn.Close()
	}
}

// TestLoad_ConcurrentConnections тестирует 100 одновременных подключений
func (s *LoadTestSuite) TestLoad_ConcurrentConnections() {
	t := s.T()
	ctx, cancel := context.WithTimeout(s.ctx, 120*time.Second)
	defer cancel()

	// Начальное состояние памяти
	var memStatsBefore runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)
	initialGoroutines := runtime.NumGoroutine()

	t.Logf("Initial state: Goroutines=%d, HeapAlloc=%d MB",
		initialGoroutines, memStatsBefore.HeapAlloc/1024/1024)

	var (
		wg              sync.WaitGroup
		successCount    int64
		errorCount      int64
		latencies       sync.Map // map[int][]time.Duration
		totalOperations int64
	)

	startTime := time.Now()

	// Запуск concurrent connections
	for i := 0; i < concurrentConnections; i++ {
		wg.Add(1)
		go func(connID int) {
			defer wg.Done()

			sessionID := fmt.Sprintf("load_session_%d_%d", connID, time.Now().Unix())
			username := fmt.Sprintf("load_user_%d", connID)

			// Connect
			connectStart := time.Now()
			connectReq := &pb.NotifyConnectRequest{
				Username:  username,
				ClientIp:  fmt.Sprintf("192.168.200.%d", connID%256),
				VpnIp:     fmt.Sprintf("10.20.10.%d", connID%256),
				SessionId: sessionID,
				DeviceId:  fmt.Sprintf("device_%d", connID),
			}

			connectResp, err := s.grpcClient.NotifyConnect(ctx, connectReq)
			connectLatency := time.Since(connectStart)

			if err != nil {
				atomic.AddInt64(&errorCount, 1)
				return
			}

			if !connectResp.Allowed {
				atomic.AddInt64(&errorCount, 1)
				return
			}

			atomic.AddInt64(&successCount, 1)
			atomic.AddInt64(&totalOperations, 1)

			// Store latency
			connLatencies := make([]time.Duration, 0, requestsPerConnection)
			connLatencies = append(connLatencies, connectLatency)

			// Выполнить несколько операций
			for j := 0; j < requestsPerConnection-1; j++ {
				// GetActiveSessions
				sessionsStart := time.Now()
				_, err := s.grpcClient.GetActiveSessions(ctx, &pb.GetActiveSessionsRequest{})
				sessionsLatency := time.Since(sessionsStart)

				connLatencies = append(connLatencies, sessionsLatency)
				atomic.AddInt64(&totalOperations, 1)

				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				}

				// Small delay between requests
				time.Sleep(10 * time.Millisecond)
			}

			// Disconnect
			disconnectStart := time.Now()
			_, err = s.grpcClient.NotifyDisconnect(ctx, &pb.NotifyDisconnectRequest{
				Username:  username,
				SessionId: sessionID,
			})
			disconnectLatency := time.Since(disconnectStart)
			connLatencies = append(connLatencies, disconnectLatency)
			atomic.AddInt64(&totalOperations, 1)

			if err != nil {
				atomic.AddInt64(&errorCount, 1)
			}

			latencies.Store(connID, connLatencies)
		}(i)
	}

	// Ожидание завершения всех goroutines
	wg.Wait()

	totalDuration := time.Since(startTime)

	// Сбор статистики
	allLatencies := make([]time.Duration, 0, totalOperations)
	latencies.Range(func(key, value interface{}) bool {
		connLatencies := value.([]time.Duration)
		allLatencies = append(allLatencies, connLatencies...)
		return true
	})

	stats := calculateLatencyStats(allLatencies)

	// Конечное состояние памяти
	var memStatsAfter runtime.MemStats
	runtime.GC() // Force GC before measurement
	time.Sleep(1 * time.Second)
	runtime.ReadMemStats(&memStatsAfter)
	finalGoroutines := runtime.NumGoroutine()

	// Результаты
	t.Logf("\n=== Load Test Results ===")
	t.Logf("Total duration: %v", totalDuration)
	t.Logf("Concurrent connections: %d", concurrentConnections)
	t.Logf("Requests per connection: %d", requestsPerConnection)
	t.Logf("Total operations: %d", totalOperations)
	t.Logf("Successful operations: %d", successCount)
	t.Logf("Failed operations: %d", errorCount)
	t.Logf("Success rate: %.2f%%", float64(successCount)/float64(totalOperations)*100)
	t.Logf("Throughput: %.2f ops/sec", float64(totalOperations)/totalDuration.Seconds())

	t.Logf("\n=== Latency Statistics ===")
	t.Logf("Min latency: %v", stats.min)
	t.Logf("Mean latency: %v", stats.mean)
	t.Logf("p50 latency: %v", stats.p50)
	t.Logf("p95 latency: %v", stats.p95)
	t.Logf("p99 latency: %v", stats.p99)
	t.Logf("Max latency: %v", stats.max)

	t.Logf("\n=== Memory Statistics ===")
	t.Logf("Initial HeapAlloc: %d MB", memStatsBefore.HeapAlloc/1024/1024)
	t.Logf("Final HeapAlloc: %d MB", memStatsAfter.HeapAlloc/1024/1024)
	t.Logf("HeapAlloc Delta: %d MB", (int64(memStatsAfter.HeapAlloc)-int64(memStatsBefore.HeapAlloc))/1024/1024)
	t.Logf("Total Allocated: %d MB", memStatsAfter.TotalAlloc/1024/1024)

	t.Logf("\n=== Goroutine Statistics ===")
	t.Logf("Initial Goroutines: %d", initialGoroutines)
	t.Logf("Final Goroutines: %d", finalGoroutines)
	t.Logf("Goroutine Delta: %d", finalGoroutines-initialGoroutines)

	// Assertions
	successRate := float64(successCount) / float64(totalOperations) * 100
	assert.GreaterOrEqual(t, successRate, 95.0, "Success rate should be at least 95%")

	// p99 latency должен быть разумным (например, < 500ms для local testing)
	assert.Less(t, stats.p99, 1*time.Second, "p99 latency should be < 1s")

	// Memory leak detection: heap не должен вырасти более чем на 100MB
	heapDelta := (int64(memStatsAfter.HeapAlloc) - int64(memStatsBefore.HeapAlloc)) / 1024 / 1024
	assert.Less(t, heapDelta, int64(100), "Heap allocation delta should be < 100MB")

	// Goroutine leak detection: не должно быть значительного роста
	goroutineDelta := finalGoroutines - initialGoroutines
	assert.Less(t, goroutineDelta, 10, "Goroutine count should not grow significantly")
}

// TestLoad_HighFrequencyUpdates тестирует частые обновления маршрутов
func (s *LoadTestSuite) TestLoad_HighFrequencyUpdates() {
	t := s.T()
	ctx, cancel := context.WithTimeout(s.ctx, 60*time.Second)
	defer cancel()

	username := "load_update_user"
	updateCount := 100
	var successCount int64
	var errorCount int64

	latencies := make([]time.Duration, 0, updateCount)

	startTime := time.Now()

	for i := 0; i < updateCount; i++ {
		updateStart := time.Now()

		routesReq := &pb.UpdateUserRoutesRequest{
			Username: username,
			Routes: []string{
				fmt.Sprintf("10.%d.0.0/16", i%256),
				fmt.Sprintf("172.%d.0.0/16", i%256),
			},
			DnsServers: []string{
				fmt.Sprintf("10.0.0.%d", i%256),
			},
		}

		resp, err := s.grpcClient.UpdateUserRoutes(ctx, routesReq)
		updateLatency := time.Since(updateStart)
		latencies = append(latencies, updateLatency)

		if err != nil || !resp.Success {
			atomic.AddInt64(&errorCount, 1)
		} else {
			atomic.AddInt64(&successCount, 1)
		}

		// Small delay between updates
		time.Sleep(10 * time.Millisecond)
	}

	totalDuration := time.Since(startTime)
	stats := calculateLatencyStats(latencies)

	t.Logf("\n=== High Frequency Update Test Results ===")
	t.Logf("Total updates: %d", updateCount)
	t.Logf("Total duration: %v", totalDuration)
	t.Logf("Successful updates: %d", successCount)
	t.Logf("Failed updates: %d", errorCount)
	t.Logf("Success rate: %.2f%%", float64(successCount)/float64(updateCount)*100)
	t.Logf("Throughput: %.2f updates/sec", float64(updateCount)/totalDuration.Seconds())

	t.Logf("\n=== Update Latency Statistics ===")
	t.Logf("Min latency: %v", stats.min)
	t.Logf("Mean latency: %v", stats.mean)
	t.Logf("p50 latency: %v", stats.p50)
	t.Logf("p95 latency: %v", stats.p95)
	t.Logf("p99 latency: %v", stats.p99)
	t.Logf("Max latency: %v", stats.max)

	// Assertions
	successRate := float64(successCount) / float64(updateCount) * 100
	assert.GreaterOrEqual(t, successRate, 95.0, "Success rate should be at least 95%")
	assert.Less(t, stats.p95, 500*time.Millisecond, "p95 latency should be < 500ms")
}

// TestLoad_SessionQueryPerformance тестирует производительность GetActiveSessions
func (s *LoadTestSuite) TestLoad_SessionQueryPerformance() {
	t := s.T()
	ctx, cancel := context.WithTimeout(s.ctx, 90*time.Second)
	defer cancel()

	// Создать 50 сессий
	sessionCount := 50
	sessionIDs := make([]string, 0, sessionCount)

	t.Logf("Creating %d sessions...", sessionCount)
	for i := 0; i < sessionCount; i++ {
		sessionID := fmt.Sprintf("query_session_%d_%d", i, time.Now().Unix())
		sessionIDs = append(sessionIDs, sessionID)

		connectReq := &pb.NotifyConnectRequest{
			Username:  fmt.Sprintf("query_user_%d", i),
			ClientIp:  fmt.Sprintf("192.168.210.%d", i%256),
			VpnIp:     fmt.Sprintf("10.30.10.%d", i%256),
			SessionId: sessionID,
			DeviceId:  fmt.Sprintf("device_%d", i),
		}

		_, err := s.grpcClient.NotifyConnect(ctx, connectReq)
		require.NoError(t, err, "Failed to create session %d", i)
	}

	t.Logf("Created %d sessions, testing query performance...", sessionCount)
	time.Sleep(500 * time.Millisecond)

	// Выполнить 100 запросов GetActiveSessions
	queryCount := 100
	latencies := make([]time.Duration, 0, queryCount)

	for i := 0; i < queryCount; i++ {
		queryStart := time.Now()
		resp, err := s.grpcClient.GetActiveSessions(ctx, &pb.GetActiveSessionsRequest{})
		queryLatency := time.Since(queryStart)

		require.NoError(t, err, "GetActiveSessions should succeed")
		assert.GreaterOrEqual(t, len(resp.Sessions), sessionCount,
			"Should have at least %d sessions", sessionCount)

		latencies = append(latencies, queryLatency)
		time.Sleep(10 * time.Millisecond)
	}

	stats := calculateLatencyStats(latencies)

	t.Logf("\n=== Session Query Performance Results ===")
	t.Logf("Sessions in store: %d", sessionCount)
	t.Logf("Query count: %d", queryCount)
	t.Logf("Min latency: %v", stats.min)
	t.Logf("Mean latency: %v", stats.mean)
	t.Logf("p50 latency: %v", stats.p50)
	t.Logf("p95 latency: %v", stats.p95)
	t.Logf("p99 latency: %v", stats.p99)
	t.Logf("Max latency: %v", stats.max)

	// Assertions
	assert.Less(t, stats.p95, 100*time.Millisecond,
		"p95 query latency should be < 100ms even with %d sessions", sessionCount)

	// Cleanup sessions
	t.Log("Cleaning up sessions...")
	for i, sessionID := range sessionIDs {
		_, _ = s.grpcClient.NotifyDisconnect(ctx, &pb.NotifyDisconnectRequest{
			Username:  fmt.Sprintf("query_user_%d", i),
			SessionId: sessionID,
		})
	}
}

// calculateLatencyStats вычисляет статистику по latency
func calculateLatencyStats(samples []time.Duration) LatencyStats {
	if len(samples) == 0 {
		return LatencyStats{}
	}

	// Sort samples
	sorted := make([]time.Duration, len(samples))
	copy(sorted, samples)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Calculate percentiles
	p50Index := int(float64(len(sorted)) * 0.50)
	p95Index := int(float64(len(sorted)) * 0.95)
	p99Index := int(float64(len(sorted)) * 0.99)

	// Calculate mean
	var sum time.Duration
	for _, s := range sorted {
		sum += s
	}
	mean := sum / time.Duration(len(sorted))

	return LatencyStats{
		samples:  sorted,
		min:      sorted[0],
		max:      sorted[len(sorted)-1],
		mean:     mean,
		p50:      sorted[p50Index],
		p95:      sorted[p95Index],
		p99:      sorted[p99Index],
		totalOps: int64(len(sorted)),
	}
}

// TestLoadE2E запускает load testing тесты
func TestLoadE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping load tests in short mode")
	}

	suite.Run(t, new(LoadTestSuite))
}
