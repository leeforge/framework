# Leeforge Framework

Leeforge Framework 是 Leeforge 后端服务的通用技术内核，提供认证授权、HTTP 工具、日志、缓存、插件系统等基础设施能力。

## 模块路径

```go
module github.com/leeforge/framework
```

## 安装

```bash
# 使用发布版本（推荐）
go get github.com/leeforge/framework@v0.1.0
```

**Monorepo 本地联调**，在 `server/backend/go.mod` 中添加：

```go
require github.com/leeforge/framework v0.0.0

replace github.com/leeforge/framework => ../framework
```

## 模块速览

| 模块 | 路径 | 说明 |
|---|---|---|
| **插件系统** | [`plugin`](./plugin/README.md) | 插件接口、AppContext、服务注册、事件总线 |
| **运行时** | [`runtime`](./runtime/README.md) | 插件生命周期管理、拓扑排序、优雅关闭 |
| **认证授权** | [`auth`](./auth/README.md) | JWT 中间件 + Casbin RBAC/ABAC |
| **HTTP 工具** | [`http`](./http/README.md) | 标准响应 responder、请求绑定 binding |
| **日志** | [`logging`](./logging/README.md) | 基于 zap 的结构化日志，支持日志轮转 |
| **配置** | [`config`](./config/README.md) | Viper 多环境配置、热重载、环境变量注入 |
| **缓存** | [`cache`](./cache/README.md) | 多级缓存（内存 + Redis）、多种缓存策略 |
| **Ent 实体** | [`entities`](./entities/README.md) | Ent Schema Mixin：审计字段、UUID v7、多租户 |
| **Ent 生成** | [`ent`](./ent/README.md) | Ent ORM 生成代码（CasbinPolicy、Media 等）|
| **权限元数据** | [`permission`](./permission/README.md) | 路由注册时附加权限码，供同步工具使用 |
| **路由组件** | [`middleware`](./middleware/README.md) | 网关限流、CORS、安全头、IP 黑白名单 |
| **指标** | [`metrics`](./metrics/README.md) | Counter/Gauge/Histogram 指标收集，Prometheus 导出 |
| **链路追踪** | [`tracing`](./tracing/README.md) | 分布式追踪 Span、采样策略、HTTP 中间件 |
| **并发工具** | [`concurrency`](./concurrency/README.md) | Worker Pool、信号量、速率限制器 |
| **安全工具** | [`security`](./security/README.md) | AES 加密、HMAC 签名、API Key 生成、密码验证 |
| **验证码** | [`captcha`](./captcha/README.md) | 数学/图片/滑块验证码生成与校验 |
| **媒体处理** | [`media`](./media/README.md) | 文件存储（本地/OSS）、图片处理、异步队列 |
| **字段组件** | [`component`](./component/README.md) | CMS 字段类型组件注册与管理 |
| **Redis 客户端** | [`redis_client`](./redis_client/README.md) | Redis 连接配置与初始化 |
| **HTTP 请求** | [`request`](./request/README.md) | HTTP 客户端、请求 ID 生成器 |
| **错误类型** | [`errors`](./errors/README.md) | 结构化错误类型，含错误码与 HTTP 状态码映射 |
| **JSON 工具** | [`json`](./json/README.md) | 高性能 JSON 序列化/反序列化封装 |
| **环境模式** | [`env_mode`](./env_mode/README.md) | 运行环境感知（dev / production / test）|
| **工具函数** | [`utils`](./utils/README.md) | 字符串转换、文件系统工具、路由打印 |
| **测试工具** | [`testing`](./testing/README.md) | HTTP 集成测试上下文、性能基准套件 |

## 快速开始

### 1. 初始化日志

```go
import "github.com/leeforge/framework/logging"

logger := logging.NewLogger(logging.DefaultConfig())
```

### 2. 加载配置

```go
import "github.com/leeforge/framework/config"

cfg, err := config.NewConfig()
var appCfg AppConfig
cfg.BindWithDefaults(&appCfg)
```

### 3. 初始化鉴权（Casbin）

```go
import frameAuth "github.com/leeforge/framework/auth"

core, err := frameAuth.Setup(ctx, frameAuth.Config{
    DatabaseURL: appCfg.Database.DSN,
    AutoMigrate: true,
    EnableCache: true,
})
```

### 4. 实现并注册插件

```go
import "github.com/leeforge/framework/plugin"

type UserPlugin struct{}

func (p *UserPlugin) Name() string { return "users" }
func (p *UserPlugin) Dependencies() []string { return []string{"auth"} }

func (p *UserPlugin) Setup(ctx context.Context, app plugin.AppContext) error {
    app.Router().Route("/users", func(r chi.Router) {
        permission.Get(r, "/", listHandler, permission.Private("列表", "users:read"))
    })
    return nil
}
```

### 5. 启动运行时

```go
import "github.com/leeforge/framework/runtime"

rt := runtime.New(appCtx)
rt.Register(&AuthPlugin{})
rt.Register(&UserPlugin{})

rt.Bootstrap(ctx)
rt.Wait()
rt.Shutdown(ctx)
```

## 响应格式契约

所有 API 响应统一使用以下结构（通过 `http/responder` 输出）：

```json
{
  "data": {},
  "error": null,
  "meta": { "traceId": "req-abc", "took": 12 }
}
```

## 架构规则

- **权限决策**必须通过 Casbin/RBAC 管理器，禁止在 handler 中硬编码权限逻辑
- **API 响应**统一使用 `http/responder`，禁止直接 `json.Marshal`
- **业务编排**留在上层插件，不进入 framework 核心层
- **UUID 主键**统一使用 `entities.NewUUID()`（UUID v7），禁止使用其他 UUID 库
- **错误处理**使用 `fmt.Errorf("context: %w", err)` 包装，保留错误链
