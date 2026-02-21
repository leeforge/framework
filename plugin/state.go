package plugin

// PluginState represents the lifecycle state of a plugin.
type PluginState int

const (
	StateRegistered PluginState = iota // Registered, not yet processed
	StateInstalled                     // Install() succeeded
	StateEnabled                       // Enable() succeeded, running
	StateDisabled                      // Disable() succeeded, stopped
	StateFailed                        // Install() or Enable() failed
)

// String returns a human-readable state name.
func (s PluginState) String() string {
	switch s {
	case StateRegistered:
		return "registered"
	case StateInstalled:
		return "installed"
	case StateEnabled:
		return "enabled"
	case StateDisabled:
		return "disabled"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// IsTerminal returns true if the state cannot transition further in normal flow.
func (s PluginState) IsTerminal() bool {
	return s == StateFailed || s == StateDisabled
}
