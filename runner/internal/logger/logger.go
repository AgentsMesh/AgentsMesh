// Package logger provides structured logging for the Runner using slog.
package logger

import (
	"log/slog"
	"os"
	"sync"
	"time"
)

const (
	// LevelTrace is a custom log level lower than Debug for high-frequency logging.
	// Use Trace for extremely verbose logs that are only useful during deep debugging.
	LevelTrace = slog.Level(-8)

	// DefaultMaxFileSize is the default maximum log file size per day (10MB)
	DefaultMaxFileSize = 10 * 1024 * 1024
	// DefaultMaxBackups is the default number of backup files to keep per day
	DefaultMaxBackups = 3
	// DefaultMaxDirSize is the default maximum total size of all log files (500MB)
	DefaultMaxDirSize = 500 * 1024 * 1024
)

// Config holds logger configuration.
type Config struct {
	Level       string // trace, debug, info, warn, error
	FilePath    string // path to log file, empty means stderr only
	Format      string // json, text (default: text)
	MaxFileSize int64  // max file size in bytes before rotation (default: 10MB)
	MaxBackups  int    // max number of backup files to keep per day (default: 3)
	MaxDirSize  int64  // max total size of all log files in directory (default: 500MB)
	// Note: File always logs Debug+ regardless of Level setting.
	// Terminal (stderr) follows the Level setting.
}

// Logger wraps slog.Logger with additional functionality.
type Logger struct {
	*slog.Logger
	writer *rotatingWriter
	config Config
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

		maxDirSize := cfg.MaxDirSize
		if maxDirSize <= 0 {
			maxDirSize = DefaultMaxDirSize
		}

		rw, err := newRotatingWriter(cfg.FilePath, maxSize, maxBackups, maxDirSize)
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
