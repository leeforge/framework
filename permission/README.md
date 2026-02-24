# permission — 路由权限元数据

为 Chi Router 路由注册提供权限元数据（`Meta`），供权限同步工具、文档生成与中间件鉴权读取。

## 功能

- 用 `permission.Get/Post/Put/Delete/...` 替代 `r.Get/Post/...`，在注册路由时附加权限描述
- 中间件可通过 `permission.ExtractMeta(handler)` 从 handler 中提取元数据
- 支持标记路由为公开（`IsPublic: true`）或受保护

## 快速开始

```go
import "github.com/leeforge/framework/permission"

func (h *UserHandler) Routes(r chi.Router) {
    // 公开路由（无需鉴权）
    permission.Post(r, "/auth/login", h.Login, permission.Public("用户登录"))

    // 受保护路由（需要 users:read 权限）
    permission.Get(r, "/users", h.List, permission.Private("获取用户列表", "users:read"))
    permission.Get(r, "/users/{id}", h.Get, permission.Private("获取用户详情", "users:read"))
    permission.Post(r, "/users", h.Create, permission.Private("创建用户", "users:write"))
    permission.Put(r, "/users/{id}", h.Update, permission.Private("更新用户", "users:write"))
    permission.Delete(r, "/users/{id}", h.Delete, permission.Private("删除用户", "users:delete"))
}
```

## Meta 结构

```go
type Meta struct {
    Description string   // 接口描述（用于文档）
    IsPublic    bool     // 是否公开（跳过鉴权）
    Permissions []string // 所需权限码列表（如 "users:read"）
}
```

## 创建元数据

```go
// 公开路由
meta := permission.Public("用户注册")
meta := permission.Public("健康检查")

// 受保护路由（支持多权限码）
meta := permission.Private("发布文章", "articles:write", "articles:publish")
```

## 提取元数据（在中间件中使用）

```go
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        meta, ok := permission.ExtractMeta(next)
        if ok && meta.IsPublic {
            next.ServeHTTP(w, r) // 公开路由跳过鉴权
            return
        }
        // 继续鉴权逻辑...
    })
}
```

## 与权限同步工具配合

框架的权限同步工具（`tools/permission-syncer`）会遍历所有注册路由，提取 `Meta` 并同步到 `APIPermission` 实体：

```
新增路由 → permission.Private(...) → 权限同步工具 → APIPermission 表 → Casbin 规则
```

## 注意事项

- 所有需要鉴权的路由都应使用此包注册，**禁止**使用裸 `r.Get/Post/...`，否则权限信息丢失
- 权限码命名约定：`{资源}:{操作}`，如 `users:read`、`articles:write`
- `permission.ExtractMeta` 能穿透 Chi 的 `ChainHandler` 包装层
