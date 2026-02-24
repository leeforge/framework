# request — HTTP 请求工具

提供 HTTP 客户端封装、请求 ID 生成器与常用请求/响应工具函数。

## 主要功能

### 请求 ID 生成器

```go
import "github.com/leeforge/framework/request"

gen := request.NewRequestIDGenerator("req")

// 生成唯一 ID（前缀 + 随机 hex）
id := gen.Generate() // "req-a3f2b1c9d4e5..."
```

### HTTP 客户端

```go
// 创建客户端（带超时、重试）
client := request.NewClient(request.ClientConfig{
    BaseURL:    "https://api.example.com",
    Timeout:    30 * time.Second,
    MaxRetries: 3,
})

// GET 请求
resp, err := client.Get(ctx, "/users/123", nil)

// POST 请求（自动 JSON 序列化）
resp, err := client.Post(ctx, "/users", map[string]any{
    "name": "Alice",
    "email": "alice@example.com",
})

// 携带自定义 Header
resp, err := client.WithHeader("X-API-Key", apiKey).Get(ctx, "/data", nil)
```

### 请求上下文工具

```go
// 从请求中提取 TraceID（来自 X-Trace-ID 头）
traceID := request.GetTraceID(r)

// 从请求中提取用户 ID（由 AuthMiddleware 注入）
userID := request.GetUserID(r.Context())

// 获取客户端真实 IP（处理代理头）
ip := request.GetClientIP(r)
```

## 在 Handler 中使用

```go
func (h *Handler) ProxyRequest(w http.ResponseWriter, r *http.Request) {
    traceID := request.GetTraceID(r)

    resp, err := h.client.
        WithHeader("X-Trace-ID", traceID).
        Get(r.Context(), "/downstream", nil)
    if err != nil {
        // 处理错误
        return
    }
    // 转发响应...
}
```

## 注意事项

- 客户端默认开启连接池复用，建议复用同一个 `Client` 实例
- 超时设置适用于整个请求周期（连接 + 传输），需根据下游服务 SLA 合理配置
