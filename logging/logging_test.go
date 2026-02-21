package logging

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Director != "logs" {
		t.Errorf("expected Director 'logs', got '%s'", cfg.Director)
	}
	if cfg.Level != "info" {
		t.Errorf("expected Level 'info', got '%s'", cfg.Level)
	}
	if cfg.Format != "json" {
		t.Errorf("expected Format 'json', got '%s'", cfg.Format)
	}
	if !cfg.LogInTerminal {
		t.Error("expected LogInTerminal to be true")
	}
}

func TestConfigTransportLevel(t *testing.T) {
	tests := []struct {
		level    string
		expected zapcore.Level
	}{
		{"debug", zapcore.DebugLevel},
		{"DEBUG", zapcore.DebugLevel},
		{"info", zapcore.InfoLevel},
		{"INFO", zapcore.InfoLevel},
		{"warn", zapcore.WarnLevel},
		{"WARN", zapcore.WarnLevel},
		{"error", zapcore.ErrorLevel},
		{"ERROR", zapcore.ErrorLevel},
		{"dpanic", zapcore.DPanicLevel},
		{"panic", zapcore.PanicLevel},
		{"fatal", zapcore.FatalLevel},
		{"unknown", zapcore.DebugLevel},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			cfg := Config{Level: tt.level}
			if got := cfg.TransportLevel(); got != tt.expected {
				t.Errorf("TransportLevel() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfigZapEncodeLevel(t *testing.T) {
	tests := []struct {
		encodeLevel string
		notNil      bool
	}{
		{"LowercaseLevelEncoder", true},
		{"LowercaseColorLevelEncoder", true},
		{"CapitalLevelEncoder", true},
		{"CapitalColorLevelEncoder", true},
		{"unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.encodeLevel, func(t *testing.T) {
			cfg := Config{EncodeLevel: tt.encodeLevel}
			encoder := cfg.ZapEncodeLevel()
			if (encoder != nil) != tt.notNil {
				t.Errorf("ZapEncodeLevel() nil = %v, want nil = %v", encoder == nil, !tt.notNil)
			}
		})
	}
}

func TestNewLoggerWithConsoleOutput(t *testing.T) {
	// Create a config that outputs to terminal only (no file)
	cfg := DefaultConfig()
	cfg.LogInTerminal = true
	cfg.Director = t.TempDir() // Use temp dir for any file output

	logger := NewLogger(cfg)
	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}

	// Test that we can log without panicking
	logger.Info("test message", zap.String("key", "value"))
}

func TestLoggerWith(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Director = t.TempDir()

	logger := NewLogger(cfg)
	childLogger := logger.With(zap.String("component", "test"))

	if childLogger == nil {
		t.Fatal("With returned nil")
	}

	// Should be a different logger instance
	if childLogger == logger {
		t.Error("With should return a new logger instance")
	}
}

func TestLoggerNamed(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Director = t.TempDir()

	logger := NewLogger(cfg)
	namedLogger := logger.Named("mylogger")

	if namedLogger == nil {
		t.Fatal("Named returned nil")
	}
}

func TestLoggerWithError(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Director = t.TempDir()

	logger := NewLogger(cfg)
	errLogger := logger.WithError(os.ErrNotExist)

	if errLogger == nil {
		t.Fatal("WithError returned nil")
	}
}

func TestFactory(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Director = t.TempDir()

	factory := NewFactory(cfg)

	logger1 := factory.GetLogger("service1")
	logger2 := factory.GetLogger("service1")
	logger3 := factory.GetLogger("service2")

	// Same name should return same logger
	if logger1.Zap() != logger2.Zap() {
		t.Error("GetLogger should return same logger for same name")
	}

	// Different name should return different logger
	if logger1.Zap() == logger3.Zap() {
		t.Error("GetLogger should return different logger for different name")
	}
}

func TestContextFunctions(t *testing.T) {
	ctx := context.Background()

	// Test SetTraceID/GetTraceID
	ctx = SetTraceID(ctx, "trace-123")
	if got := GetTraceID(ctx); got != "trace-123" {
		t.Errorf("GetTraceID() = %v, want %v", got, "trace-123")
	}

	// Test SetSpanID/GetSpanID
	ctx = SetSpanID(ctx, "span-456")
	if got := GetSpanID(ctx); got != "span-456" {
		t.Errorf("GetSpanID() = %v, want %v", got, "span-456")
	}

	// Test SetRequestID/GetRequestID
	ctx = SetRequestID(ctx, "req-789")
	if got := GetRequestID(ctx); got != "req-789" {
		t.Errorf("GetRequestID() = %v, want %v", got, "req-789")
	}

	// Test SetUserID/GetUserID
	ctx = SetUserID(ctx, "user-abc")
	if got := GetUserID(ctx); got != "user-abc" {
		t.Errorf("GetUserID() = %v, want %v", got, "user-abc")
	}
}

func TestContextLoggerStorage(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Director = t.TempDir()

	logger := NewLogger(cfg)
	ctx := context.Background()

	// Store logger in context
	ctx = ToContext(ctx, logger)

	// Retrieve logger from context
	retrieved := FromContext(ctx)
	if retrieved.Zap() != logger.Zap() {
		t.Error("FromContext should return the stored logger")
	}
}

func TestFromContextWithNil(t *testing.T) {
	// Should return global logger when context is nil
	logger := FromContext(context.TODO())
	if logger == nil {
		t.Error("FromContext(nil) should return global logger, not nil")
	}
}

func TestGlobalLogger(t *testing.T) {
	// Global should never return nil
	logger := Global()
	if logger == nil {
		t.Fatal("Global() returned nil")
	}

	// Should be able to log
	logger.Info("global logger test")
}

func TestSetGlobal(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Director = t.TempDir()

	newLogger := NewLogger(cfg)
	SetGlobal(newLogger)

	if Global().Zap() != newLogger.Zap() {
		t.Error("SetGlobal should replace the global logger")
	}
}

func TestPackageLevelFunctions(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Director = t.TempDir()
	Init(cfg)

	// These should not panic
	Debug("debug message", zap.String("key", "value"))
	Info("info message", zap.Int("count", 42))
	Warn("warn message")
	Error("error message")

	Debugf("debug %s", "formatted")
	Infof("info %d", 123)
	Warnf("warn %v", true)
	Errorf("error %s", "test")

	withLogger := With(zap.String("service", "test"))
	if withLogger == nil {
		t.Error("With should return a logger")
	}

	errLogger := WithError(os.ErrNotExist)
	if errLogger == nil {
		t.Error("WithError should return a logger")
	}

	namedLogger := Named("testpkg")
	if namedLogger == nil {
		t.Error("Named should return a logger")
	}
}

func TestEncoder(t *testing.T) {
	cfg := DefaultConfig()

	// Test JSON encoder
	cfg.Format = "json"
	jsonEncoder := GetEncoder(cfg)
	if jsonEncoder == nil {
		t.Error("GetEncoder should return non-nil for json format")
	}

	// Test console encoder
	cfg.Format = "console"
	consoleEncoder := GetEncoder(cfg)
	if consoleEncoder == nil {
		t.Error("GetEncoder should return non-nil for console format")
	}
}

func TestCusTimeEncoder(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Prefix = "[TEST] "
	cfg.TimeFormat = "2006-01-02"

	encoder := CusTimeEncoder(cfg)
	if encoder == nil {
		t.Error("CusTimeEncoder should return non-nil")
	}
}

func TestHook(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Director = t.TempDir()

	logger := NewLogger(cfg)

	hook := func(entry zapcore.Entry) error {
		return nil
	}

	hookedLogger := WithHook(logger, hook)
	hookedLogger.Info("test")

	if hookedLogger == nil {
		t.Error("WithHook should return non-nil logger")
	}
}

func TestLevelWriter(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Director = t.TempDir()

	writer := newLevelWriter(cfg, "info")

	// Test Write
	n, err := writer.Write([]byte("test log line\n"))
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}
	if n == 0 {
		t.Error("Write should return bytes written")
	}

	// Test Sync
	if err := writer.Sync(); err != nil {
		t.Errorf("Sync failed: %v", err)
	}

	// Test Close
	if err := writer.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestLoggerZapAndSugar(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Director = t.TempDir()

	logger := NewLogger(cfg)

	// Test Zap()
	zapLogger := logger.Zap()
	if zapLogger == nil {
		t.Error("Zap() should return non-nil")
	}

	// Test Sugar()
	sugarLogger := logger.Sugar()
	if sugarLogger == nil {
		t.Error("Sugar() should return non-nil")
	}
}

func TestLoggerSync(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Director = t.TempDir()

	logger := NewLogger(cfg)
	logger.Info("test message")

	err := logger.Sync()
	// Sync may return error on some systems (e.g., stdout not syncable)
	// but it shouldn't panic
	_ = err
}

func TestConfigApplyDefaults(t *testing.T) {
	cfg := Config{}
	cfg.applyDefaults()

	if cfg.MessageKey != "message" {
		t.Errorf("expected MessageKey 'message', got '%s'", cfg.MessageKey)
	}
	if cfg.Format != "json" {
		t.Errorf("expected Format 'json', got '%s'", cfg.Format)
	}
	if cfg.MaxSize != 100 {
		t.Errorf("expected MaxSize 100, got %d", cfg.MaxSize)
	}
}

func TestWithContextAddsFields(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Director = t.TempDir()

	logger := NewLogger(cfg)
	ctx := context.Background()
	ctx = SetTraceID(ctx, "trace-123")
	ctx = SetSpanID(ctx, "span-456")

	ctxLogger := WithContext(logger, ctx)
	if ctxLogger == nil {
		t.Fatal("WithContext returned nil")
	}

	// The context logger should be different from original
	if ctxLogger == logger {
		t.Error("WithContext should return a new logger with fields")
	}
}

func TestWithContextNilContext(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Director = t.TempDir()

	logger := NewLogger(cfg)
	result := WithContext(logger, nil)

	if result != logger {
		t.Error("WithContext(nil) should return the original logger")
	}
}

func TestGetTraceIDNilContext(t *testing.T) {
	if got := GetTraceID(context.TODO()); got != "" {
		t.Errorf("GetTraceID(nil) = %v, want empty string", got)
	}
}

func TestCloseAllWriters(t *testing.T) {
	// This is a cleanup function, just ensure it doesn't panic
	err := CloseAllWriters()
	if err != nil {
		t.Logf("CloseAllWriters returned error (may be expected): %v", err)
	}
}

// TestOutputFormat verifies that JSON format produces valid JSON output
func TestOutputFormat(t *testing.T) {
	var buf bytes.Buffer

	// Create encoder config
	cfg := DefaultConfig()
	cfg.Format = "json"

	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "message",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(&buf),
		zapcore.InfoLevel,
	)

	logger := zap.New(core)
	logger.Info("test message", zap.String("key", "value"))

	output := buf.String()
	if !strings.Contains(output, `"message":"test message"`) {
		t.Errorf("JSON output should contain message field, got: %s", output)
	}
	if !strings.Contains(output, `"key":"value"`) {
		t.Errorf("JSON output should contain key field, got: %s", output)
	}
}
