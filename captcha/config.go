package captcha

import "time"

// Config 验证码配置
type Config struct {
	Enabled bool          `json:"enabled" yaml:"enabled"` // 是否启用
	TTL     time.Duration `json:"ttl" yaml:"ttl"`         // 过期时间

	// 生成限制
	GenerateLimit  int           `json:"generateLimit" yaml:"generateLimit"`   // 时间窗口内的生成次数限制
	GenerateWindow time.Duration `json:"generateWindow" yaml:"generateWindow"` // 生成限流时间窗口

	// 验证限制
	MaxAttempts   int           `json:"maxAttempts" yaml:"maxAttempts"`     // 最大尝试次数
	AttemptWindow time.Duration `json:"attemptWindow" yaml:"attemptWindow"` // 尝试限流时间窗口

	// 类型特定配置
	Math  MathConfig  `json:"math" yaml:"math"`
	Image ImageConfig `json:"image" yaml:"image"`
}

// MathConfig 数学验证码配置
type MathConfig struct {
	Width           int `json:"width" yaml:"width"`                     // 宽度
	Height          int `json:"height" yaml:"height"`                   // 高度
	NoiseCount      int `json:"noiseCount" yaml:"noiseCount"`           // 噪点数量
	ShowLineOptions int `json:"showLineOptions" yaml:"showLineOptions"` // 线条选项
}

// ImageConfig 图片验证码配置
type ImageConfig struct {
	Width      int `json:"width" yaml:"width"`           // 宽度
	Height     int `json:"height" yaml:"height"`         // 高度
	Length     int `json:"length" yaml:"length"`         // 字符长度
	NoiseCount int `json:"noiseCount" yaml:"noiseCount"` // 噪点数量
}
