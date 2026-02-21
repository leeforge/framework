# Logging Package

基于 [zap](https://github.com/uber-go/zap) 和 [lumberjack](https://github.com/natefinch/lumberjack) 的高性能结构化日志系统。

## 特性

- **高性能**: 基于 Uber 的 zap 日志库，零分配设计
- **日志轮转**: 使用 lumberjack 实现自动日志轮转（按大小、时间、数量）
- **按级别分文件**: 不同日志级别输出到不同文件（debug.log, info.log, error.log 等）
- **按日期分目录**: 日志文件按日期自动归档到子目录
- **多输出**: 支持同时输出到终端和文件
- **结构化日志**: 支持 JSON 和 Console 两种输出格式
- **上下文感知**: 自动从 context 提取 trace_id、span_id、request_id 等
- **HTTP 中间件**: 内置请求日志和 panic 恢复中间件
- **Hook 系统**: 支持自定义日志处理钩子
- **工厂模式**: 支持创建命名的子日志器

## 快速开始

### 基本使用

```go
package main

import (
    "github.com/JsonLee12138/leeforge/frame-core/logging"
    "go.uber.org/zap"
)

func main() {
    // 使用默认配置初始化全局日志器
    logging.Init(logging.DefaultConfig())
    defer logging.Sync()

    // 包级别函数 - 使用全局日志器
    logging.Info("应用启动", zap.String("version", "1.0.0"))
    logging.Debug("调试信息")
    logging.Warn("警告信息")
    logging.Error("错误信息", zap.Error(err))

    // 格式化日志
    logging.Infof("用户 %s 登录成功", username)
    logging.Errorf("请求失败: %v", err)
}
```

### 自定义配置

```go
config := logging.Config{
    Director:       "logs",           // 日志目录
    Level:          "debug",          // 最低日志级别
    Format:         "json",           // 输出格式: json 或 console
    LogInTerminal:  true,             // 是否同时输出到终端
    MaxSize:        100,              // 单文件最大大小 (MB)
    MaxBackups:     10,               // 保留的旧文件数量
    MaxAge:         7,                // 保留天数
    Compress:       true,             // 是否压缩旧文件
    ShowLineNumber: true,             // 是否显示调用位置
    TimeFormat:     "2006/01/02 - 15:04:05",
    EncodeLevel:    "LowercaseLevelEncoder", // 级别编码器
}

logging.Init(config)
```

### 创建独立日志器

```go
// 创建独立的日志器实例
logger := logging.NewLogger(config)
logger.Info("独立日志器")

// 创建子日志器
childLogger := logger.With(zap.String("module", "user"))
childLogger.Info("用户模块日志")

// 命名日志器
namedLogger := logger.Named("auth")
namedLogger.Info("认证模块日志")

// 错误日志器
errLogger := logger.WithError(err)
errLogger.Error("操作失败")
```

## 配置项说明

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `Director` | string | `"logs"` | 日志文件存储目录 |
| `Level` | string | `"info"` | 最低日志级别 (debug/info/warn/error/dpanic/panic/fatal) |
| `Format` | string | `"json"` | 输出格式 (json/console) |
| `LogInTerminal` | bool | `true` | 是否同时输出到终端 |
| `MaxSize` | int | `100` | 单个日志文件最大大小 (MB) |
| `MaxBackups` | int | `10` | 保留的旧日志文件数量 |
| `MaxAge` | int | `7` | 日志文件保留天数 |
| `Compress` | bool | `true` | 是否压缩归档的日志文件 |
| `ShowLineNumber` | bool | `true` | 是否在日志中显示调用位置 |
| `TimeFormat` | string | `"2006/01/02 - 15:04:05"` | 时间格式 |
| `Prefix` | string | `""` | 日志前缀 |
| `EncodeLevel` | string | `"LowercaseLevelEncoder"` | 级别编码器 |

### EncodeLevel 可选值

- `LowercaseLevelEncoder` - 小写 (info, error)
- `LowercaseColorLevelEncoder` - 小写带颜色
- `CapitalLevelEncoder` - 大写 (INFO, ERROR)
- `CapitalColorLevelEncoder` - 大写带颜色

## 日志文件结构

```
logs/
├── 2026-01-20/
│   ├── debug.log
│   ├── info.log
│   ├── warn.log
│   ├── error.log
│   └── fatal.log
├── 2026-01-21/
│   ├── debug.log
│   ├── info.log
│   └── ...
```

## 上下文日志

### 设置上下文信息

```go
ctx := context.Background()
ctx = logging.SetTraceID(ctx, "trace-123")
ctx = logging.SetSpanID(ctx, "span-456")
ctx = logging.SetRequestID(ctx, "req-789")
ctx = logging.SetUserID(ctx, "user-abc")
```

### 从上下文创建日志器

```go
// 自动提取 context 中的 trace_id, span_id 等字段
ctxLogger := logging.WithContext(logger, ctx)
ctxLogger.Info("带上下文的日志")
// 输出: {"message":"带上下文的日志","trace_id":"trace-123","span_id":"span-456",...}
```

### 在上下文中存储/获取日志器

```go
// 存储日志器到 context
ctx = logging.ToContext(ctx, logger)

// 从 context 获取日志器 (如果没有则返回全局日志器)
logger := logging.FromContext(ctx)
```

## HTTP 中间件

### 请求日志中间件

```go
import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/JsonLee12138/leeforge/frame-core/logging"
)

func main() {
    logger := logging.NewLogger(config)

    r := chi.NewRouter()

    // 添加请求日志中间件
    r.Use(logging.HTTPMiddleware(logger))

    // 添加 panic 恢复中间件
    r.Use(logging.RecoveryMiddleware(logger))

    r.Get("/api/users", handleUsers)
    http.ListenAndServe(":8080", r)
}
```

中间件会自动记录：
- 请求开始：method, path, query, remote_addr, user_agent
- 请求完成：method, path, status, duration, bytes

### 在 Handler 中使用日志器

```go
func handleUsers(w http.ResponseWriter, r *http.Request) {
    // 从 context 获取日志器 (已包含请求上下文信息)
    logger := logging.FromContext(r.Context())

    logger.Info("处理用户请求", zap.Int("user_count", len(users)))
}
```

## 工厂模式

用于管理多个命名的日志器实例：

```go
// 创建工厂
factory := logging.NewFactory(config)

// 获取命名日志器 (相同名称返回相同实例)
userLogger := factory.GetLogger("user-service")
orderLogger := factory.GetLogger("order-service")

userLogger.Info("用户服务日志")
orderLogger.Info("订单服务日志")
```

## Hook 系统

添加自定义日志处理钩子：

```go
import "go.uber.org/zap/zapcore"

// 定义 hook
alertHook := func(entry zapcore.Entry) error {
    if entry.Level >= zapcore.ErrorLevel {
        // 发送告警通知
        sendAlert(entry.Message)
    }
    return nil
}

// 添加单个 hook
hookedLogger := logging.WithHook(logger, alertHook)

// 添加多个 hooks
hookedLogger := logging.WithHooks(logger, alertHook, metricsHook)
```

## 访问底层 zap 日志器

```go
logger := logging.NewLogger(config)

// 获取底层 *zap.Logger
zapLogger := logger.Zap()

// 获取底层 *zap.SugaredLogger
sugarLogger := logger.Sugar()

// 使用 zap 的高级特性
zapLogger.With(zap.Namespace("request")).Info("namespaced log")
```

## 全局日志器

```go
// 初始化全局日志器
logging.Init(config)

// 获取全局日志器
logger := logging.Global()

// 替换全局日志器
logging.SetGlobal(newLogger)

// 包级别函数直接使用全局日志器
logging.Info("message")
logging.Debug("debug")
logging.Error("error")
logging.Infof("formatted %s", "message")
logging.Debugf("debug %d", 123)

// 刷新缓冲
logging.Sync()
```

## 清理资源

```go
// 应用退出时刷新日志缓冲
defer logging.Sync()

// 关闭所有文件写入器 (可选)
defer logging.CloseAllWriters()
```

## 日志输出示例

### JSON 格式

```json
{"level":"info","time":"2026/01/20 - 15:04:05","caller":"main.go:42","message":"用户登录","user_id":"123","ip":"192.168.1.1"}
{"level":"error","time":"2026/01/20 - 15:04:06","caller":"auth.go:89","message":"认证失败","error":"invalid token","trace_id":"abc-123"}
```

### Console 格式

```
2026/01/20 - 15:04:05	info	main.go:42	用户登录	{"user_id": "123", "ip": "192.168.1.1"}
2026/01/20 - 15:04:06	error	auth.go:89	认证失败	{"error": "invalid token", "trace_id": "abc-123"}
```

## YAML 配置示例

```yaml
log:
  director: logs
  level: info
  format: json
  log-in-terminal: true
  max-size: 100
  max-backups: 10
  max-age: 7
  compress: true
  show-line-number: true
  time-format: "2006/01/02 - 15:04:05"
  encode-level: LowercaseLevelEncoder
```

## 最佳实践

1. **初始化**: 在应用启动时调用 `logging.Init(config)` 初始化全局日志器
2. **清理**: 使用 `defer logging.Sync()` 确保退出时刷新日志缓冲
3. **上下文**: 在 HTTP 处理中使用 `FromContext` 获取带请求信息的日志器
4. **命名**: 为不同模块创建命名日志器便于区分和过滤
5. **级别**: 生产环境使用 `info` 级别，开发环境使用 `debug` 级别
6. **格式**: 生产环境使用 `json` 格式便于日志分析工具解析
