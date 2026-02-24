# plugin — 插件系统

提供插件的接口定义、生命周期管理、服务注册、事件总线与应用上下文，是整个框架的扩展骨架。

## 核心概念

| 概念 | 说明 |
|---|---|
| `Plugin` | 插件接口，每个业务模块实现此接口注入框架 |
| `AppContext` | 插件运行时可访问的全局上下文（DB、Cache、Logger 等） |
| `ServiceRegistry` | 服务注册表，用于插件间依赖注入 |
| `EventBus` | 发布/订阅事件总线，实现模块间解耦通信 |
| `Domain` | 多租户隔离域 |

## 主要接口

### Plugin 接口

```go
type Plugin interface {
    // Name 返回插件唯一标识，用于依赖解析
    Name() string
    // Dependencies 返回该插件依赖的其他插件 Name 列表
    Dependencies() []string
    // Setup 插件初始化入口，接收应用上下文
    Setup(ctx context.Context, app AppContext) error
    // Teardown 插件清理（可选），在关闭时调用
    Teardown(ctx context.Context) error
}
```

### AppContext 接口

```go
type AppContext interface {
    DB() *ent.Client
    Cache() cache.Cache
    Logger() logging.Logger
    Config() *config.Config
    EventBus() EventBus
    Services() ServiceRegistry
    Router() chi.Router
}
```

### EventBus

```go
// 订阅事件
bus.Subscribe("user.created", func(ctx context.Context, e Event) error {
    // 处理事件
    return nil
})

// 发布事件
bus.Publish(ctx, "user.created", map[string]any{"userID": "..."})
```

### ServiceRegistry

```go
// 注册服务
registry.Register("emailService", &EmailService{})

// 获取服务
svc := registry.Get("emailService").(*EmailService)
```

## 实现插件

```go
package myplugin

import (
    "context"
    "github.com/leeforge/framework/plugin"
)

type MyPlugin struct{}

func (p *MyPlugin) Name() string { return "my-plugin" }

func (p *MyPlugin) Dependencies() []string {
    return []string{"auth"} // 声明依赖
}

func (p *MyPlugin) Setup(ctx context.Context, app plugin.AppContext) error {
    logger := app.Logger()
    logger.Info("my-plugin initialized")

    // 注册服务
    app.Services().Register("myService", &MyService{db: app.DB()})

    // 注册路由
    app.Router().Route("/my-resource", func(r chi.Router) {
        r.Get("/", handleList)
    })

    return nil
}

func (p *MyPlugin) Teardown(ctx context.Context) error {
    return nil
}
```

## 注意事项

- 插件 `Name()` 必须全局唯一
- 循环依赖会导致运行时 panic，框架会在启动时做拓扑排序检测
- `Setup` 返回错误会中止整个应用启动流程
- `Teardown` 保证按依赖逆序调用
