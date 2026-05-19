package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Config struct {
	Level      string // debug, info, warn, error
	Format     string // json, text
	FilePath   string // path to log file, empty means stdout only
	MaxSizeMB  int    // max size in MB before rotation (0 = no limit)
	MaxBackups int
}

type Logger struct {
	*slog.Logger
	file   *os.File
	config Config
}

func New(cfg Config) (*Logger, error) {
	var writers []io.Writer

	writers = append(writers, os.Stdout)

	var logFile *os.File

	if cfg.FilePath != "" {
		dir := filepath.Dir(cfg.FilePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}

		f, err := os.OpenFile(cfg.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		logFile = f
		writers = append(writers, f)
	}

	multiWriter := io.MultiWriter(writers...)

	level := parseLevel(cfg.Level)

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	}

	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(multiWriter, opts)
	} else {
		handler = slog.NewTextHandler(multiWriter, opts)
	}

	logger := slog.New(handler)

	return &Logger{
		Logger: logger,
		file:   logFile,
		config: cfg,
	}, nil
}

func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) RotateIfNeeded() error {
	if l.file == nil || l.config.MaxSizeMB <= 0 {
		return nil
	}

	info, err := l.file.Stat()
	if err != nil {
		return err
	}

	maxBytes := int64(l.config.MaxSizeMB) * 1024 * 1024
	if info.Size() < maxBytes {
		return nil
	}

	return l.rotate()
}

func (l *Logger) rotate() error {
	if l.file == nil {
		return nil
	}

	if err := l.file.Close(); err != nil {
		return err
	}

	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.%s", l.config.FilePath, timestamp)
	if err := os.Rename(l.config.FilePath, backupPath); err != nil {
		return err
	}

	f, err := os.OpenFile(l.config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	l.file = f

	l.cleanupBackups()

	return nil
}

func (l *Logger) cleanupBackups() {
	if l.config.MaxBackups <= 0 {
		return
	}

	dir := filepath.Dir(l.config.FilePath)
	base := filepath.Base(l.config.FilePath)
	pattern := base + ".*"

	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return
	}

	filtered := matches[:0]
	for _, m := range matches {
		if !strings.HasPrefix(filepath.Base(m), "._") {
			filtered = append(filtered, m)
		}
	}
	matches = filtered

	if len(matches) > l.config.MaxBackups {
		for i := 0; i < len(matches)-l.config.MaxBackups; i++ {
			os.Remove(matches[i])
		}
	}
}

func parseLevel(level string) slog.Level {
	switch level {
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

func (l *Logger) SetDefault() {
	slog.SetDefault(l.Logger)
}

func (l *Logger) GinWriter() io.Writer {
	if l.file != nil {
		return io.MultiWriter(os.Stdout, l.file)
	}
	return os.Stdout
}
