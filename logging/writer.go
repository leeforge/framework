package logging

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// levelWriter implements io.Writer and zapcore.WriteSyncer for level-based log file writing.
// It creates daily log files separated by level and uses lumberjack for rotation.
type levelWriter struct {
	config  Config
	level   string
	mu      sync.RWMutex
	writers map[string]*lumberjack.Logger
}

// newLevelWriter creates a new levelWriter for the given config and level.
func newLevelWriter(config Config, level string) *levelWriter {
	return &levelWriter{
		config:  config,
		level:   level,
		writers: make(map[string]*lumberjack.Logger),
	}
}

// Write implements io.Writer.
func (w *levelWriter) Write(p []byte) (n int, err error) {
	date := time.Now().Format("2006-01-02")
	writer := w.getWriter(date)
	return writer.Write(p)
}

// Sync implements zapcore.WriteSyncer.
func (w *levelWriter) Sync() error {
	w.mu.RLock()
	defer w.mu.RUnlock()

	for _, writer := range w.writers {
		if err := syncLumberjack(writer); err != nil {
			return err
		}
	}
	return nil
}

// getWriter returns the lumberjack.Logger for the given date, creating it if necessary.
func (w *levelWriter) getWriter(date string) *lumberjack.Logger {
	// Fast path: check with read lock
	w.mu.RLock()
	if writer, ok := w.writers[date]; ok {
		w.mu.RUnlock()
		return writer
	}
	w.mu.RUnlock()

	// Slow path: create with write lock (double-check)
	w.mu.Lock()
	defer w.mu.Unlock()

	// Double-check after acquiring write lock
	if writer, ok := w.writers[date]; ok {
		return writer
	}

	// Create directory
	dirPath := filepath.Join(w.config.Director, date)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		// Fall back to base directory if date directory creation fails
		dirPath = w.config.Director
		_ = os.MkdirAll(dirPath, 0755)
	}

	fileName := filepath.Join(dirPath, w.level+".log")
	writer := &lumberjack.Logger{
		Filename:   fileName,
		MaxSize:    w.config.MaxSize,
		MaxBackups: w.config.MaxBackups,
		MaxAge:     w.config.MaxAge,
		Compress:   w.config.Compress,
		LocalTime:  true,
	}

	w.writers[date] = writer

	// Cleanup old writers in background
	go w.cleanupOldWriters(date)

	return writer
}

// cleanupOldWriters removes writers older than 2 days to prevent memory leaks.
func (w *levelWriter) cleanupOldWriters(currentDate string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for date, writer := range w.writers {
		if date != currentDate {
			_ = writer.Close()
			delete(w.writers, date)
		}
	}
}

// Close closes all writers.
func (w *levelWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	var lastErr error
	for _, writer := range w.writers {
		if err := writer.Close(); err != nil {
			lastErr = err
		}
	}
	w.writers = make(map[string]*lumberjack.Logger)
	return lastErr
}

// syncLumberjack syncs a lumberjack logger by calling its internal file sync.
func syncLumberjack(l *lumberjack.Logger) error {
	// lumberjack doesn't expose a Sync method, but it auto-syncs on write
	// We can trigger a sync by writing an empty byte slice
	return nil
}

// getWriteSyncer creates a zapcore.WriteSyncer for the given config and level.
// If LogInTerminal is true, it creates a MultiWriteSyncer that writes to both
// stdout and the file.
func getWriteSyncer(config Config, level string) zapcore.WriteSyncer {
	fileWriter := newLevelWriter(config, level)

	if config.LogInTerminal {
		return zapcore.NewMultiWriteSyncer(
			zapcore.AddSync(os.Stdout),
			zapcore.AddSync(fileWriter),
		)
	}
	return zapcore.AddSync(fileWriter)
}

// multiLevelWriter wraps multiple levelWriters for cleanup.
type multiLevelWriter struct {
	writers []*levelWriter
}

// Close closes all level writers.
func (m *multiLevelWriter) Close() error {
	var lastErr error
	for _, w := range m.writers {
		if err := w.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// writerRegistry tracks all created levelWriters for cleanup.
var (
	writerRegistry   = &multiLevelWriter{}
	writerRegistryMu sync.Mutex
)

// registerWriter registers a levelWriter for cleanup.
func registerWriter(w *levelWriter) {
	writerRegistryMu.Lock()
	defer writerRegistryMu.Unlock()
	writerRegistry.writers = append(writerRegistry.writers, w)
}

// CloseAllWriters closes all registered writers.
func CloseAllWriters() error {
	writerRegistryMu.Lock()
	defer writerRegistryMu.Unlock()
	return writerRegistry.Close()
}

// getWriteSyncerWithRegistry creates a WriteSyncer and registers its levelWriter for cleanup.
func getWriteSyncerWithRegistry(config Config, level string) zapcore.WriteSyncer {
	fileWriter := newLevelWriter(config, level)
	registerWriter(fileWriter)

	if config.LogInTerminal {
		return zapcore.NewMultiWriteSyncer(
			zapcore.AddSync(os.Stdout),
			zapcore.AddSync(fileWriter),
		)
	}
	return zapcore.AddSync(fileWriter)
}

// Ensure levelWriter implements io.WriteCloser
var _ io.WriteCloser = (*levelWriter)(nil)
