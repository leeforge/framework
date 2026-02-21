package logging

import (
	"sync"
)

// Factory creates and manages named loggers.
type Factory struct {
	config  Config
	loggers sync.Map // map[string]Logger
}

// NewFactory creates a new Factory with the given config.
func NewFactory(config Config) *Factory {
	config.applyDefaults()
	return &Factory{
		config: config,
	}
}

// GetLogger returns a named logger, creating it if necessary.
// Named loggers share the same configuration but have different names
// for identification in log output.
func (f *Factory) GetLogger(name string) Logger {
	// Fast path: check if logger exists
	if v, ok := f.loggers.Load(name); ok {
		return v.(Logger)
	}

	// Slow path: create new logger
	logger := NewLogger(f.config).Named(name)
	actual, loaded := f.loggers.LoadOrStore(name, logger)
	if loaded {
		return actual.(Logger)
	}
	return logger
}

// Config returns a copy of the factory's configuration.
func (f *Factory) Config() Config {
	return f.config
}
