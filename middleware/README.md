# middleware — 网关中间件

提供 HTTP API 网关层能力：限流（Rate Limiter）、安全防护（CORS、Helmet、IP 黑白名单）与中间件链组合。

## 组件概览

| 组件 | 说明 |
|---|---|
| `RateLimiter` | 基于 API Key 的分钟/日/突发三维限流 |
| `SlidingWindowLimiter` | 滑动窗口限流器 |
| `TokenBucketLimiter` | 令牌桶限流器 |
| `SecurityMiddleware` | CORS + Helmet + IP 黑白名单 |
| `GatewayMiddleware` | 集成限流、鉴权、日志、指标、追踪的统一网关链 |
| `MiddlewareChain` | 通用中间件链 Builder |

## 快速开始

### 限流器

```go
import "github.com/leeforge/framework/middleware"

backend := middleware.NewRedisBackend()
rateLimiter := middleware.NewRateLimiter(backend, middleware.RateLimitConfig{
    DefaultRate:  100, // 每分钟 100 次
    DefaultDaily: 1000, // 每天 1000 次
    Burst:        10,  // 突发 10 次/秒
    Strategies: map[string]middleware.Strategy{
        "/api/upload": {Rate: 10, Daily: 100, Burst: 2}, // 上传接口单独限流
    },
})

r.Use(rateLimiter.Middleware)
```

### 安全中间件

```go
security := middleware.NewSecurityMiddleware(middleware.SecurityConfig{
    CORS: middleware.CORSConfig{
        Enabled:        true,
        AllowedOrigins: []string{"https://myapp.com"},
        AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
        AllowedHeaders: []string{"Authorization", "Content-Type"},
        AllowCredentials: true,
        MaxAge:         86400,
    },
    Helmet:      true, // 注入安全响应头
    IPBlacklist: []string{"1.2.3.4"},
    RequestSize: 10 * 1024 * 1024, // 10MB 请求体限制
})

r.Use(security.Middleware)
```

### 网关 Builder

```go
gw, err := middleware.NewGatewayBuilder(logger).
    WithRateLimit(middleware.RateLimitConfig{DefaultRate: 200}).
    WithAuth(true).
    WithMetrics(true).
    WithTracing(true).
    Build()

handler, _ := gw.Handler(r)
```

### 中间件链

```go
chain := middleware.NewMiddlewareChain().
    Use(loggingMiddleware).
    Use(authMiddleware).
    Use(metricsMiddleware)

http.ListenAndServe(":8080", chain.Then(handler))
```

## 限流管理 API

```go
// 注册管理路由（需要鉴权保护）
mux.HandleFunc("/admin/rate-limit/usage", rateLimitHandler.GetUsage)
mux.HandleFunc("/admin/rate-limit/reset", rateLimitHandler.ResetUsage)
```

| 接口 | 说明 |
|---|---|
| `GET /admin/rate-limit/usage?api_key=xxx` | 查看 API Key 的使用量 |
| `GET /admin/rate-limit/reset?api_key=xxx` | 重置 API Key 的限流计数 |

## 网关中间件执行顺序

```
请求 → Metrics → Tracing → Auth → RateLimit → Logging → Handler
```

## 注意事项

- `BackendAdapter` 的 `RedisBackend` 当前为内存模拟实现，生产环境需替换为真实 Redis 实现
- IP 黑白名单的 IP 匹配使用字符串精确匹配（不支持 CIDR 段），需要 CIDR 时需自行扩展
- Helmet 安全头包含：`X-Content-Type-Options`、`X-Frame-Options`、`Strict-Transport-Security` 等
