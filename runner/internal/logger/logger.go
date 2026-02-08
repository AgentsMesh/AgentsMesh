// Package logger provides structured logging for the Runner using slog.
package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// LevelTrace is a custom log level lower than Debug for high-frequency logging.
	// Use Trace for extremely verbose logs that are only useful during deep debugging.
	LevelTrace = slog.Level(-8)

	// DefaultMaxFileSize is the default maximum log file size (10MB)
	DefaultMaxFileSize = 10 * 1024 * 1024
	// DefaultMaxBackups is the default number of backup files to keep
	DefaultMaxBackups = 3
)

// Config holds logger configuration.
type Config struct {
	Level       string // trace, debug, info, warn, error
	FilePath    string // path to log file, empty means stderr only
	Format      string // json, text (default: text)
	MaxFileSize int64  // max file size in bytes before rotation (default: 10MB)
	MaxBackups  int    // max number of backup files to keep (default: 3)
	// Note: File always logs Debug+ regardless of Level setting.
	// Terminal (stderr) follows the Level setting.
}

// Logger wraps slog.Logger with additional functionality.
type Logger struct {
	*slog.Logger
	writer *rotatingWriter
	config Config
}

// rotatingWriter implements io.Writer with log rotation support.
type rotatingWriter struct {
	filePath    string
	maxSize     int64
	maxBackups  int
	currentSize int64
	file        *os.File
	mu          sync.Mutex
}

func newRotatingWriter(filePath string, maxSize int64, maxBackups int) (*rotatingWriter, error) {
	rw := &rotatingWriter{
		filePath:   filePath,
		maxSize:    maxSize,
		maxBackups: maxBackups,
	}

	if err := rw.openFile(); err != nil {
		return nil, err
	}

	return rw, nil
}

func (rw *rotatingWriter) openFile() error {
	// Ensure directory exists
	dir := filepath.Dir(rw.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file (append mode)
	f, err := os.OpenFile(rw.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Get current file size
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return fmt.Errorf("failed to stat log file: %w", err)
	}

	rw.file = f
	rw.currentSize = info.Size()
	return nil
}

func (rw *rotatingWriter) Write(p []byte) (n int, err error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	// Check if rotation is needed
	if rw.currentSize+int64(len(p)) > rw.maxSize {
		if err := rw.rotate(); err != nil {
			// Log rotation failed, but continue writing to current file
			// to avoid losing log data
			fmt.Fprintf(os.Stderr, "log rotation failed: %v\n", err)
		}
	}

	n, err = rw.file.Write(p)
	rw.currentSize += int64(n)
	return n, err
}

func (rw *rotatingWriter) rotate() error {
	// Close current file
	if rw.file != nil {
		rw.file.Close()
	}

	// Remove oldest backup if we have too many
	for i := rw.maxBackups - 1; i >= 0; i-- {
		oldPath := rw.backupPath(i)
		newPath := rw.backupPath(i + 1)

		if i == rw.maxBackups-1 {
			// Remove the oldest backup
			os.Remove(oldPath)
		} else {
			// Rename backup.N to backup.N+1
			if _, err := os.Stat(oldPath); err == nil {
				os.Rename(oldPath, newPath)
			}
		}
	}

	// Rename current log to backup.0
	if _, err := os.Stat(rw.filePath); err == nil {
		os.Rename(rw.filePath, rw.backupPath(0))
	}

	// Open new file
	return rw.openFile()
}

func (rw *rotatingWriter) backupPath(index int) string {
	return fmt.Sprintf("%s.%d", rw.filePath, index)
}

func (rw *rotatingWriter) Close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.file != nil {
		return rw.file.Close()
	}
	return nil
}

