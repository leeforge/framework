# runtime — 插件运行时

负责插件的生命周期管理：依赖解析（拓扑排序）、有序初始化与优雅关闭。

## 功能

- 注册并管理所有插件
- 基于依赖声明自动拓扑排序，确保依赖先于依赖方初始化
- 并发安全的插件状态跟踪
- 信号监听 + 优雅关闭（按逆序调用 `Teardown`）

## 快速开始

```go
package main

import (
    "context"
    "github.com/leeforge/framework/runtime"
)

func main() {
    rt := runtime.New(appCtx) // appCtx 实现 plugin.AppContext

    // 注册插件（顺序无关，运行时会自动拓扑排序）
    rt.Register(&AuthPlugin{})
    rt.Register(&UserPlugin{})   // 依赖 AuthPlugin
    rt.Register(&MediaPlugin{})

    // 启动：按依赖顺序调用各插件的 Setup
    if err := rt.Bootstrap(context.Background()); err != nil {
        log.Fatalf("bootstrap failed: %v", err)
    }

    // 阻塞直到收到 SIGINT / SIGTERM
    rt.Wait()

    // 优雅关闭：按逆序调用各插件的 Teardown
    rt.Shutdown(context.Background())
}
```

## 事件总线

运行时内置 `EventBus`，插件可通过 `AppContext.EventBus()` 访问：

```go
// 在插件 A 中发布事件
app.EventBus().Publish(ctx, "order.created", orderPayload)

// 在插件 B 中订阅事件
app.EventBus().Subscribe("order.created", func(ctx context.Context, e plugin.Event) error {
    // 异步处理订单创建事件
    return nil
})
```

## 错误处理

- `Bootstrap` 时任意插件的 `Setup` 失败，会立即返回错误，已初始化的插件会按逆序调用 `Teardown`
- 循环依赖在 `Bootstrap` 前即被检测，返回描述性错误

## 注意事项

- 不要在 `Setup` 内启动长时间阻塞操作，应使用 goroutine 并在 `Teardown` 中优雅停止
- `EventBus` 的事件处理器应当幂等，避免重复消费导致副作用
