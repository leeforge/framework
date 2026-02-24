# component — CMS 字段组件注册

提供 CMS 内容字段组件的注册与管理机制，每种字段类型（文本、富文本、关系、媒体等）对应一个 `Component` 实现。

## 核心接口

```go
type Component interface {
    // Name 组件唯一标识（如 "text", "richtext", "relation"）
    Name() string

    // Validate 校验字段值是否符合组件规则
    Validate(ctx context.Context, config map[string]any, value any) error

    // GetOptions 获取下拉/单选组件的可选项列表
    GetOptions(ctx context.Context, config map[string]any) ([]Option, error)

    // PopulateDisplay 批量填充列表展示信息（用于关联字段）
    PopulateDisplay(ctx context.Context, config map[string]any, values []any) (map[any]Display, error)
}
```

## 注册组件

```go
import "github.com/leeforge/framework/component"

// 注册自定义组件
err := component.Register(&MySelectComponent{})

// 重复注册同名组件会返回错误
```

## 获取组件

```go
// 根据名称获取已注册的组件
comp, err := component.Get("select")
if err != nil {
    // 组件未注册
}

// 获取所有已注册组件
all := component.All()
```

## 实现自定义组件

```go
type SelectComponent struct{}

func (c *SelectComponent) Name() string { return "select" }

func (c *SelectComponent) Validate(ctx context.Context, config map[string]any, value any) error {
    var cfg SelectConfig
    component.ParseConfig(config, &cfg)

    strVal, ok := value.(string)
    if !ok {
        return errors.New("select 字段值必须为字符串")
    }

    for _, opt := range cfg.Options {
        if opt.Value == strVal {
            return nil
        }
    }
    return fmt.Errorf("值 %q 不在可选项中", strVal)
}

func (c *SelectComponent) GetOptions(ctx context.Context, config map[string]any) ([]component.Option, error) {
    var cfg SelectConfig
    component.ParseConfig(config, &cfg)
    return cfg.Options, nil
}

func (c *SelectComponent) PopulateDisplay(ctx context.Context, config map[string]any, values []any) (map[any]component.Display, error) {
    options, _ := c.GetOptions(ctx, config)
    result := make(map[any]component.Display)
    for _, opt := range options {
        result[opt.Value] = component.Display{Label: opt.Label, Value: opt.Value}
    }
    return result, nil
}
```

## 工具函数

```go
// ParseConfig 将 map[string]any 反序列化为结构体
type SelectConfig struct {
    Options []component.Option `json:"options"`
    Multiple bool              `json:"multiple"`
}
var cfg SelectConfig
component.ParseConfig(configMap, &cfg)
```

## 注意事项

- 组件注册应在应用启动时（`plugin.Setup`）调用，避免并发注册冲突
- `PopulateDisplay` 用于列表页批量填充，应尽量使用批量查询而非循环单查
