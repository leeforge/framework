# tracing — 分布式链路追踪

提供 OpenTelemetry 风格的分布式链路追踪能力，支持 Span 生命周期管理、采样策略、批处理导出与 HTTP 中间件。

## 核心概念

| 概念 | 说明 |
|---|---|
| `Trace` | 一次完整请求的全链路追踪标识（TraceID） |
| `Span` | 一次具体操作（如 DB 查询、HTTP 调用）的记录 |
| `Sampler` | 采样器，控制哪些 Trace 需要被记录 |
| `SpanProcessor` | Span 结束后的处理器（导出、聚合等） |
| `SpanExporter` | 导出实现（控制台、Jaeger、Zipkin 等） |

## 快速开始

```go
import "github.com/leeforge/framework/tracing"

// 创建 Tracer
tracer, err := tracing.NewTracer(tracing.TracerConfig{
    ServiceName:    "user-service",
    ServiceVersion: "1.0.0",
    SamplingRate:   1.0, // 100% 采样
})

// 开始一个 Span
ctx, span := tracer.Start(ctx, "user.query")
defer tracer.End(span, err) // 传入 err 自动标记状态

// 添加属性
tracer.SetAttributes(span, map[string]interface{}{
    "user.id": userID,
    "db.table": "users",
})

// 添加事件
tracer.AddEvent(span, "cache.hit", map[string]interface{}{"key": cacheKey})
```

## HTTP 追踪中间件

```go
middleware := tracing.NewTracerMiddleware(tracer)

r := chi.NewRouter()
r.Use(middleware.Middleware)
```

中间件自动记录：`http.method`、`http.url`、`http.host`、`http.status_code`。

## 高级用法

### 分布式追踪（DistributedTracer）

```go
dt, err := tracing.NewDistributedTracer(tracing.DistributedTracingConfig{
    Enable:             true,
    ServiceName:        "cms-backend",
    SamplingRate:       0.1, // 10% 采样（生产推荐）
    EnableBatching:     true,
    BatchTimeout:       5 * time.Second,
    EnableDBTracing:    true,
    EnableCacheTracing: true,
    EnableHTTPTracing:  true,
})

// 追踪 DB 查询
err = dt.TraceDBQuery(ctx, "SELECT * FROM users WHERE id = $1", func(ctx context.Context) error {
    return db.QueryRow(ctx, id, &user)
})

// 追踪缓存操作
err = dt.TraceCacheOperation(ctx, "get", "user:123", func(ctx context.Context) error {
    return cache.Get(ctx, "user:123", &user)
})

// 追踪下游 HTTP 调用
err = dt.TraceHTTPCall(ctx, "GET", "https://api.service.com/data", func(ctx context.Context) error {
    return httpClient.Get(ctx, url)
})
```

### 批量处理（减少导出开销）

```go
exporter := tracing.NewConsoleExporter()
processor := tracing.NewBatchSpanProcessor(exporter, 100, 5*time.Second)

tracer, _ := tracing.NewTracer(tracing.TracerConfig{
    ServiceName: "my-service",
    Processor:   processor,
})
```

### 从 Context 获取追踪信息

```go
traceID := tracing.GetTraceID(ctx)
spanID := tracing.GetSpanID(ctx)
sampled := tracing.IsSampled(ctx)
```

## 采样策略

```go
// 总是采样（开发/测试）
sampler := &tracing.AlwaysSampler{}

// 从不采样（禁用追踪）
sampler := &tracing.NeverSampler{}

// 按比例采样（生产推荐，如 10%）
sampler := tracing.NewTraceIDRatioBased(0.1)
```

## SpanKind

| 类型 | 说明 |
|---|---|
| `SpanKindServer` | 接收请求的服务端 |
| `SpanKindClient` | 发出请求的客户端 |
| `SpanKindProducer` | 消息发布方 |
| `SpanKindConsumer` | 消息消费方 |
| `SpanKindInternal` | 内部操作（默认） |

## 注意事项

- 当前 `ConsoleExporter` 仅输出到 stdout，生产环境需实现 `SpanExporter` 接口对接 Jaeger/Zipkin
- Span 的 `End` 方法必须调用，建议使用 `defer tracer.End(span, err)` 模式
- 高流量场景务必配置合理的采样率（如 1%~10%），避免追踪数据淹没存储
