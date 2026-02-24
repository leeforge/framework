# env_mode — 运行环境模式

通过 `GO_ENV_MODE` 环境变量感知当前运行环境（development / production / test），供配置加载等模块使用。

## 环境常量

| 常量 | 值 | 说明 |
|---|---|---|
| `DevMode` | `"development"` | 开发环境 |
| `ProMode` | `"production"` | 生产环境 |
| `TestMode` | `"test"` | 测试环境 |

## 使用方式

```go
import "github.com/leeforge/framework/env_mode"

// 获取当前环境
mode := env_mode.Mode()

switch mode {
case env_mode.DevMode:
    // 开发模式：启用热重载、详细日志
case env_mode.ProMode:
    // 生产模式：启用缓存、JSON 日志
case env_mode.TestMode:
    // 测试模式：使用内存数据库
}

// 快捷判断
if env_mode.IsDev() {
    logger.SetLevel("debug")
}

if env_mode.IsProd() {
    logger.SetLevel("info")
}
```

## 设置环境

通过环境变量设置（支持多种简写）：

```bash
# 开发环境
GO_ENV_MODE=development  # 或 dev、Dev

# 生产环境
GO_ENV_MODE=production   # 或 pro、prod、Pro、Prod

# 测试环境
GO_ENV_MODE=test
```

## 默认值

若 `GO_ENV_MODE` 未设置，默认为 `development`。

## 注意事项

- 框架的 `config` 模块依赖此包自动加载环境对应的配置文件
- 仅在应用启动前通过环境变量设置，运行中不应动态修改
