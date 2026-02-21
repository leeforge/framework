package plugin

import "testing"

func TestMapConfigProvider_GetString(t *testing.T) {
	cfg := NewMapConfigProvider(map[string]any{
		"host": "localhost",
		"port": 8080,
	})

	if got := cfg.GetString("host", ""); got != "localhost" {
		t.Errorf("GetString(host) = %q, want %q", got, "localhost")
	}
	if got := cfg.GetString("missing", "default"); got != "default" {
		t.Errorf("GetString(missing) = %q, want %q", got, "default")
	}
}

func TestMapConfigProvider_GetInt(t *testing.T) {
	cfg := NewMapConfigProvider(map[string]any{
		"port": 8080,
	})

	if got := cfg.GetInt("port", 0); got != 8080 {
		t.Errorf("GetInt(port) = %d, want %d", got, 8080)
	}
	if got := cfg.GetInt("missing", 3000); got != 3000 {
		t.Errorf("GetInt(missing) = %d, want %d", got, 3000)
	}
}

func TestMapConfigProvider_GetBool(t *testing.T) {
	cfg := NewMapConfigProvider(map[string]any{
		"debug": true,
	})

	if got := cfg.GetBool("debug", false); got != true {
		t.Errorf("GetBool(debug) = %v, want true", got)
	}
	if got := cfg.GetBool("missing", false); got != false {
		t.Errorf("GetBool(missing) = %v, want false", got)
	}
}

func TestMapConfigProvider_Get(t *testing.T) {
	cfg := NewMapConfigProvider(map[string]any{
		"key": "value",
	})

	val, ok := cfg.Get("key")
	if !ok || val != "value" {
		t.Errorf("Get(key) = (%v, %v), want (value, true)", val, ok)
	}

	_, ok = cfg.Get("nope")
	if ok {
		t.Error("Get(nope) should return false")
	}
}

func TestMapConfigProvider_Bind(t *testing.T) {
	cfg := NewMapConfigProvider(map[string]any{
		"host": "localhost",
		"port": float64(8080), // JSON numbers decode as float64
	})

	type Config struct {
		Host string  `json:"host"`
		Port float64 `json:"port"`
	}

	var target Config
	if err := cfg.Bind(&target); err != nil {
		t.Fatalf("Bind failed: %v", err)
	}
	if target.Host != "localhost" {
		t.Errorf("Bind Host = %q, want %q", target.Host, "localhost")
	}
	if target.Port != 8080 {
		t.Errorf("Bind Port = %v, want %v", target.Port, 8080)
	}
}

func TestMapConfigProvider_IsEnabled(t *testing.T) {
	enabled := NewPluginConfigEntry("test", true, map[string]any{})
	if !enabled.IsEnabled() {
		t.Error("should be enabled")
	}

	disabled := NewPluginConfigEntry("test", false, map[string]any{})
	if disabled.IsEnabled() {
		t.Error("should be disabled")
	}
}

func TestEmptyConfigProvider(t *testing.T) {
	cfg := EmptyConfig()
	if got := cfg.GetString("any", "fallback"); got != "fallback" {
		t.Errorf("empty config should return default, got %q", got)
	}
	if cfg.IsEnabled() {
		t.Error("empty config should not be enabled")
	}
}
