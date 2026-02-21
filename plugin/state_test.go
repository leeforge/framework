package plugin

import "testing"

func TestPluginState_String(t *testing.T) {
	tests := []struct {
		state PluginState
		want  string
	}{
		{StateRegistered, "registered"},
		{StateInstalled, "installed"},
		{StateEnabled, "enabled"},
		{StateDisabled, "disabled"},
		{StateFailed, "failed"},
		{PluginState(99), "unknown"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("PluginState(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}

func TestPluginState_IsTerminal(t *testing.T) {
	if StateEnabled.IsTerminal() {
		t.Error("Enabled should not be terminal")
	}
	if !StateFailed.IsTerminal() {
		t.Error("Failed should be terminal")
	}
	if !StateDisabled.IsTerminal() {
		t.Error("Disabled should be terminal")
	}
}
