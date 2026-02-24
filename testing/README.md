# testing — 测试辅助工具

提供 HTTP 测试上下文、性能基准测试套件等测试辅助工具，简化 Go 单元测试与集成测试编写。

## 主要组件

### TestContext — HTTP 集成测试上下文

```go
import frameTesting "github.com/leeforge/framework/testing"

func TestCreateUser(t *testing.T) {
    tc := frameTesting.NewTestContext(t)
    defer tc.Cleanup()

    // 创建测试 HTTP 服务
    handler := setupTestHandler()
    server := tc.NewServer(handler)

    // 发起 POST 请求
    resp := tc.POST(server, "/users", map[string]any{
        "name": "Alice",
        "email": "alice@test.com",
    })

    tc.AssertStatus(resp, http.StatusCreated)
    tc.AssertJSONField(resp, "data.name", "Alice")
}
```

### HTTP 测试辅助

```go
// 构建测试请求
req := tc.NewRequest("GET", "/users/123", nil)
req.Header.Set("Authorization", "Bearer "+token)

// 记录响应
rec := httptest.NewRecorder()
handler.ServeHTTP(rec, req)

// 断言
tc.AssertStatus(rec, http.StatusOK)
tc.AssertJSON(rec, `{"data": {"id": "123"}}`)
```

### 性能测试套件

```go
suite := frameTesting.NewPerformanceTestSuite("UserAPI")

suite.Add("CreateUser", func(pc *frameTesting.PerformanceContext) *frameTesting.BenchmarkResult {
    start := time.Now()
    err := createUser(pc.Ctx())
    return pc.Result(time.Since(start), err)
})

results := suite.Run(frameTesting.PerformanceConfig{
    Iterations:  1000,
    Concurrency: 10,
    WarmupRuns:  50,
})

suite.Report(results)
// 输出：平均耗时、P99、最大并发、成功率等
```

### 组件注册（测试 Mock）

```go
tc := frameTesting.NewTestContext(t)

// 注册测试用 Mock 组件
tc.Register("db", mockDB)
tc.Register("cache", mockCache)

// 获取组件
db := tc.Get("db").(MockDB)
```

## 常用断言

```go
tc.AssertStatus(resp, http.StatusOK)
tc.AssertStatus(resp, http.StatusNotFound)

// 断言 JSON 响应字段
tc.AssertJSONField(resp, "data.email", "alice@test.com")
tc.AssertJSONField(resp, "error", nil)
```

## 注意事项

- `TestContext` 内置 30 秒超时，单个测试超时会自动取消
- `NewServer` 创建的 `httptest.Server` 会在 `Cleanup` 时自动关闭
- 性能测试的 `Concurrency` 值不应超过机器 CPU 核心数 × 2，避免测试结果失真
