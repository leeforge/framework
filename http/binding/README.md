# HTTP Binding 模块

## 📚 文档导航

### 核心功能
- **[JSON 绑定](./docs/json/JSON_OPTIONS.md)** - JSON 请求体绑定与解码选项
- **[Query 绑定](./docs/query/README.md)** - Query 参数绑定完整指南
- **[Query 使用示例](./docs/query/examples.md)** - 丰富的使用示例
- **[Query 详细指南](./docs/query/usage.md)** - 深度使用指南

### 模块概述

`frame-core/http/binding` 包提供强大的 HTTP 数据绑定功能：

- ✅ **JSON 绑定** - 请求体 JSON 到结构体的自动绑定 + 验证
- ✅ **Query 绑定** - 高性能查询参数解析（~1μs）
- ✅ **数据校验** - 集成 validator/v10
- ✅ **错误处理** - 详细的字段级错误信息
- ✅ **灵活配置** - 支持自定义选项和策略

## 🚀 快速开始

### 1. JSON 绑定（请求体）

#### 定义请求结构体

```go
type CreateUserRequest struct {
    Name     string `json:"name" validate:"required,min=2,max=50"`
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age" validate:"required,gte=0,lte=120"`
    Password string `json:"password" validate:"required,min=8"`
    Role     string `json:"role" validate:"oneof=admin user guest"`
}
```

### 2. 在处理函数中使用

```go
package main

import (
    "net/http"
    "github.com/JsonLee12138/headless-cms/core/http/bind"
)

func CreateUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest

    // 绑定和校验请求数据
    if err := bind.JSON(r, &req); err != nil {
        // 处理绑定或校验错误
        handleBindError(w, err)
        return
    }

    // 数据已成功绑定和校验，继续业务逻辑
    // ...
}
```

### 3. 错误处理

```go
func handleBindError(w http.ResponseWriter, err error) {
    switch e := err.(type) {
    case *bind.BindError:
        // 单个绑定错误
        writeError(w, http.StatusBadRequest, e.Message)
    case bind.ValidationErrors:
        // 多个校验错误
        writeValidationErrors(w, e)
    default:
        writeError(w, http.StatusBadRequest, "请求格式错误")
    }
}
```

## API 参考

### 核心函数

#### `JSON(r *http.Request, v interface{}) error`

绑定 HTTP 请求体的 JSON 数据到指定的结构体，并进行数据校验。

**参数:**
- `r *http.Request`: HTTP 请求对象
- `v interface{}`: 目标结构体指针

**返回值:**
- `error`: 绑定或校验错误，如果成功则返回 nil

**示例:**
```go
var user User
if err := bind.JSON(r, &user); err != nil {
    // 处理错误
}
```

### 错误类型

#### `BindError`

表示绑定过程中的错误（JSON 解析、请求体读取等）。

```go
type BindError struct {
    Type    string `json:"type"`     // 错误类型
    Message string `json:"message"`  // 错误信息
    Field   string `json:"field,omitempty"` // 相关字段（可选）
}
```

#### `ValidationErrors`

表示数据校验错误的集合。

```go
type ValidationErrors []BindError
```

### 工具函数

#### `SetValidator(v *validator.Validate)`

设置自定义的校验器实例。

#### `GetValidator() *validator.Validate`

获取当前的校验器实例。

## 支持的校验标签

| 标签 | 说明 | 示例 |
|------|------|------|
| `required` | 必填字段 | `validate:"required"` |
| `email` | 邮箱格式 | `validate:"email"` |
| `min` | 最小长度/值 | `validate:"min=3"` |
| `max` | 最大长度/值 | `validate:"max=100"` |
| `len` | 固定长度 | `validate:"len=11"` |
| `gte` | 大于等于 | `validate:"gte=0"` |
| `lte` | 小于等于 | `validate:"lte=120"` |
| `gt` | 大于 | `validate:"gt=0"` |
| `lt` | 小于 | `validate:"lt=100"` |
| `alphanum` | 只包含字母数字 | `validate:"alphanum"` |
| `alpha` | 只包含字母 | `validate:"alpha"` |
| `numeric` | 数字格式 | `validate:"numeric"` |
| `url` | URL 格式 | `validate:"url"` |
| `uri` | URI 格式 | `validate:"uri"` |
| `oneof` | 枚举值 | `validate:"oneof=admin user guest"` |

## 完整示例

### 定义结构体

```go
// 用户创建请求
type CreateUserRequest struct {
    Name     string `json:"name" validate:"required,min=2,max=50"`
    Email    string `json:"email" validate:"required,email"`
    Age      int    `json:"age" validate:"required,gte=0,lte=120"`
    Password string `json:"password" validate:"required,min=8"`
    Role     string `json:"role" validate:"required,oneof=admin user guest"`
    Phone    string `json:"phone" validate:"len=11,numeric"`
    Website  string `json:"website,omitempty" validate:"omitempty,url"`
}

