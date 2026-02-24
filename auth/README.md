# auth — 认证与授权

基于 Casbin 实现的 RBAC/ABAC 鉴权核心，提供 JWT 中间件、权限检查器与角色管理器。

## 功能

- JWT 认证中间件（解析 Bearer Token，注入用户上下文）
- 基于 Casbin 的 RBAC 权限引擎（角色→资源→操作）
- Ent ORM 适配器持久化 Casbin 规则
- 角色分配、权限检查、策略管理 API

## 快速开始

### 初始化鉴权核心

```go
import frameAuth "github.com/leeforge/framework/auth"

core, err := frameAuth.Setup(ctx, frameAuth.Config{
    DatabaseURL: "postgres://...",
    AutoMigrate: true,   // 自动迁移 CasbinRule 表
    EnableCache: true,   // 启用策略缓存（提升性能）
})
```

### 注册 JWT 中间件

```go
r := chi.NewRouter()
r.Use(core.AuthMiddleware())

// 受保护路由
r.Group(func(r chi.Router) {
    r.Get("/users", listUsersHandler)
})
```

### 权限检查

```go
// 检查用户是否对某域内资源有操作权限
allowed, err := core.CheckUserPermission(ctx, userID, domain, resource, action)

// 为用户分配角色
err = core.AssignRole(ctx, userID, domain, "editor")

// 为角色授予权限
err = core.GrantPermission(ctx, domain, "editor", "articles", "write")
```

## 配置项

```go
type Config struct {
    DatabaseURL string // Casbin 策略存储数据库 URL
    AutoMigrate bool   // 是否自动迁移 casbin_rules 表
    EnableCache bool   // 是否启用内存策略缓存
    JWTSecret   string // JWT 签名密钥
    JWTExpiry   time.Duration // Token 过期时间
}
```

## 中间件执行流程

```
请求 → JWT 验证 → 解析用户ID/域 → Casbin 策略检查 → 通过 → 继续
                                                       ↓ 拒绝 → 401/403
```

## 注意事项

- 所有权限决策必须通过此模块，**禁止**在 handler 中硬编码权限逻辑
- `CasbinRule` 实体由框架管理，业务层通过 `AuthCore` API 操作策略，不要直接操作该表
- 策略变更后缓存会自动失效（若启用 `EnableCache`）
