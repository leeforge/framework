package component

import "context"

// Component 是所有组件必须实现的接口
type Component interface {
	// Name 返回组件的唯一标识符
	Name() string

	// Validate 验证字段值是否符合组件规则
	Validate(ctx context.Context, config map[string]any, value any) error

	// GetOptions 获取字段的可选值列表（用于下拉框、单选框等）
	GetOptions(ctx context.Context, config map[string]any) ([]Option, error)

	// PopulateDisplay 批量填充字段的显示信息（用于列表展示）
	PopulateDisplay(ctx context.Context, config map[string]any, values []any) (map[any]Display, error)
}

// Option 表示组件的可选项
type Option struct {
	Label string `json:"label"`           // 显示文本
	Value any    `json:"value"`           // 实际值
	Extra any    `json:"extra,omitempty"` // 额外信息（如颜色、图标）
}

// Display 表示组件的显示信息
type Display struct {
	Label string `json:"label"`           // 显示文本
	Value any    `json:"value"`           // 原始值
	Extra any    `json:"extra,omitempty"` // 扩展信息
}

// DisplayConfig Display 配置
type DisplayConfig struct {
	Mode   string   `json:"mode"`             // auto | always | never | on-demand
	Fields []string `json:"fields,omitempty"` // 需要填充的字段
	Cache  bool     `json:"cache,omitempty"`  // 是否缓存
}
