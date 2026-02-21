package captcha

import (
	"context"
	"time"
)

// Generator 创建验证码挑战
type Generator interface {
	// Generate 生成验证码
	// 返回: CaptchaData (返回给客户端), answer (存储用), error
	Generate(ctx context.Context, captchaType CaptchaType) (*CaptchaData, string, error)
}

// Store 管理验证码持久化
type Store interface {
	// Save 保存验证码答案
	Save(ctx context.Context, id string, answer string, ttl time.Duration) error

	// Get 获取验证码答案
	Get(ctx context.Context, id string) (answer string, err error)

	// Delete 删除验证码
	Delete(ctx context.Context, id string) error

	// Exists 检查验证码是否存在
	Exists(ctx context.Context, id string) (bool, error)
}

// RateLimiter 防止滥用
type RateLimiter interface {
	// AllowGenerate 检查用户是否可以生成新验证码
	AllowGenerate(ctx context.Context, identifier string) error

	// AllowVerify 检查用户是否可以验证（处理最大尝试次数）
	AllowVerify(ctx context.Context, identifier string) error

	// RecordFailure 记录验证失败
	RecordFailure(ctx context.Context, identifier string) error

	// Reset 清除标识符的限流记录（成功验证后）
	Reset(ctx context.Context, identifier string) error
}

// Service 编排验证码操作
type Service interface {
	// Generate 生成验证码
	Generate(ctx context.Context, captchaType CaptchaType, identifier string) (*CaptchaData, error)

	// Verify 验证验证码
	Verify(ctx context.Context, id string, answer string, identifier string) (*VerifyResult, error)
}
