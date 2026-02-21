package logging

import (
	"sync"

	"go.uber.org/zap"
)

var (
	globalLogger Logger
	globalMu     sync.RWMutex
	once         sync.Once
)

// initGlobal initializes the global logger with default config.
func initGlobal() {
	once.Do(func() {
		globalLogger = NewLogger(DefaultConfig())
	})
}

// Global returns the global logger instance.
func Global() Logger {
	globalMu.RLock()
	if globalLogger != nil {
		defer globalMu.RUnlock()
		return globalLogger
	}
	globalMu.RUnlock()

	initGlobal()

	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalLogger
}

// SetGlobal replaces the global logger with the given logger.
func SetGlobal(logger Logger) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLogger = logger
}

// Init initializes the global logger with the given config.
func Init(config Config) {
	SetGlobal(NewLogger(config))
}

// Package-level convenience functions that delegate to the global logger.

// Debug logs a message at DebugLevel using the global logger.
func Debug(msg string, fields ...zap.Field) {
	Global().Debug(msg, fields...)
}

// Info logs a message at InfoLevel using the global logger.
func Info(msg string, fields ...zap.Field) {
	Global().Info(msg, fields...)
}

// Warn logs a message at WarnLevel using the global logger.
func Warn(msg string, fields ...zap.Field) {
	Global().Warn(msg, fields...)
}

// Error logs a message at ErrorLevel using the global logger.
func Error(msg string, fields ...zap.Field) {
	Global().Error(msg, fields...)
}

// Fatal logs a message at FatalLevel using the global logger and exits.
func Fatal(msg string, fields ...zap.Field) {
	Global().Fatal(msg, fields...)
}

// Debugf logs a formatted message at DebugLevel using the global logger.
func Debugf(format string, args ...any) {
	Global().Debugf(format, args...)
}

// Infof logs a formatted message at InfoLevel using the global logger.
func Infof(format string, args ...any) {
	Global().Infof(format, args...)
}

// Warnf logs a formatted message at WarnLevel using the global logger.
func Warnf(format string, args ...any) {
	Global().Warnf(format, args...)
}

// Errorf logs a formatted message at ErrorLevel using the global logger.
func Errorf(format string, args ...any) {
	Global().Errorf(format, args...)
}

// Fatalf logs a formatted message at FatalLevel using the global logger and exits.
func Fatalf(format string, args ...any) {
	Global().Fatalf(format, args...)
}

// With creates a child logger from the global logger with additional fields.
func With(fields ...zap.Field) Logger {
	return Global().With(fields...)
}

// WithError creates a child logger from the global logger with an error field.
func WithError(err error) Logger {
	return Global().WithError(err)
}

// Named creates a child logger from the global logger with the given name.
func Named(name string) Logger {
	return Global().Named(name)
}

// Sync flushes any buffered log entries from the global logger.
func Sync() error {
	return Global().Sync()
}