// 文章创建请求
type CreateArticleRequest struct {
    Title    string   `json:"title" validate:"required,min=1,max=200"`
    Content  string   `json:"content" validate:"required,min=10"`
    Tags     []string `json:"tags" validate:"required,min=1,max=10"`
    Status   string   `json:"status" validate:"required,oneof=draft published archived"`
    AuthorID uint     `json:"author_id" validate:"required,gt=0"`
}
```

### 控制器实现

```go
package controllers

import (
    "net/http"
    "github.com/JsonLee12138/headless-cms/core/http/bind"
    "github.com/JsonLee12138/headless-cms/core/http/responder"
)

type UserController struct {
    responderFactory *responder.ResponderFactory
}

func (c *UserController) CreateUser(w http.ResponseWriter, r *http.Request) {
    resp := c.responderFactory.New(w, r)

    var req CreateUserRequest
    if err := bind.JSON(r, &req); err != nil {
        c.handleBindError(resp, err)
        return
    }

    // 业务逻辑...
    user, err := c.userService.Create(r.Context(), req)
    if err != nil {
        resp.WriteError(http.StatusInternalServerError, responder.Error{
            Code:    "CREATE_FAILED",
            Message: "用户创建失败",
        })
        return
    }

    resp.Write(http.StatusCreated, user)
}

func (c *UserController) handleBindError(resp *responder.Responder[any], err error) {
    switch e := err.(type) {
    case *bind.BindError:
        resp.WriteError(http.StatusBadRequest, responder.Error{
            Code:    e.Type,
            Message: e.Message,
        })
    case bind.ValidationErrors:
        // 构造详细的校验错误信息
        var fields []responder.ErrorField
        for _, ve := range e {
            fields = append(fields, responder.ErrorField{
                Field:   ve.Field,
                Message: ve.Message,
            })
        }
        resp.WriteError(http.StatusBadRequest, responder.Error{
            Code:    "VALIDATION_FAILED",
            Message: "数据校验失败",
            Fields:  fields,
        })
    default:
        resp.WriteError(http.StatusBadRequest, responder.Error{
            Code:    "BIND_ERROR",
            Message: "请求数据格式错误",
        })
    }
}
```

### 路由配置

```go
package routes

import (
    "github.com/go-chi/chi/v5"
    "github.com/JsonLee12138/headless-cms/core/http/bind"
)

func SetupUserRoutes(r chi.Router, controller *UserController) {
    r.Route("/users", func(r chi.Router) {
        r.Post("/", controller.CreateUser)
        r.Put("/{id}", controller.UpdateUser)
    })
}
```

## 自定义校验器

### 注册自定义校验规则

```go
package main

import (
    "github.com/JsonLee12138/headless-cms/core/http/bind"
    "github.com/go-playground/validator/v10"
)

func init() {
    v := bind.GetValidator()

    // 注册自定义校验规则
    v.RegisterValidation("phone", validatePhone)
    v.RegisterValidation("idcard", validateIDCard)
}

func validatePhone(fl validator.FieldLevel) bool {
    phone := fl.Field().String()
    // 实现手机号校验逻辑
    return len(phone) == 11 && phone[0] == '1'
}

func validateIDCard(fl validator.FieldLevel) bool {
    idcard := fl.Field().String()
    // 实现身份证号校验逻辑
    return len(idcard) == 18
}
```

### 使用自定义校验规则

```go
type UserRequest struct {
    Name   string `json:"name" validate:"required,min=2"`
    Phone  string `json:"phone" validate:"required,phone"`
    IDCard string `json:"id_card" validate:"required,idcard"`
}
```

## 最佳实践

### 1. 结构体设计

- 使用明确的 JSON 标签
- 合理设置校验规则
- 为可选字段使用 `omitempty`

```go
type UpdateUserRequest struct {
    Name     *string `json:"name,omitempty" validate:"omitempty,min=2,max=50"`
    Email    *string `json:"email,omitempty" validate:"omitempty,email"`
    Age      *int    `json:"age,omitempty" validate:"omitempty,gte=0,lte=120"`
}
```

### 2. 错误处理

- 区分不同类型的错误
- 提供用户友好的错误信息
- 记录详细的日志用于调试

### 3. 性能优化

- 复用校验器实例
- 避免在热路径上创建过多临时对象

## 故障排除

### 常见问题

**Q: 为什么校验不生效？**
A: 检查结构体字段是否为导出字段（首字母大写），以及 `validate` 标签是否正确。

**Q: 如何处理嵌套结构体？**
A: 在嵌套字段上使用 `dive` 标签：
```go
type User struct {
    Address Address `json:"address" validate:"required,dive"`
}
```

**Q: 如何跳过某些字段的校验？**
A: 使用 `omitempty` 或 `-` 标签：
```go
type User struct {
    Name     string `json:"name" validate:"required"`
    Internal string `json:"-" validate:"-"`
}
```

## 依赖

- `github.com/JsonLee12138/headless-cms/core/json` - 项目自定义 JSON 序列化
- `github.com/go-playground/validator/v10` - 数据校验库

## 许可证

本项目遵循项目主许可证。
