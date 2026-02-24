# captcha — 验证码

提供验证码生成、存储与校验的抽象接口，支持数学运算、图片与滑块三种验证码类型，内置防刷限制。

## 接口概览

```go
// Generator 生成验证码挑战
type Generator interface {
    Generate(ctx context.Context, captchaType CaptchaType) (*CaptchaData, string, error)
}

// Store 管理验证码持久化
type Store interface {
    Save(ctx context.Context, id string, answer string, ttl time.Duration) error
    Get(ctx context.Context, id string) (string, error)
    Delete(ctx context.Context, id string) error
    Exists(ctx context.Context, id string) (bool, error)
}

// Verifier 验证用户答案
type Verifier interface {
    Verify(ctx context.Context, id string, answer string) (bool, error)
}
```

## 验证码类型

| 类型 | 常量 | 说明 |
|---|---|---|
| 数学运算 | `TypeMath` | 如 "3 + 5 = ?" |
| 图片识别 | `TypeImage` | 图片中的字符 |
| 滑块 | `TypeSlider` | 拖动到指定位置 |

## 快速开始

```go
import "github.com/leeforge/framework/captcha"

// 配置
cfg := captcha.Config{
    Enabled: true,
    TTL:     5 * time.Minute,
    // 生成限制：同一 IP 1 分钟内最多生成 10 次
    GenerateLimit:  10,
    GenerateWindow: time.Minute,
    // 验证限制：同一验证码最多尝试 3 次
    MaxAttempts:   3,
    AttemptWindow: 5 * time.Minute,
    Math: captcha.MathConfig{
        Width: 120, Height: 40, NoiseCount: 5,
    },
}

// 生成验证码
captchaData, answer, err := generator.Generate(ctx, captcha.TypeMath)
// captchaData 返回给前端（含 ID + 挑战数据）
// answer 存入 Store

// 保存答案
store.Save(ctx, captchaData.ID, answer, cfg.TTL)

// 验证
ok, err := verifier.Verify(ctx, captchaID, userAnswer)
```

## 典型流程

```
1. 前端请求验证码 → Handler 调用 Generate → 返回 {id, imageBase64}
2. 用户填写 → 提交表单携带 {captchaID, captchaAnswer}
3. Handler 调用 Verify → 通过 → 继续业务逻辑
                       → 失败 → 返回 422 验证码错误
```

## 注意事项

- `Store` 的实际实现需对接 Redis，推荐使用 `cache` 模块的 Redis 适配器
- 验证成功后应立即删除验证码（`Store.Delete`），防止重放攻击
- 启用 `RateLimiter` 防止爬虫批量获取验证码
