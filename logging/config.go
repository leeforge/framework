package logging

import (
	"strings"

	"go.uber.org/zap/zapcore"
)

// Config represents the logger configuration.
type Config struct {
	// Director is the directory where log files will be stored.
	Director string `mapstructure:"director" json:"director" yaml:"director" toml:"director"`

	// MessageKey is the JSON key for the message field.
	MessageKey string `mapstructure:"message-key" json:"messageKey" yaml:"message-key" toml:"message-key"`

	// LevelKey is the JSON key for the level field.
	LevelKey string `mapstructure:"level-key" json:"levelKey" yaml:"level-key" toml:"level-key"`

	// TimeKey is the JSON key for the timestamp field.
	TimeKey string `mapstructure:"time-key" json:"timeKey" yaml:"time-key" toml:"time-key"`

	// NameKey is the JSON key for the logger name field.
	NameKey string `mapstructure:"name-key" json:"nameKey" yaml:"name-key" toml:"name-key"`

	// CallerKey is the JSON key for the caller field.
	CallerKey string `mapstructure:"caller-key" json:"callerKey" yaml:"caller-key" toml:"caller-key"`

	// LineEnding is the line ending character(s).
	LineEnding string `mapstructure:"line-ending" json:"lineEnding" yaml:"line-ending" toml:"line-ending"`

	// StacktraceKey is the JSON key for the stacktrace field.
	StacktraceKey string `mapstructure:"stacktrace-key" json:"stacktraceKey" yaml:"stacktrace-key" toml:"stacktrace-key"`

	// Level is the minimum log level (debug, info, warn, error, dpanic, panic, fatal).
	Level string `mapstructure:"level" json:"level" yaml:"level" toml:"level"`

	// EncodeLevel is the level encoder type (LowercaseLevelEncoder, LowercaseColorLevelEncoder, CapitalLevelEncoder, CapitalColorLevelEncoder).
	EncodeLevel string `mapstructure:"encode-level" json:"encodeLevel" yaml:"encode-level" toml:"encode-level"`

	// Prefix is the prefix to prepend to each log line.
	Prefix string `mapstructure:"prefix" json:"prefix" yaml:"prefix" toml:"prefix"`

	// TimeFormat is the time format string (uses Go time format).
	TimeFormat string `mapstructure:"time-format" json:"timeFormat" yaml:"time-format" toml:"time-format"`

	// Format is the log format (json or console).
	Format string `mapstructure:"format" json:"format" yaml:"format" toml:"format"`

	// LogInTerminal enables logging to terminal in addition to file.
	LogInTerminal bool `mapstructure:"log-in-terminal" json:"logInTerminal" yaml:"log-in-terminal" toml:"log-in-terminal"`

	// MaxAge is the maximum number of days to retain old log files.
	MaxAge int `mapstructure:"max-age" json:"maxAge" yaml:"max-age" toml:"max-age"`

	// MaxSize is the maximum size in megabytes of the log file before it gets rotated.
	MaxSize int `mapstructure:"max-size" json:"maxSize" yaml:"max-size" toml:"max-size"`

	// MaxBackups is the maximum number of old log files to retain.
	MaxBackups int `mapstructure:"max-backups" json:"maxBackups" yaml:"max-backups" toml:"max-backups"`

	// Compress determines if the rotated log files should be compressed using gzip.
	Compress bool `mapstructure:"compress" json:"compress" yaml:"compress" toml:"compress"`

	// ShowLineNumber enables adding caller information to log entries.
	ShowLineNumber bool `mapstructure:"show-line-number" json:"showLineNumber" yaml:"show-line-number" toml:"show-line-number"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Director:       "logs",
		MessageKey:     "message",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "caller",
		LineEnding:     zapcore.DefaultLineEnding,
		StacktraceKey:  "stacktrace",
		Level:          "info",
		EncodeLevel:    "LowercaseLevelEncoder",
		Prefix:         "",
		TimeFormat:     "2006/01/02 - 15:04:05",
		Format:         "json",
		LogInTerminal:  true,
		MaxAge:         7,
		MaxSize:        100,
		MaxBackups:     10,
		Compress:       true,
		ShowLineNumber: true,
	}
}

// TransportLevel converts the string level to zapcore.Level.
func (c Config) TransportLevel() zapcore.Level {
	level := strings.ToLower(c.Level)
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.DebugLevel
	}
}

// ZapEncodeLevel returns the zapcore.LevelEncoder based on EncodeLevel.
func (c Config) ZapEncodeLevel() zapcore.LevelEncoder {
	switch c.EncodeLevel {
	case "LowercaseLevelEncoder":
		return zapcore.LowercaseLevelEncoder
	case "LowercaseColorLevelEncoder":
		return zapcore.LowercaseColorLevelEncoder
	case "CapitalLevelEncoder":
		return zapcore.CapitalLevelEncoder
	case "CapitalColorLevelEncoder":
		return zapcore.CapitalColorLevelEncoder
	default:
		return zapcore.LowercaseLevelEncoder
	}
}

// applyDefaults applies default values to empty fields.
func (c *Config) applyDefaults() {
	defaults := DefaultConfig()

	if c.MessageKey == "" {
		c.MessageKey = defaults.MessageKey
	}
	if c.LevelKey == "" {
		c.LevelKey = defaults.LevelKey
	}
	if c.TimeKey == "" {
		c.TimeKey = defaults.TimeKey
	}
	if c.NameKey == "" {
		c.NameKey = defaults.NameKey
	}
	if c.CallerKey == "" {
		c.CallerKey = defaults.CallerKey
	}
	if c.LineEnding == "" {
		c.LineEnding = defaults.LineEnding
	}
	if c.StacktraceKey == "" {
		c.StacktraceKey = defaults.StacktraceKey
	}
	if c.TimeFormat == "" {
		c.TimeFormat = defaults.TimeFormat
	}
	if c.MaxBackups == 0 {
		c.MaxBackups = defaults.MaxBackups
	}
	if c.MaxSize == 0 {
		c.MaxSize = defaults.MaxSize
	}
	if c.MaxAge == 0 {
		c.MaxAge = defaults.MaxAge
	}
	if c.Format == "" {
		c.Format = defaults.Format
	}
	if c.Director == "" {
		c.Director = defaults.Director
	}
}
