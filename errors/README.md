# errors — 错误类型系统

提供结构化错误类型，支持错误码、HTTP 状态码映射、堆栈追踪与错误链包装，统一全框架的错误处理模式。

## 错误分类

| 类型常量 | 含义 |
|---|---|
| `ErrorTypeValidation` | 数据校验失败 |
| `ErrorTypeRequired` | 必填字段缺失 |
| `ErrorTypeInvalid` | 非法值/格式 |
| `ErrorTypeNotFound` | 资源不存在 |
| `ErrorTypeConflict` | 资源冲突 |
| `ErrorTypeUnauthorized` | 未认证 |
| `ErrorTypeForbidden` | 无权限 |
| `ErrorTypeDatabase` | 数据库错误 |
| `ErrorTypeInternal` | 内部错误 |
| `ErrorTypeTimeout` | 超时 |
| `ErrorTypeRateLimit` | 限流 |

## 快速开始

### 创建错误

```go
import framerrors "github.com/leeforge/framework/errors"

// 业务错误（带错误码）
err := framerrors.New(framerrors.ErrorTypeNotFound, "用户不存在")

// 带 HTTP 状态码
err := framerrors.NewWithStatus(framerrors.ErrorTypeUnauthorized, "Token 已过期", http.StatusUnauthorized)

// 包装下层错误（保留堆栈）
err := framerrors.Wrap(dbErr, framerrors.ErrorTypeDatabase, "查询用户失败")
```

### 错误判断

```go
var fe *framerrors.FrameworkError
if errors.As(err, &fe) {
    httpStatus := fe.HTTPStatus()
    code := fe.Code()
    msg := fe.Message()
}

// 快捷判断
if framerrors.IsNotFound(err) {
    responder.NotFound(w, r, "资源不存在")
    return
}

if framerrors.IsValidation(err) {
    res.ValidationError(err)
    return
}
```

### 错误包装（与标准库兼容）

```go
// 使用 fmt.Errorf 包装（推荐用于内部传递）
err = fmt.Errorf("createUser: %w", framerrors.New(framerrors.ErrorTypeConflict, "邮箱已存在"))

// 使用框架 Wrap（携带堆栈）
err = framerrors.Wrap(originalErr, framerrors.ErrorTypeInternal, "保存媒体失败")
```

## 在 Handler 中使用

```go
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
    user, err := h.svc.Create(ctx, dto)
    if err != nil {
        var fe *framerrors.FrameworkError
        if errors.As(err, &fe) {
            res := responder.New(w, r, nil)
            res.Error(fe.Code(), fe.Message(), nil)
            return
        }
        res.InternalError(err)
        return
    }
    responder.Created(w, r, user)
}
```

## 注意事项

- 错误包装使用 `%w`，确保 `errors.Is` / `errors.As` 能正常工作
- 禁止在 handler 中直接返回原始数据库错误，必须通过 `framerrors.Wrap` 包装后再传递
- 堆栈追踪仅在 `Wrap` 时生成，`New` 不自动附加堆栈（避免性能开销）
