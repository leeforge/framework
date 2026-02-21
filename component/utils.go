package component

import (
	"encoding/json"
	"fmt"
)

// ParseConfig 解析组件配置
func ParseConfig(config map[string]any, target any) error {
	data, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return nil
}

// ExtractDisplayConfig 提取 Display 配置
func ExtractDisplayConfig(config map[string]any) *DisplayConfig {
	displayRaw, ok := config["display"]
	if !ok {
		return &DisplayConfig{
			Mode: "auto", // 默认模式
		}
	}

	displayCfg := &DisplayConfig{}
	if err := ParseConfig(displayRaw.(map[string]any), displayCfg); err != nil {
		return &DisplayConfig{Mode: "auto"}
	}

	// 设置默认值
	if displayCfg.Mode == "" {
		displayCfg.Mode = "auto"
	}

	return displayCfg
}

// ShouldPopulateDisplay 判断是否应该填充显示信息
func ShouldPopulateDisplay(config map[string]any) bool {
	displayCfg := ExtractDisplayConfig(config)

	switch displayCfg.Mode {
	case "never":
		return false
	case "always":
		return true
	case "auto":
		return true // 默认自动填充
	case "on-demand":
		// TODO: 从 context 中获取请求参数判断
		return false
	default:
		return true
	}
}

// FilterFields 根据 display.fields 过滤字段
func FilterFields(extra any, fields []string) any {
	if len(fields) == 0 {
		return extra
	}

	extraMap, ok := extra.(map[string]any)
	if !ok {
		return extra
	}

	filtered := make(map[string]any)
	for _, field := range fields {
		if val, ok := extraMap[field]; ok {
			filtered[field] = val
		}
	}

	return filtered
}

// ParseExtendJSON 解析扩展字段（JSON 字符串）
func ParseExtendJSON(extend string) map[string]any {
	if extend == "" {
		return nil
	}

	var extra map[string]any
	if err := json.Unmarshal([]byte(extend), &extra); err != nil {
		return nil
	}

	return extra
}