// multiHandler dispatches log records to multiple handlers with different levels.
// File handler always logs Debug+, stderr handler follows configured level.
type multiHandler struct {
	fileHandler   slog.Handler // File: Debug level (always)
	stderrHandler slog.Handler // Stderr: configured level
	fileLevel     slog.Level   // Debug
	stderrLevel   slog.Level   // configured level
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// Enabled if either handler accepts this level
	return level >= h.fileLevel || level >= h.stderrLevel
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	// File always logs Debug+ (not Trace)
	if h.fileHandler != nil && r.Level >= h.fileLevel {
		if err := h.fileHandler.Handle(ctx, r); err != nil {
			return err
		}
	}
	// Stderr follows configured level
	if h.stderrHandler != nil && r.Level >= h.stderrLevel {
		if err := h.stderrHandler.Handle(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandler := &multiHandler{
		fileLevel:   h.fileLevel,
		stderrLevel: h.stderrLevel,
	}
	if h.fileHandler != nil {
		newHandler.fileHandler = h.fileHandler.WithAttrs(attrs)
	}
	if h.stderrHandler != nil {
		newHandler.stderrHandler = h.stderrHandler.WithAttrs(attrs)
	}
	return newHandler
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	newHandler := &multiHandler{
		fileLevel:   h.fileLevel,
		stderrLevel: h.stderrLevel,
	}
	if h.fileHandler != nil {
		newHandler.fileHandler = h.fileHandler.WithGroup(name)
	}
	if h.stderrHandler != nil {
		newHandler.stderrHandler = h.stderrHandler.WithGroup(name)
	}
	return newHandler
}

var (
	defaultLogger *Logger
	mu            sync.RWMutex
)

// Init initializes the global logger with the given configuration.
func Init(cfg Config) error {
	logger, err := New(cfg)
	if err != nil {
		return err
	}

	mu.Lock()
	defer mu.Unlock()

	// Close previous logger if exists
	if defaultLogger != nil && defaultLogger.writer != nil {
		defaultLogger.writer.Close()
	}

	defaultLogger = logger
	slog.SetDefault(logger.Logger)
	return nil
}

// New creates a new logger with the given configuration.
// File always logs Debug+ regardless of Level setting.
// Stderr follows the Level setting (default: Info).
func New(cfg Config) (*Logger, error) {
	var rotWriter *rotatingWriter

	// Parse configured log level (for stderr)
	stderrLevel := parseLevel(cfg.Level)
	// File always uses Debug level
	fileLevel := slog.LevelDebug

	// Common ReplaceAttr function for formatting
	replaceAttr := func(groups []string, a slog.Attr) slog.Attr {
		// Custom level name for Trace
		if a.Key == slog.LevelKey {
			if lvl, ok := a.Value.Any().(slog.Level); ok && lvl == LevelTrace {
				return slog.String(slog.LevelKey, "TRACE")
			}
		}
		// Format time as short format for text output
		if a.Key == slog.TimeKey && cfg.Format != "json" {
			if t, ok := a.Value.Any().(time.Time); ok {
				return slog.String(slog.TimeKey, t.Format("15:04:05.000"))
			}
		}
		return a
	}

	// Create stderr handler
	stderrOpts := &slog.HandlerOptions{
		Level:       stderrLevel,
		AddSource:   stderrLevel <= slog.LevelDebug,
		ReplaceAttr: replaceAttr,
	}
	var stderrHandler slog.Handler
	if cfg.Format == "json" {
		stderrHandler = slog.NewJSONHandler(os.Stderr, stderrOpts)
	} else {
		stderrHandler = slog.NewTextHandler(os.Stderr, stderrOpts)
	}

	// Create file handler if file path is configured
	var fileHandler slog.Handler
	if cfg.FilePath != "" {
		maxSize := cfg.MaxFileSize
		if maxSize <= 0 {
			maxSize = DefaultMaxFileSize
		}

		maxBackups := cfg.MaxBackups
		if maxBackups <= 0 {
			maxBackups = DefaultMaxBackups
		}

		rw, err := newRotatingWriter(cfg.FilePath, maxSize, maxBackups)
		if err != nil {
			return nil, err
		}
		rotWriter = rw

		fileOpts := &slog.HandlerOptions{
			Level:       fileLevel,
			AddSource:   true, // Always include source in file logs
			ReplaceAttr: replaceAttr,
		}
		if cfg.Format == "json" {
			fileHandler = slog.NewJSONHandler(rw, fileOpts)
		} else {
			fileHandler = slog.NewTextHandler(rw, fileOpts)
		}
	}

	// Create multi-handler that dispatches to both
	handler := &multiHandler{
		fileHandler:   fileHandler,
		stderrHandler: stderrHandler,
		fileLevel:     fileLevel,
		stderrLevel:   stderrLevel,
	}

	logger := slog.New(handler)

	return &Logger{
		Logger: logger,
		writer: rotWriter,
		config: cfg,
	}, nil
}

// Close closes the log file if open.
func (l *Logger) Close() error {
	if l.writer != nil {
		return l.writer.Close()
	}
	return nil
}

// parseLevel converts string level to slog.Level.
func parseLevel(level string) slog.Level {
	switch level {
	case "trace":
		return LevelTrace
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Close closes the default logger.
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if defaultLogger != nil && defaultLogger.writer != nil {
		return defaultLogger.writer.Close()
	}
	return nil
}

// Default returns the default logger.
func Default() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()

	if defaultLogger != nil {
		return defaultLogger.Logger
	}
	return slog.Default()
}
