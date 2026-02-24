# utils — 工具函数集

提供字符串转换（驼峰/下划线互转）、文件系统工具与路由打印等通用工具函数。

## 字符串工具 (`utils/strings.go`)

```go
import "github.com/leeforge/framework/utils"

// snake_case → lowerCamelCase
utils.LowerCamelCase("created_by_id") // → "createdById"
utils.LowerCamelCase("user_name")      // → "userName"

// snake_case / lowerCamelCase → UpperCamelCase
utils.UpperCamelCase("user_name")      // → "UserName"
utils.UpperCamelCase("userName")       // → "UserName"

// lowerCamelCase / UpperCamelCase → snake_case
utils.SnakeCase("createdById")         // → "created_by_id"
utils.SnakeCase("UserName")            // → "user_name"
```

此工具在 `entities` 模块中用于自动生成 JSON tag（`IDField` 函数依赖 `LowerCamelCase`）。

## 文件系统工具 (`utils/files.go`)

```go
// 获取当前调用者文件所在目录
dir := utils.Dirname()

// 检查路径是否存在及类型
isDir, exists, err := utils.Exists("/path/to/file")
```

## 路由打印 (`utils/print.go`)

在开发模式下打印已注册的 Chi 路由表：

```go
r := chi.NewRouter()
// ... 注册路由

// 打印所有路由
utils.PrintRoutes(r)
// 输出：
// GET    /users
// POST   /users
// GET    /users/{id}
// PUT    /users/{id}
// DELETE /users/{id}
```

## 错误工具 (`utils/errors.go`)

```go
// 将 panic 的 error 包装为带堆栈的 error
err := utils.RecoverError(panicValue)
```

## 注意事项

- `LowerCamelCase` 与 `SnakeCase` 等转换函数依赖 `golang.org/x/text`
- `Dirname()` 基于 `runtime.Caller(1)`，调用层级固定，不可嵌套调用
