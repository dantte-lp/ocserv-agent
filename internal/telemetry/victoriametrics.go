package telemetry

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/dantte-lp/ocserv-agent/internal/config"
)

// VictoriaMetricsExporter экспортирует метрики напрямую в VictoriaMetrics.
// Использует Prometheus text exposition format.
type VictoriaMetricsExporter struct {
	config      config.VictoriaMetricsConfig
	client      *http.Client
	meterReader *manualReader
	mu          sync.Mutex
	done        chan struct{}
	wg          sync.WaitGroup
}

// manualReader реализует metric.Reader для сбора метрик.
type manualReader struct {
	mu      sync.Mutex
	metrics []metricData
}

// metricData хранит данные метрики.
type metricData struct {
	Name   string
	Type   string
	Value  float64
	Labels map[string]string
}

// NewVictoriaMetricsExporter создает новый exporter для VictoriaMetrics.
func NewVictoriaMetricsExporter(cfg config.VictoriaMetricsConfig) *VictoriaMetricsExporter {
	e := &VictoriaMetricsExporter{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		meterReader: &manualReader{
			metrics: make([]metricData, 0),
		},
		done: make(chan struct{}),
	}

	return e
}

// Start запускает фоновый процесс отправки метрик.
func (e *VictoriaMetricsExporter) Start(ctx context.Context) {
	if !e.config.Enabled {
		return
	}

	e.wg.Add(1)
	go e.pushLoop(ctx)
}

// pushLoop периодически отправляет метрики в VictoriaMetrics.
func (e *VictoriaMetricsExporter) pushLoop(ctx context.Context) {
	defer e.wg.Done()

	ticker := time.NewTicker(e.config.PushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := e.Push(ctx); err != nil {
				// TODO: Логирование ошибок через slog
				fmt.Printf("Failed to push metrics to VictoriaMetrics: %v\n", err)
			}
		case <-e.done:
			// Финальный push перед закрытием
			_ = e.Push(ctx)
			return
		case <-ctx.Done():
			return
		}
	}
}

// Push отправляет метрики в VictoriaMetrics.
func (e *VictoriaMetricsExporter) Push(ctx context.Context) error {
	if !e.config.Enabled {
		return nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Собираем метрики (это заглушка, реальный сбор будет через OTEL SDK)
	// В production это должно работать с metric.MeterProvider
	metricsText := e.formatPrometheusMetrics()

	if metricsText == "" {
		// Нет метрик для отправки
		return nil
	}

	// Создаем HTTP запрос
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.config.Endpoint, strings.NewReader(metricsText))
	if err != nil {
		return errors.Wrap(err, "failed to create request")
	}

	req.Header.Set("Content-Type", "text/plain")

	// Добавляем Basic Auth если настроено
	if e.config.Username != "" && e.config.Password != "" {
		req.SetBasicAuth(e.config.Username, e.config.Password)
	}

	// Отправляем запрос
	resp, err := e.client.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed to send metrics to VictoriaMetrics")
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			slog.Error("failed to close response body", "error", err)
		}
	}()

	// Проверяем статус ответа
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		// errors.Newf создаёт новую ошибку, а не оборачивает существующую - это корректно
		//nolint:wrapcheck // creating new error, not wrapping external one
		return errors.Newf("VictoriaMetrics returned error status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// formatPrometheusMetrics форматирует метрики в Prometheus text exposition format.
func (e *VictoriaMetricsExporter) formatPrometheusMetrics() string {
	var buf bytes.Buffer

	// Пример формата Prometheus:
	// # HELP metric_name Description
	// # TYPE metric_name counter
	// metric_name{label1="value1",label2="value2"} 42.0 1234567890000

	// Базовые метки из конфигурации
	globalLabels := e.formatLabels(e.config.Labels)

	// Добавляем timestamp
	timestamp := time.Now().UnixMilli()

	// Пример метрики (в реальном коде это будет собираться из OTEL metrics)
	buf.WriteString("# HELP ocserv_agent_info Agent information\n")
	buf.WriteString("# TYPE ocserv_agent_info gauge\n")
	buf.WriteString(fmt.Sprintf("ocserv_agent_info%s 1 %d\n", globalLabels, timestamp))

	return buf.String()
}

// formatLabels форматирует метки в формат Prometheus {key="value",key2="value2"}.
func (e *VictoriaMetricsExporter) formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}

	var parts []string
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, k, escapePrometheusValue(v)))
	}

	return "{" + strings.Join(parts, ",") + "}"
}

// escapePrometheusValue экранирует специальные символы в значениях меток.
func escapePrometheusValue(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

// Close корректно завершает работу exporter.
func (e *VictoriaMetricsExporter) Close() error {
	close(e.done)
	e.wg.Wait()
	return nil
}

// RegisterMetric регистрирует метрику для отправки (заглушка для будущей интеграции).
func (e *VictoriaMetricsExporter) RegisterMetric(name string, metricType string, value float64, labels map[string]string) {
	e.meterReader.mu.Lock()
	defer e.meterReader.mu.Unlock()

	e.meterReader.metrics = append(e.meterReader.metrics, metricData{
		Name:   name,
		Type:   metricType,
		Value:  value,
		Labels: labels,
	})
}

// Forceflush выполняет принудительную отправку метрик (для интерфейса метрического reader).
func (mr *manualReader) Forceflush(ctx context.Context) error {
	// Заглушка - в полной реализации здесь будет сбор из instruments
	return nil
}

// Shutdown завершает работу reader.
func (mr *manualReader) Shutdown(ctx context.Context) error {
	return nil
}
