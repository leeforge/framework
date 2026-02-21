# Leeforge Framework

Leeforge Framework 是 Leeforge 后端服务的通用技术内核。

## 模块边界

该模块只承载基础设施能力：

- 认证与授权（基于 Casbin）
- 标准化 HTTP 响应输出（`http/responder`）
- 日志与可观测能力
- 插件运行时与公共中间件/工具

不承载：

- 业务编排与领域流程

## 模块路径

```go
module github.com/leeforge/framework
```

## 安装方式

### 使用发布版本（推荐）

```bash
go get github.com/leeforge/framework@v0.1.0
```

### Monorepo 本地联调

在 backend 的 `go.mod` 中添加：

```go
require github.com/leeforge/framework v0.0.0

replace github.com/leeforge/framework => ../framework
```

## 包结构速览

- `auth`: 鉴权初始化 + RBAC/ABAC 管理器
- `http/responder`: 统一 API 成功/失败响应
- `logging`: 基于 zap 的结构化日志抽象
- `plugin`: 插件运行时基础能力
- `permission`: 权限与域相关公共能力

## 快速开始

### 1. 初始化日志

```go
package main

import (
	"github.com/leeforge/framework/logging"
)

func newLogger() logging.Logger {
	cfg := logging.DefaultConfig()
	cfg.Level = "info"
	cfg.Format = "json"
	return logging.NewLogger(cfg)
}
```

### 2. 初始化鉴权核心（Casbin 链路）

```go
package main

import (
	"context"

	frameAuth "github.com/leeforge/framework/auth"
)

func setupAuth(ctx context.Context, databaseURL string) (*frameAuth.AuthCore, error) {
	return frameAuth.Setup(ctx, frameAuth.Config{
		DatabaseURL: databaseURL,
		AutoMigrate: true,
		EnableCache: true,
	})
}
```

### 3. 通过 RBAC 管理器校验权限

```go
package main

import (
	"context"

	frameAuth "github.com/leeforge/framework/auth"
)

func canReadUsers(ctx context.Context, core *frameAuth.AuthCore, userID string) (bool, error) {
	return core.CheckUserPermission(ctx, userID, "default", "users", "read")
}
```

### 4. 输出标准响应

```go
package main

import (
	"net/http"
	"time"

	"github.com/leeforge/framework/http/responder"
)

func listUsers(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	traceID := r.Header.Get("X-Trace-ID")

	users := []map[string]any{
		{"id": "u1", "name": "Alice"},
		{"id": "u2", "name": "Bob"},
	}

	pager := &responder.PaginationMeta{
		Page:       1,
		PageSize:   20,
		Total:      2,
		TotalPages: 1,
		HasMore:    false,
	}

	responder.WriteList(
		w,
		r,
		http.StatusOK,
		users,
		pager,
		responder.WithTraceID(traceID),
		responder.WithTook(time.Since(start).Milliseconds()),
	)
}

func createUser(w http.ResponseWriter, r *http.Request) {
	responder.Created(w, r, map[string]any{"id": "u3", "name": "Carol"})
}

func badInput(w http.ResponseWriter, r *http.Request, details any) {
	res := responder.New(w, r, nil)
	res.ValidationError(details)
}
```

## 标准响应结构

所有 API 响应都应保持以下结构：

```json
{
  "data": {},
  "error": null,
  "meta": {
    "traceId": "req-123",
    "took": 12
  }
}
```

错误响应示例：

```json
{
  "data": null,
  "error": {
    "code": 4002,
    "message": "Validation Failed",
    "details": [
      {"field": "name", "message": "required"}
    ]
  },
  "meta": {
    "traceId": "req-123",
    "took": 4
  }
}
```

## 集成约束

- 权限决策必须通过 Casbin/RBAC 管理器完成。
- 禁止在应用层 handler 硬编码权限逻辑。
- 所有 API 输出统一使用 `http/responder`，保持响应结构一致。
- 业务编排必须留在上层业务模块/插件，不进入 framework 核心层。

## 迁移说明

- Monorepo 迁移阶段可使用本地 `replace` 联调。
- 独立仓库发布版本后，下游服务应改为版本化 `require`，并移除本地 `replace`。
