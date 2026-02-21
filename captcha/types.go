package captcha

import (
	"errors"
	"time"
)

// CaptchaType 定义验证码类型
type CaptchaType string

const (
	TypeMath   CaptchaType = "math"   // 数学运算
	TypeImage  CaptchaType = "image"  // 图片验证码
	TypeSlider CaptchaType = "slider" // 滑块验证码
)

// CaptchaData 表示生成的验证码数据
type CaptchaData struct {
	ID        string      `json:"id"`
	Type      CaptchaType `json:"type"`
	Content   string      `json:"content"`   // Base64 编码的图片或问题
	ExpiresAt time.Time   `json:"expiresAt"`
}

// VerifyResult 包含验证结果
type VerifyResult struct {
	Valid         bool   `json:"valid"`                   // 是否验证通过
	FailureReason string `json:"failureReason,omitempty"` // 失败原因
	AttemptsLeft  int    `json:"attemptsLeft,omitempty"`  // 剩余尝试次数
}

// 错误定义
var (
	ErrCaptchaNotFound   = errors.New("captcha not found")         // 验证码不存在
	ErrCaptchaExpired    = errors.New("captcha expired")           // 验证码已过期
	ErrInvalidAnswer     = errors.New("invalid answer")            // 答案错误
	ErrRateLimitExceeded = errors.New("rate limit exceeded")       // 超过限流
	ErrGenerationFailed  = errors.New("captcha generation failed") // 生成失败
)
