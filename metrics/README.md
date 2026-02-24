# metrics — 指标收集

提供内存型指标收集器，支持 Counter（计数器）、Gauge（仪表盘）、Histogram（直方图）三种指标类型，并内置 Prometheus 格式导出与 HTTP 中间件。

## 指标类型

| 类型 | 说明 | 使用场景 |
|---|---|---|
| Counter | 单调递增计数 | 请求总数、错误次数 |
| Gauge | 任意值，可升可降 | 当前连接数、内存用量 |
| Histogram | 观测值分布，保留历史 | 请求耗时、响应大小 |

## 快速开始

```go
import "github.com/leeforge/framework/metrics"

collector := metrics.NewCollector()

// Counter：请求计数
collector.IncCounter("http_requests_total", map[string]string{
    "method": "GET",
    "path":   "/users",
    "status": "200",
})

// Gauge：当前连接数
collector.SetGauge("active_connections", 42, map[string]string{
    "service": "api",
})

// Histogram：记录请求耗时（秒）
collector.ObserveHistogram("request_duration_seconds", 0.023, map[string]string{
    "handler": "listUsers",
})
```

## HTTP 指标中间件

```go
manager := metrics.NewMetricsManager(metrics.DefaultMetricsConfig())

// 在 Chi Router 上注册中间件
r.Use(manager.GetHTTPMiddleware().Middleware)

// 暴露 /metrics 端点
r.Handle("/metrics", manager.GetMetricsHandler())
```

## 内置快捷方法

```go
// 记录 HTTP 请求（自动处理计数+耗时）
collector.RecordRequest("GET", "/users", 200, 0.015)

// 记录数据库查询
collector.RecordDBQuery("SELECT * FROM users", 0.003)

// 记录缓存命中
collector.RecordCacheHit("redis", true)  // hit
collector.RecordCacheHit("redis", false) // miss

// 记录错误
collector.RecordError("POST", "/users", err)
```

## 业务指标

```go
biz := metrics.NewBusinessMetrics(collector)

biz.RecordUserAction("user-123", "login")
biz.RecordOrder("user-123", 99.99, true)
biz.RecordAPICall("payment-service", "Charge", 200, 0.05)
```

## 系统仪表

```go
gauge := metrics.NewGaugeManager(collector)

gauge.SetSystemMetrics(cpu, memory, goroutines)
gauge.SetQueueMetrics("email-queue", 100, 10)
gauge.SetConnectionMetrics(dbConns, redisConns)
```

## 指标摘要与健康检查

```go
dashboard := metrics.NewMetricsDashboard(collector)
summary := dashboard.GetSummary()
// 返回：http_requests_total, db_queries_total, cache_hit_rate 等汇总数据

// 基于阈值的健康检查
healthCheck := metrics.NewMetricsHealthCheck(collector, metrics.MetricsHealthThreshold{
    MaxErrorRate:     0.05,   // 错误率不超过 5%
    MaxAvgDuration:   0.5,    // 平均耗时不超过 500ms
    MaxCacheMissRate: 0.3,    // 缓存未命中率不超过 30%
})

result := healthCheck.Check()
if !result.Healthy {
    log.Warn("metrics health check failed", result.Issues)
}
```

## Prometheus 格式导出

```go
exporter := metrics.NewPrometheusExporter(collector)
prometheusText := exporter.GetPrometheusFormat()
// 输出标准 Prometheus 文本格式
```

## 注意事项

- 当前 Histogram 最多保留 100 个历史观测值，旧值会被丢弃
- 指标 Key 使用 `name:label=value` 格式，标签顺序可能影响 Key 的一致性（建议使用固定顺序）
- 生产环境建议配合 Prometheus + Grafana 使用
