# redis_client — Redis 客户端

提供 Redis 连接配置与连接池初始化，基于 `go-redis/redis` 封装。

## 配置

```go
type Config struct {
    Host     string // Redis 主机（如 "localhost"）
    Port     string // 端口（如 "6379"）
    Password string // 认证密码（无密码时为空）
    DB       int    // 数据库编号（默认 0）
}
```

## 快速开始

### 在配置文件中声明

```yaml
# config/config.yaml
redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
```

### 初始化连接

```go
import "github.com/leeforge/framework/redis_client"

var redisCfg redis_client.Config
cfg.Bind(&redisCfg)  // 从 config 模块绑定

client, err := redis_client.Setup(redisCfg)
if err != nil {
    log.Fatalf("redis connect failed: %v", err)
}
defer client.Close()

// 使用 go-redis 客户端
err = client.Set(ctx, "key", "value", time.Hour).Err()
val, err := client.Get(ctx, "key").Result()
```

### 在插件中使用

```go
func (p *MyPlugin) Setup(ctx context.Context, app plugin.AppContext) error {
    // 通过 Cache 接口使用 Redis（推荐）
    cache := app.Cache()

    // 或者直接获取 Redis 客户端（高级用法）
    rdb := app.Redis()

    return nil
}
```

## 连接地址

`Config.Addr()` 返回 `host:port` 格式的连接地址：

```go
cfg := redis_client.Config{Host: "redis", Port: "6379"}
fmt.Println(cfg.Addr()) // "redis:6379"
```

## 生产环境建议

```yaml
redis:
  host: ${REDIS_HOST}        # 通过环境变量注入
  port: ${REDIS_PORT}
  password: ${REDIS_PASSWORD}
  db: 0
```

## 注意事项

- 密码应通过环境变量注入，**禁止**硬编码在配置文件中并提交
- 对于需要高可用的场景，考虑使用 Redis Sentinel 或 Cluster 模式（需扩展此包）
