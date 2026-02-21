package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is the interface for structured logging.
type Logger interface {
	// Debug logs a message at DebugLevel.
	Debug(msg string, fields ...zap.Field)
	// Info logs a message at InfoLevel.
	Info(msg string, fields ...zap.Field)
	// Warn logs a message at WarnLevel.
	Warn(msg string, fields ...zap.Field)
	// Error logs a message at ErrorLevel.
	Error(msg string, fields ...zap.Field)
	// Fatal logs a message at FatalLevel and then calls os.Exit(1).
	Fatal(msg string, fields ...zap.Field)

	// Debugf logs a formatted message at DebugLevel.
	Debugf(format string, args ...any)
	// Infof logs a formatted message at InfoLevel.
	Infof(format string, args ...any)
	// Warnf logs a formatted message at WarnLevel.
	Warnf(format string, args ...any)
	// Errorf logs a formatted message at ErrorLevel.
	Errorf(format string, args ...any)
	// Fatalf logs a formatted message at FatalLevel and then calls os.Exit(1).
	Fatalf(format string, args ...any)

	// With creates a child logger with additional fields.
	With(fields ...zap.Field) Logger
	// WithError creates a child logger with an error field.
	WithError(err error) Logger
	// Named creates a child logger with the given name.
	Named(name string) Logger

	// Zap returns the underlying *zap.Logger.
	Zap() *zap.Logger
	// Sugar returns the underlying *zap.SugaredLogger.
	Sugar() *zap.SugaredLogger
	// Sync flushes any buffered log entries.
	Sync() error
}

// zapLogger wraps *zap.Logger to implement the Logger interface.
type zapLogger struct {
	zl *zap.Logger
	sl *zap.SugaredLogger
}

// NewLogger creates a new Logger from the given Config.
func NewLogger(config Config) Logger {
	config.applyDefaults()

	cores := getZapCores(config)
	zapLog := zap.New(zapcore.NewTee(cores...))

	if config.ShowLineNumber {
		zapLog = zapLog.WithOptions(zap.AddCaller(), zap.AddCallerSkip(1))
	}

	return &zapLogger{
		zl: zapLog,
		sl: zapLog.Sugar(),
	}
}

// newZapLogger wraps an existing *zap.Logger as a Logger.
func newZapLogger(zl *zap.Logger) Logger {
	return &zapLogger{
		zl: zl,
		sl: zl.Sugar(),
	}
}

// FromZap wraps an existing *zap.Logger as a Logger.
func FromZap(zl *zap.Logger) Logger {
	return newZapLogger(zl)
}

func (l *zapLogger) Debug(msg string, fields ...zap.Field) {
	l.zl.Debug(msg, fields...)
}

func (l *zapLogger) Info(msg string, fields ...zap.Field) {
	l.zl.Info(msg, fields...)
}

func (l *zapLogger) Warn(msg string, fields ...zap.Field) {
	l.zl.Warn(msg, fields...)
}

func (l *zapLogger) Error(msg string, fields ...zap.Field) {
	l.zl.Error(msg, fields...)
}

func (l *zapLogger) Fatal(msg string, fields ...zap.Field) {
	l.zl.Fatal(msg, fields...)
}

func (l *zapLogger) Debugf(format string, args ...any) {
	l.sl.Debugf(format, args...)
}

func (l *zapLogger) Infof(format string, args ...any) {
	l.sl.Infof(format, args...)
}

func (l *zapLogger) Warnf(format string, args ...any) {
	l.sl.Warnf(format, args...)
}

func (l *zapLogger) Errorf(format string, args ...any) {
	l.sl.Errorf(format, args...)
}

func (l *zapLogger) Fatalf(format string, args ...any) {
	l.sl.Fatalf(format, args...)
}

func (l *zapLogger) With(fields ...zap.Field) Logger {
	return newZapLogger(l.zl.With(fields...))
}

func (l *zapLogger) WithError(err error) Logger {
	return newZapLogger(l.zl.With(zap.Error(err)))
}

func (l *zapLogger) Named(name string) Logger {
	return newZapLogger(l.zl.Named(name))
}

func (l *zapLogger) Zap() *zap.Logger {
	return l.zl
}

func (l *zapLogger) Sugar() *zap.SugaredLogger {
	return l.sl
}

func (l *zapLogger) Sync() error {
	return l.zl.Sync()
}

// Ensure zapLogger implements Logger.
var _ Logger = (*zapLogger)(nil)
