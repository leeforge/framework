package plugin

import "encoding/json"

// ConfigProvider gives plugins type-safe access to their scoped configuration.
type ConfigProvider interface {
	Get(key string) (any, bool)
	GetString(key string, defaultVal string) string
	GetInt(key string, defaultVal int) int
	GetBool(key string, defaultVal bool) bool
	Bind(target any) error
	IsEnabled() bool
}

// PluginConfigEntry represents a single plugin's configuration entry.
type PluginConfigEntry struct {
	name     string
	enabled  bool
	settings map[string]any
}

// NewPluginConfigEntry creates a plugin config entry.
func NewPluginConfigEntry(name string, enabled bool, settings map[string]any) *PluginConfigEntry {
	if settings == nil {
		settings = make(map[string]any)
	}
	return &PluginConfigEntry{name: name, enabled: enabled, settings: settings}
}

func (c *PluginConfigEntry) Get(key string) (any, bool) {
	v, ok := c.settings[key]
	return v, ok
}

func (c *PluginConfigEntry) GetString(key string, defaultVal string) string {
	v, ok := c.settings[key]
	if !ok {
		return defaultVal
	}
	s, ok := v.(string)
	if !ok {
		return defaultVal
	}
	return s
}

func (c *PluginConfigEntry) GetInt(key string, defaultVal int) int {
	v, ok := c.settings[key]
	if !ok {
		return defaultVal
	}
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	case int64:
		return int(n)
	default:
		return defaultVal
	}
}

func (c *PluginConfigEntry) GetBool(key string, defaultVal bool) bool {
	v, ok := c.settings[key]
	if !ok {
		return defaultVal
	}
	b, ok := v.(bool)
	if !ok {
		return defaultVal
	}
	return b
}

func (c *PluginConfigEntry) Bind(target any) error {
	data, err := json.Marshal(c.settings)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func (c *PluginConfigEntry) IsEnabled() bool {
	return c.enabled
}

// MapConfigProvider is a simple ConfigProvider backed by a map.
// Used for testing and inline configuration.
type MapConfigProvider = PluginConfigEntry

// NewMapConfigProvider creates a ConfigProvider from a settings map (always enabled).
func NewMapConfigProvider(settings map[string]any) *PluginConfigEntry {
	return NewPluginConfigEntry("", true, settings)
}

// emptyConfig is a ConfigProvider that returns defaults for everything.
type emptyConfig struct{}

func (e *emptyConfig) Get(string) (any, bool)              { return nil, false }
func (e *emptyConfig) GetString(_ string, d string) string  { return d }
func (e *emptyConfig) GetInt(_ string, d int) int           { return d }
func (e *emptyConfig) GetBool(_ string, d bool) bool        { return d }
func (e *emptyConfig) Bind(any) error                       { return nil }
func (e *emptyConfig) IsEnabled() bool                      { return false }

// EmptyConfig returns a ConfigProvider that always returns defaults.
func EmptyConfig() ConfigProvider { return &emptyConfig{} }
