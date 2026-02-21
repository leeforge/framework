package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Hook is a function that is called for each log entry.
// It can be used for custom log processing, alerting, metrics, etc.
type Hook func(entry zapcore.Entry) error

// hookCore wraps a zapcore.Core and calls hooks on each log entry.
type hookCore struct {
	zapcore.Core
	hooks []Hook
}

// newHookCore creates a new hookCore wrapping the given core.
func newHookCore(core zapcore.Core, hooks []Hook) zapcore.Core {
	return &hookCore{
		Core:  core,
		hooks: hooks,
	}
}

// Check implements zapcore.Core.
func (c *hookCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}
	return ce
}

// Write implements zapcore.Core.
func (c *hookCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// Call hooks first
	for _, hook := range c.hooks {
		if err := hook(entry); err != nil {
			// Log hook errors but don't stop the write
			// This prevents hook errors from breaking logging
		}
	}
	return c.Core.Write(entry, fields)
}

// With implements zapcore.Core.
func (c *hookCore) With(fields []zapcore.Field) zapcore.Core {
	return &hookCore{
		Core:  c.Core.With(fields),
		hooks: c.hooks,
	}
}

// WithHook creates a new Logger with the given hook attached.
func WithHook(logger Logger, hook Hook) Logger {
	zl := logger.Zap()
	core := zl.Core()
	hookedCore := newHookCore(core, []Hook{hook})
	return newZapLogger(zap.New(hookedCore))
}

// WithHooks creates a new Logger with multiple hooks attached.
func WithHooks(logger Logger, hooks ...Hook) Logger {
	if len(hooks) == 0 {
		return logger
	}

	zl := logger.Zap()
	core := zl.Core()
	hookedCore := newHookCore(core, hooks)
	return newZapLogger(zap.New(hookedCore))
}
