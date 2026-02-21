package binding

import (
	"errors"
	"io"
	"net/http"

	"github.com/leeforge/framework/json"
)

type jsonBinding struct{}

// DecodeOptions JSON 解码选项配置
type DecodeOptions struct {
	// UseNumber 将 JSON 数字解析为 Number 类型而不是 float64
	useNumber bool
	// DisallowUnknownFields 不允许 JSON 中包含未知字段
	disallowUnknownFields bool
}

// Option 解码选项函数类型
type Option func(*DecodeOptions)

// WithUseNumber 使用 json.Number 来解析数字，而不是 float64
// 这样可以保持数字的精度，避免大整数丢失精度
func WithUseNumber() Option {
	return func(opts *DecodeOptions) {
		opts.useNumber = true
	}
}

// WithDisallowUnknownFields 不允许 JSON 中包含结构体未定义的字段
// 启用此选项可以提高安全性，避免接收不期望的数据
func WithDisallowUnknownFields() Option {
	return func(opts *DecodeOptions) {
		opts.disallowUnknownFields = true
	}
}

// newDefaultDecodeOptions 创建默认解码选项
func newDefaultDecodeOptions() *DecodeOptions {
	return &DecodeOptions{
		useNumber:             false,
		disallowUnknownFields: false,
	}
}

// applyDecodeOptions 应用选项
func applyDecodeOptions(opts ...Option) *DecodeOptions {
	options := newDefaultDecodeOptions()
	for _, opt := range opts {
		opt(options)
	}
	return options
}

// decodeJson 解码 JSON 数据到目标对象
// 支持通过 Option 自定义解码行为
func decodeJson(r io.Reader, v any, opts ...Option) error {
	options := applyDecodeOptions(opts...)

	decoder := json.NewDecoder(r)

	// 应用 UseNumber 选项
	if options.useNumber {
		decoder.Decoder.UseNumber()
	}

	// 应用 DisallowUnknownFields 选项
	if options.disallowUnknownFields {
		decoder.Decoder.DisallowUnknownFields()
	}

	return decoder.Decode(v)
}

func (jsonBinding) Name() string {
	return "json"
}

// Bind 绑定 HTTP 请求体到目标对象
// 支持通过 Option 自定义解码行为
//
// 示例:
//
//	err := binding.Bind(r, &user)
//
//	// 使用选项
//	err := binding.Bind(r, &user, binding.WithUseNumber())
func (jsonBinding) Bind(r *http.Request, v any, opts ...Option) error {
	if r == nil || r.Body == nil {
		return errors.New(InvalidRequestBodyError)
	}
	return decodeJson(r.Body, v, opts...)
}
