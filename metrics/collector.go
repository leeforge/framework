package metrics

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Collector 指标收集器
type Collector struct {
	metrics map[string]*Metric
	mu      sync.RWMutex
}

// Metric 指标
type Metric struct {
	Type      string            `json:"type"`
	Value     float64           `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
	History   []float64         `json:"history,omitempty"`
	Timestamp int64             `json:"timestamp"`
}

// NewCollector 创建指标收集器
func NewCollector() *Collector {
	return &Collector{
		metrics: make(map[string]*Metric),
	}
}

// IncCounter 增加计数器
func (c *Collector) IncCounter(name string, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.buildKey(name, labels)
	if metric, exists := c.metrics[key]; exists {
		metric.Value++
		metric.Timestamp = time.Now().Unix()
	} else {
		c.metrics[key] = &Metric{
			Type:      "counter",
			Value:     1,
			Labels:    labels,
			Timestamp: time.Now().Unix(),
		}
	}
}

// AddCounter 增加计数器值
func (c *Collector) AddCounter(name string, value float64, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.buildKey(name, labels)
	if metric, exists := c.metrics[key]; exists {
		metric.Value += value
		metric.Timestamp = time.Now().Unix()
	} else {
		c.metrics[key] = &Metric{
			Type:      "counter",
			Value:     value,
			Labels:    labels,
			Timestamp: time.Now().Unix(),
		}
	}
}

// SetGauge 设置仪表值
func (c *Collector) SetGauge(name string, value float64, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.buildKey(name, labels)
	c.metrics[key] = &Metric{
		Type:      "gauge",
		Value:     value,
		Labels:    labels,
		Timestamp: time.Now().Unix(),
	}
}

// ObserveHistogram 观察直方图
func (c *Collector) ObserveHistogram(name string, value float64, labels map[string]string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := c.buildKey(name, labels)
	if metric, exists := c.metrics[key]; exists {
		metric.History = append(metric.History, value)
		if len(metric.History) > 100 {
			metric.History = metric.History[1:]
		}
		metric.Timestamp = time.Now().Unix()
	} else {
		c.metrics[key] = &Metric{
			Type:      "histogram",
			Value:     value,
			Labels:    labels,
			History:   []float64{value},
			Timestamp: time.Now().Unix(),
		}
	}
}

// RecordRequest 记录 HTTP 请求
func (c *Collector) RecordRequest(method, path string, status int, duration float64) {
	labels := map[string]string{
		"method": method,
		"path":   path,
		"status": strconv.Itoa(status),
	}

	c.IncCounter("http_requests_total", labels)
	c.ObserveHistogram("http_request_duration_seconds", duration, labels)
}

// RecordDBQuery 记录数据库查询
func (c *Collector) RecordDBQuery(query string, duration float64) {
	labels := map[string]string{
		"query": query,
	}

	c.IncCounter("db_queries_total", labels)
	c.ObserveHistogram("db_query_duration_seconds", duration, labels)
}

// RecordCacheHit 记录缓存命中
func (c *Collector) RecordCacheHit(cacheType string, hit bool) {
	labels := map[string]string{
		"type": cacheType,
		"hit":  strconv.FormatBool(hit),
	}

	c.IncCounter("cache_requests_total", labels)
	if hit {
		c.IncCounter("cache_hits_total", labels)
	} else {
		c.IncCounter("cache_misses_total", labels)
	}
}

// RecordError 记录错误
func (c *Collector) RecordError(method, path string, err error) {
	if err == nil {
		return
	}

	labels := map[string]string{
		"method": method,
		"path":   path,
	}

	c.IncCounter("http_errors_total", labels)
}

// buildKey 构建指标键
func (c *Collector) buildKey(name string, labels map[string]string) string {
	key := name
	if len(labels) > 0 {
		// 简化实现：按顺序添加标签
		for k, v := range labels {
			key += ":" + k + "=" + v
		}
	}
	return key
}

// GetMetrics 获取所有指标
func (c *Collector) GetMetrics() map[string]*Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 返回副本
	result := make(map[string]*Metric)
	for k, v := range c.metrics {
		result[k] = v
	}
	return result
}

// GetMetric 获取单个指标
func (c *Collector) GetMetric(name string, labels map[string]string) *Metric {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := c.buildKey(name, labels)
	return c.metrics[key]
}

// Reset 重置指标
func (c *Collector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.metrics = make(map[string]*Metric)
}

// MetricsMiddleware 指标中间件
type MetricsMiddleware struct {
	collector *Collector
}

// NewMetricsMiddleware 创建指标中间件
func NewMetricsMiddleware(collector *Collector) *MetricsMiddleware {
	return &MetricsMiddleware{
		collector: collector,
	}
}

// Middleware HTTP 指标中间件
func (m *MetricsMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 包装 ResponseWriter
		ww := &responseWriter{ResponseWriter: w, statusCode: 200}

		next.ServeHTTP(ww, r)

		duration := time.Since(start).Seconds()

		// 记录指标
		m.collector.RecordRequest(r.Method, r.URL.Path, ww.statusCode, duration)
	})
}

// responseWriter 包装器
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// MetricsHandler 指标处理器
type MetricsHandler struct {
	collector *Collector
}

// NewMetricsHandler 创建指标处理器
func NewMetricsHandler(collector *Collector) *MetricsHandler {
	return &MetricsHandler{
		collector: collector,
	}
}

// ServeHTTP 实现 http.Handler
func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	metrics := h.collector.GetMetrics()

	// 格式化输出
	data := make(map[string]interface{})
	for key, metric := range metrics {
		data[key] = map[string]interface{}{
			"type":      metric.Type,
			"value":     metric.Value,
			"labels":    metric.Labels,
			"history":   metric.History,
			"timestamp": metric.Timestamp,
		}
	}

	json.NewEncoder(w).Encode(data)
}

// MetricsConfig 指标配置
type MetricsConfig struct {
	EnableHTTPMetrics     bool
	EnableDBMetrics       bool
	EnableCacheMetrics    bool
	EnableBusinessMetrics bool
}

// MetricsManager 指标管理器
type MetricsManager struct {
	collector *Collector
	config    MetricsConfig
}

// NewMetricsManager 创建指标管理器
func NewMetricsManager(config MetricsConfig) *MetricsManager {
	return &MetricsManager{
		collector: NewCollector(),
		config:    config,
	}
}

// GetCollector 获取收集器
func (m *MetricsManager) GetCollector() *Collector {
	return m.collector
}

// GetHTTPMiddleware 获取 HTTP 中间件
func (m *MetricsManager) GetHTTPMiddleware() *MetricsMiddleware {
	return NewMetricsMiddleware(m.collector)
}

// GetMetricsHandler 获取指标处理器
func (m *MetricsManager) GetMetricsHandler() http.Handler {
	return NewMetricsHandler(m.collector)
}

// BusinessMetrics 业务指标
type BusinessMetrics struct {
	collector *Collector
}

// NewBusinessMetrics 创建业务指标
func NewBusinessMetrics(collector *Collector) *BusinessMetrics {
	return &BusinessMetrics{
		collector: collector,
	}
}

// RecordUserAction 记录用户行为
func (b *BusinessMetrics) RecordUserAction(userID, action string) {
	labels := map[string]string{
		"user_id": userID,
		"action":  action,
	}
	b.collector.IncCounter("user_actions_total", labels)
}

// RecordOrder 记录订单
func (b *BusinessMetrics) RecordOrder(userID string, amount float64, success bool) {
	labels := map[string]string{
		"user_id": userID,
		"success": strconv.FormatBool(success),
	}
	b.collector.IncCounter("orders_total", labels)
	b.collector.AddCounter("orders_amount_total", amount, labels)
}

// RecordAPICall 记录 API 调用
func (b *BusinessMetrics) RecordAPICall(service, method string, status int, duration float64) {
	labels := map[string]string{
		"service": service,
		"method":  method,
		"status":  strconv.Itoa(status),
	}
	b.collector.IncCounter("api_calls_total", labels)
	b.collector.ObserveHistogram("api_call_duration_seconds", duration, labels)
}

// GaugeManager 仪表管理器
type GaugeManager struct {
	collector *Collector
}

// NewGaugeManager 创建仪表管理器
func NewGaugeManager(collector *Collector) *GaugeManager {
	return &GaugeManager{
		collector: collector,
	}
}

// SetSystemMetrics 设置系统指标
func (g *GaugeManager) SetSystemMetrics(cpu, memory, goroutines int) {
	g.collector.SetGauge("system_cpu_usage", float64(cpu), map[string]string{})
	g.collector.SetGauge("system_memory_usage", float64(memory), map[string]string{})
	g.collector.SetGauge("system_goroutines", float64(goroutines), map[string]string{})
}

// SetQueueMetrics 设置队列指标
func (g *GaugeManager) SetQueueMetrics(queueName string, size, processing int) {
	labels := map[string]string{"queue": queueName}
	g.collector.SetGauge("queue_size", float64(size), labels)
	g.collector.SetGauge("queue_processing", float64(processing), labels)
}

// SetConnectionMetrics 设置连接指标
func (g *GaugeManager) SetConnectionMetrics(db, redis int) {
	g.collector.SetGauge("connections_db", float64(db), map[string]string{})
	g.collector.SetGauge("connections_redis", float64(redis), map[string]string{})
}

// MetricsDashboard 指标仪表板
type MetricsDashboard struct {
	collector *Collector
}

// NewMetricsDashboard 创建指标仪表板
func NewMetricsDashboard(collector *Collector) *MetricsDashboard {
	return &MetricsDashboard{
		collector: collector,
	}
}

// GetSummary 获取指标摘要
func (d *MetricsDashboard) GetSummary() map[string]interface{} {
	summary := make(map[string]interface{})
	metrics := d.collector.GetMetrics()

	// 计算总数
	var httpRequests, dbQueries, cacheHits, cacheMisses int64
	var avgHTTPDuration, avgDBDuration float64
	var httpCount, dbCount int

	for key, metric := range metrics {
		switch {
		case metric.Type == "counter":
			switch {
			case keyContains(key, "http_requests_total"):
				httpRequests += int64(metric.Value)
			case keyContains(key, "db_queries_total"):
				dbQueries += int64(metric.Value)
			case keyContains(key, "cache_hits_total"):
				cacheHits += int64(metric.Value)
			case keyContains(key, "cache_misses_total"):
				cacheMisses += int64(metric.Value)
			}
		case metric.Type == "histogram":
			if len(metric.History) > 0 {
				var sum float64
				for _, v := range metric.History {
					sum += v
				}
				avg := sum / float64(len(metric.History))
				switch {
				case keyContains(key, "http_request_duration"):
					avgHTTPDuration += avg
					httpCount++
				case keyContains(key, "db_query_duration"):
					avgDBDuration += avg
					dbCount++
				}
			}
		}
	}

	summary["http_requests_total"] = httpRequests
	summary["db_queries_total"] = dbQueries
	summary["cache_hit_rate"] = 0.0
	if cacheHits+cacheMisses > 0 {
		summary["cache_hit_rate"] = float64(cacheHits) / float64(cacheHits+cacheMisses) * 100
	}
	if httpCount > 0 {
		summary["avg_http_duration"] = avgHTTPDuration / float64(httpCount)
	}
	if dbCount > 0 {
		summary["avg_db_duration"] = avgDBDuration / float64(dbCount)
	}

	return summary
}

// keyContains 检查 key 是否包含指定字符串
func keyContains(key, substr string) bool {
	return len(key) >= len(substr) && (key == substr || len(key) > len(substr) && (key[:len(substr)] == substr || key[len(key)-len(substr):] == substr || containsSubstring(key, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// GetSlowQueries 获取慢查询
func (d *MetricsDashboard) GetSlowQueries(threshold float64) []string {
	metrics := d.collector.GetMetrics()
	var slow []string

	for key, metric := range metrics {
		if metric.Type == "histogram" && len(metric.History) > 0 {
			var sum float64
			for _, v := range metric.History {
				sum += v
			}
			avg := sum / float64(len(metric.History))
			if avg > threshold {
				slow = append(slow, key)
			}
		}
	}

	return slow
}

// MetricsExporterConfig 指标导出器配置
type MetricsExporterConfig struct {
	Endpoint   string
	EnableAuth bool
	Username   string
	Password   string
}

// MetricsHTTPExporter HTTP 指标导出器
type MetricsHTTPExporter struct {
	collector *Collector
	config    MetricsExporterConfig
}

// NewMetricsHTTPExporter 创建 HTTP 指标导出器
func NewMetricsHTTPExporter(collector *Collector, config MetricsExporterConfig) *MetricsHTTPExporter {
	return &MetricsHTTPExporter{
		collector: collector,
		config:    config,
	}
}

// ServeHTTP 实现 http.Handler
func (e *MetricsHTTPExporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 认证检查
	if e.config.EnableAuth {
		username, password, ok := r.BasicAuth()
		if !ok || username != e.config.Username || password != e.config.Password {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}
	}

	// 导出指标
	handler := NewMetricsHandler(e.collector)
	handler.ServeHTTP(w, r)
}

// RegisterMetricsRoutes 注册指标路由
func RegisterMetricsRoutes(mux *http.ServeMux, exporter *MetricsHTTPExporter) {
	mux.Handle("/metrics", exporter)
}

// DefaultMetricsConfig 默认配置
func DefaultMetricsConfig() MetricsConfig {
	return MetricsConfig{
		EnableHTTPMetrics:     true,
		EnableDBMetrics:       true,
		EnableCacheMetrics:    true,
		EnableBusinessMetrics: true,
	}
}

// MetricsRecorder 指标记录器
type MetricsRecorder struct {
	collector *Collector
}

func NewMetricsRecorder(collector *Collector) *MetricsRecorder {
	return &MetricsRecorder{
		collector: collector,
	}
}

func (r *MetricsRecorder) RecordDuration(name string, labels map[string]string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start).Seconds()
	r.collector.ObserveHistogram(name, duration, labels)
	return err
}

func (r *MetricsRecorder) RecordSuccess(name string, labels map[string]string) {
	r.collector.IncCounter(name, labels)
}

func (r *MetricsRecorder) RecordFailure(name string, labels map[string]string) {
	r.collector.IncCounter(name, labels)
}

type MetricsExporter interface {
	Export(metrics map[string]*Metric) error
}

type PrometheusExporter struct {
	collector *Collector
}

func NewPrometheusExporter(collector *Collector) *PrometheusExporter {
	return &PrometheusExporter{
		collector: collector,
	}
}

func (e *PrometheusExporter) Export(metrics map[string]*Metric) error {
	return nil
}

func (e *PrometheusExporter) GetPrometheusFormat() string {
	metrics := e.collector.GetMetrics()
	var sb strings.Builder

	for key, metric := range metrics {
		labels := ""
		if metric.Labels != nil && len(metric.Labels) > 0 {
			var labelPairs []string
			for k, v := range metric.Labels {
				labelPairs = append(labelPairs, k+"=\""+v+"\"")
			}
			labels = "{" + strings.Join(labelPairs, ",") + "}"
		}

		switch metric.Type {
		case "counter":
			sb.WriteString(fmt.Sprintf("%s%s %.2f\n", key, labels, metric.Value))
		case "gauge":
			sb.WriteString(fmt.Sprintf("%s%s %.2f\n", key, labels, metric.Value))
		case "histogram":
			if len(metric.History) > 0 {
				var sum float64
				for _, v := range metric.History {
					sum += v
				}
				avg := sum / float64(len(metric.History))
				sb.WriteString(fmt.Sprintf("%s_avg%s %.2f\n", key, labels, avg))
				sb.WriteString(fmt.Sprintf("%s_count%s %d\n", key, labels, len(metric.History)))
			}
		}
	}

	return sb.String()
}

type MetricsHealthCheck struct {
	collector *Collector
	threshold MetricsHealthThreshold
}

type MetricsHealthThreshold struct {
	MaxErrorRate     float64
	MaxAvgDuration   float64
	MaxCacheMissRate float64
}

func NewMetricsHealthCheck(collector *Collector, threshold MetricsHealthThreshold) *MetricsHealthCheck {
	return &MetricsHealthCheck{
		collector: collector,
		threshold: threshold,
	}
}

func (h *MetricsHealthCheck) Check() HealthResult {
	metrics := h.collector.GetMetrics()
	result := HealthResult{
		Healthy: true,
		Details: make(map[string]interface{}),
	}

	var totalRequests, totalErrors int64
	for key, metric := range metrics {
		if metric.Type == "counter" {
			if keyContains(key, "http_requests_total") {
				totalRequests += int64(metric.Value)
			}
			if keyContains(key, "http_errors_total") {
				totalErrors += int64(metric.Value)
			}
		}
	}

	if totalRequests > 0 {
		errorRate := float64(totalErrors) / float64(totalRequests)
		result.Details["error_rate"] = errorRate
		if errorRate > h.threshold.MaxErrorRate {
			result.Healthy = false
			result.Issues = append(result.Issues, fmt.Sprintf("Error rate too high: %.2f%%", errorRate*100))
		}
	}

	var totalDuration, durationCount float64
	for key, metric := range metrics {
		if metric.Type == "histogram" && keyContains(key, "http_request_duration") && len(metric.History) > 0 {
			for _, v := range metric.History {
				totalDuration += v
			}
			durationCount += float64(len(metric.History))
		}
	}

	if durationCount > 0 {
		avgDuration := totalDuration / durationCount
		result.Details["avg_duration"] = avgDuration
		if avgDuration > h.threshold.MaxAvgDuration {
			result.Healthy = false
			result.Issues = append(result.Issues, fmt.Sprintf("Average duration too high: %.3fs", avgDuration))
		}
	}

	var cacheHits, cacheMisses int64
	for key, metric := range metrics {
		if metric.Type == "counter" {
			if keyContains(key, "cache_hits_total") {
				cacheHits += int64(metric.Value)
			}
			if keyContains(key, "cache_misses_total") {
				cacheMisses += int64(metric.Value)
			}
		}
	}

	if cacheHits+cacheMisses > 0 {
		missRate := float64(cacheMisses) / float64(cacheHits+cacheMisses)
		result.Details["cache_miss_rate"] = missRate
		if missRate > h.threshold.MaxCacheMissRate {
			result.Healthy = false
			result.Issues = append(result.Issues, fmt.Sprintf("Cache miss rate too high: %.2f%%", missRate*100))
		}
	}

	return result
}

type HealthResult struct {
	Healthy bool
	Details map[string]interface{}
	Issues  []string
}

type MetricsCollectorFactory struct {
	collectors map[string]*Collector
	mu         sync.RWMutex
}

func NewMetricsCollectorFactory() *MetricsCollectorFactory {
	return &MetricsCollectorFactory{
		collectors: make(map[string]*Collector),
	}
}

func (f *MetricsCollectorFactory) GetCollector(name string) *Collector {
	f.mu.RLock()
	collector, exists := f.collectors[name]
	f.mu.RUnlock()

	if exists {
		return collector
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if collector, exists := f.collectors[name]; exists {
		return collector
	}

	collector = NewCollector()
	f.collectors[name] = collector
	return collector
}

func (f *MetricsCollectorFactory) GetAllCollectors() map[string]*Collector {
	f.mu.RLock()
	defer f.mu.RUnlock()

	result := make(map[string]*Collector)
	for k, v := range f.collectors {
		result[k] = v
	}
	return result
}

func (f *MetricsCollectorFactory) ResetCollector(name string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.collectors, name)
}

func (f *MetricsCollectorFactory) ResetAll() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.collectors = make(map[string]*Collector)
}

type MetricsAggregator struct {
	collectors []*Collector
}

func NewMetricsAggregator(collectors ...*Collector) *MetricsAggregator {
	return &MetricsAggregator{
		collectors: collectors,
	}
}

func (a *MetricsAggregator) Aggregate() map[string]*Metric {
	aggregated := make(map[string]*Metric)

	for _, collector := range a.collectors {
		metrics := collector.GetMetrics()
		for key, metric := range metrics {
			if existing, exists := aggregated[key]; exists {
				existing.Value += metric.Value
				if metric.History != nil {
					existing.History = append(existing.History, metric.History...)
				}
			} else {
				aggregated[key] = &Metric{
					Type:      metric.Type,
					Value:     metric.Value,
					Labels:    metric.Labels,
					History:   metric.History,
					Timestamp: metric.Timestamp,
				}
			}
		}
	}

	return aggregated
}

type MetricsRateLimiter struct {
	collector *Collector
	limit     int64
	window    time.Duration
	requests  map[string][]int64
	mu        sync.RWMutex
}

func NewMetricsRateLimiter(collector *Collector, limit int64, window time.Duration) *MetricsRateLimiter {
	return &MetricsRateLimiter{
		collector: collector,
		limit:     limit,
		window:    window,
		requests:  make(map[string][]int64),
	}
}

func (l *MetricsRateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().Unix()
	cutoff := now - int64(l.window.Seconds())

	if timestamps, exists := l.requests[key]; exists {
		var valid []int64
		for _, ts := range timestamps {
			if ts >= cutoff {
				valid = append(valid, ts)
			}
		}
		l.requests[key] = valid

		if len(valid) >= int(l.limit) {
			l.collector.IncCounter("rate_limit_exceeded", map[string]string{"key": key})
			return false
		}
	}

	l.requests[key] = append(l.requests[key], now)
	l.collector.IncCounter("rate_limit_allowed", map[string]string{"key": key})
	return true
}

func (l *MetricsRateLimiter) GetUsage(key string) int {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if timestamps, exists := l.requests[key]; exists {
		now := time.Now().Unix()
		cutoff := now - int64(l.window.Seconds())
		count := 0
		for _, ts := range timestamps {
			if ts >= cutoff {
				count++
			}
		}
		return count
	}
	return 0
}

type MetricsSampler struct {
	collector *Collector
	interval  time.Duration
	stop      chan struct{}
}

func NewMetricsSampler(collector *Collector, interval time.Duration) *MetricsSampler {
	return &MetricsSampler{
		collector: collector,
		interval:  interval,
		stop:      make(chan struct{}),
	}
}

func (s *MetricsSampler) Start(sampler func() map[string]float64) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			data := sampler()
			for name, value := range data {
				s.collector.SetGauge(name, value, nil)
			}
		case <-s.stop:
			return
		}
	}
}

func (s *MetricsSampler) Stop() {
	close(s.stop)
}

type MetricsEventForwarder struct {
	collector *Collector
	forwarder func(event string, data map[string]interface{}) error
}

func NewMetricsEventForwarder(collector *Collector, forwarder func(string, map[string]interface{}) error) *MetricsEventForwarder {
	return &MetricsEventForwarder{
		collector: collector,
		forwarder: forwarder,
	}
}

func (f *MetricsEventForwarder) Forward() error {
	metrics := f.collector.GetMetrics()
	data := make(map[string]interface{})
	for key, metric := range metrics {
		data[key] = map[string]interface{}{
			"type":      metric.Type,
			"value":     metric.Value,
			"labels":    metric.Labels,
			"timestamp": metric.Timestamp,
		}
	}
	return f.forwarder("metrics_snapshot", data)
}

func RegisterMetricsEventForwarder(collector *Collector, forwarder func(string, map[string]interface{}) error) *MetricsEventForwarder {
	return NewMetricsEventForwarder(collector, forwarder)
}

type MetricsSnapshot struct {
	Timestamp time.Time              `json:"timestamp"`
	Metrics   map[string]*Metric     `json:"metrics"`
	Summary   map[string]interface{} `json:"summary"`
}

func TakeSnapshot(collector *Collector) MetricsSnapshot {
	return MetricsSnapshot{
		Timestamp: time.Now(),
		Metrics:   collector.GetMetrics(),
		Summary:   NewMetricsDashboard(collector).GetSummary(),
	}
}

type MetricsExporterRegistry struct {
	exporters map[string]MetricsExporter
	mu        sync.RWMutex
}

func NewMetricsExporterRegistry() *MetricsExporterRegistry {
	return &MetricsExporterRegistry{
		exporters: make(map[string]MetricsExporter),
	}
}

func (r *MetricsExporterRegistry) Register(name string, exporter MetricsExporter) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.exporters[name] = exporter
}

func (r *MetricsExporterRegistry) ExportAll(collector *Collector) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics := collector.GetMetrics()
	for name, exporter := range r.exporters {
		if err := exporter.Export(metrics); err != nil {
			return fmt.Errorf("exporter %s failed: %w", name, err)
		}
	}
	return nil
}

func (r *MetricsExporterRegistry) GetExporter(name string) MetricsExporter {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.exporters[name]
}

type MetricsCollector struct {
	*Collector
	*MetricsManager
	*BusinessMetrics
	*GaugeManager
	*MetricsRecorder
}

func NewMetricsCollector(config MetricsConfig) *MetricsCollector {
	collector := NewCollector()
	return &MetricsCollector{
		Collector:       collector,
		MetricsManager:  NewMetricsManager(config),
		BusinessMetrics: NewBusinessMetrics(collector),
		GaugeManager:    NewGaugeManager(collector),
		MetricsRecorder: NewMetricsRecorder(collector),
	}
}

func DefaultMetricsCollector() *MetricsCollector {
	return NewMetricsCollector(DefaultMetricsConfig())
}
