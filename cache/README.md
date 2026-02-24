# cache — 多级缓存

提供统一的缓存接口（`Cache`），内置内存缓存与 Redis 缓存适配器，支持多种缓存策略（Write-Through、Write-Back、LRU/LFU 等）。

## 核心接口

```go
type Cache interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    Flush(ctx context.Context) error
}
```

## 快速开始

### 内存缓存

```go
import "github.com/leeforge/framework/cache"

// 创建 LRU 内存缓存（容量 1000 条）
c := cache.NewMemoryCache(cache.MemoryConfig{
    MaxSize: 1000,
    Policy:  cache.LRU,
})

// 读写
c.Set(ctx, "user:123", data, 5*time.Minute)
val, err := c.Get(ctx, "user:123")
```

### Redis 缓存

```go
// 基于 redis_client 初始化
rc := cache.NewRedisCache(redisClient, cache.RedisConfig{
    KeyPrefix: "myapp:",
})
```

### 二级缓存（L1 内存 + L2 Redis）

```go
twoLevel := cache.NewTwoLevelCache(
    cache.NewMemoryCache(...), // L1：本地内存，速度最快
    cache.NewRedisCache(...),  // L2：Redis，跨实例共享
    cache.TwoLevelConfig{
        L1TTL: 1 * time.Minute,
        L2TTL: 30 * time.Minute,
    },
)
```

## 缓存策略

| 策略 | 说明 |
|---|---|
| `WriteThrough` | 写时同时更新缓存和存储，强一致性 |
| `WriteBack` | 先写缓存，异步刷盘，高吞吐 |
| `WriteAround` | 绕过缓存直接写存储，适合一次写多次读 |
| `LRU` | 最近最少使用淘汰策略 |
| `LFU` | 最低使用频率淘汰策略 |

```go
strategy := cache.NewWriteThroughStrategy(store, cacheInstance)
strategy.Write(ctx, "key", value)
```

## 适配器接口

`BackendAdapter` 用于统一不同缓存后端，可自定义实现：

```go
type BackendAdapter interface {
    Get(ctx context.Context, key string) ([]byte, error)
    Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
    Delete(ctx context.Context, key string) error
}
```

## 注意事项

- 在插件中通过 `AppContext.Cache()` 获取框架内置的缓存实例
- 二级缓存的 L1 TTL 应小于 L2 TTL，防止数据不一致
- Key 命名建议使用 `{模块}:{类型}:{ID}` 格式，如 `user:profile:123`
