# http — HTTP 工具集

提供标准化 API 响应输出、请求绑定/校验与 HTTP 中间件。

## 子包

| 包 | 路径 | 功能 |
|---|---|---|
| `responder` | `http/responder` | 统一成功/失败响应格式输出 |
| `binding` | `http/binding` | 请求 JSON/Query 绑定与校验 |
| `middleware` | `http/middleware` | TraceID 注入、请求耗时记录 |

---

## responder — 标准响应

所有 API 响应必须使用此包，**禁止**直接 `json.Marshal` 返回原生结构。

### 响应格式

```json
{
  "data": {},
  "error": null,
  "meta": {
    "traceId": "req-abc123",
    "took": 12
  }
}
```

### 常用方法

```go
import "github.com/leeforge/framework/http/responder"

// 返回单条记录
responder.OK(w, r, user)

// 返回列表（带分页）
responder.WriteList(w, r, http.StatusOK, users, &responder.PaginationMeta{
    Page: 1, PageSize: 20, Total: 100, TotalPages: 5, HasMore: true,
}, responder.WithTraceID(traceID), responder.WithTook(took))

// 创建成功 (201)
responder.Created(w, r, newUser)

// 无内容 (204)
responder.NoContent(w, r)

// 校验错误 (422)
res := responder.New(w, r, nil)
res.ValidationError(validationDetails)

// 自定义错误码
res.Error(4001, "用户不存在", nil)
```

### 错误码规范

| 范围 | 含义 |
|---|---|
| 4xxx | 客户端错误（参数错误、未授权等） |
| 5xxx | 服务端错误（内部错误、依赖服务失败等） |

---

## binding — 请求绑定

```go
import "github.com/leeforge/framework/http/binding"

// 绑定 JSON Body
var req CreateUserDTO
if err := binding.BindJSON(r, &req); err != nil {
    // err 为 ValidationErrors 类型，包含字段级别错误信息
    res.ValidationError(err)
    return
}

// 绑定 Query 参数
var query ListQuery
if err := binding.BindQuery(r, &query); err != nil {
    res.ValidationError(err)
    return
}
```

DTO 支持 `validate` tag（基于 `go-playground/validator/v10`）：

```go
type CreateUserDTO struct {
    Name  string `json:"name"  validate:"required,min=2,max=50"`
    Email string `json:"email" validate:"required,email"`
}
```

---

## middleware

```go
// 注入 TraceID（从请求头读取或生成新 ID）
r.Use(httpMiddleware.TraceID)

// 记录请求耗时
r.Use(httpMiddleware.Timing)
```

## 注意事项

- `responder` 方法已内置错误处理，无需在 handler 中再次 `w.WriteHeader`
- 绑定失败的 `ValidationErrors` 可直接传给 `res.ValidationError` 输出标准格式
