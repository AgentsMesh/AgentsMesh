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

// rotatingWriter implements io.Writer with log rotation support.
// It supports daily log files with size-based rotation and directory size limits.
type rotatingWriter struct {
	baseDir     string // log directory
	baseName    string // base name without extension (e.g., "runner")
	ext         string // file extension (e.g., ".log")
	maxSize     int64  // max size per file before rotation
	maxBackups  int    // max backup files per day
	maxDirSize  int64  // max total directory size
	currentDate string // current date string (YYYY-MM-DD)
	currentSize int64  // current file size
	file        *os.File
	mu          sync.Mutex
}

// logFileInfo holds information about a log file for cleanup purposes.
type logFileInfo struct {
	path    string
	modTime time.Time
	size    int64
}

func newRotatingWriter(filePath string, maxSize int64, maxBackups int, maxDirSize int64) (*rotatingWriter, error) {
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)]

	rw := &rotatingWriter{
		baseDir:    dir,
		baseName:   name,
		ext:        ext,
		maxSize:    maxSize,
		maxBackups: maxBackups,
		maxDirSize: maxDirSize,
	}

	if err := rw.openFile(); err != nil {
		return nil, err
	}

	// Clean up old logs on startup
	rw.cleanupOldLogs()

	return rw, nil
}

// currentLogPath returns the log file path for the current date.
// Format: baseName-YYYY-MM-DD.ext (e.g., runner-2024-01-15.log)
func (rw *rotatingWriter) currentLogPath() string {
	return filepath.Join(rw.baseDir, fmt.Sprintf("%s-%s%s", rw.baseName, rw.currentDate, rw.ext))
}

func (rw *rotatingWriter) openFile() error {
	// Ensure directory exists
	if err := os.MkdirAll(rw.baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Update current date
	rw.currentDate = time.Now().Format("2006-01-02")

	// Open log file for current date (append mode)
	filePath := rw.currentLogPath()
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
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

	// Check if date has changed (new day)
	today := time.Now().Format("2006-01-02")
	if today != rw.currentDate {
		if err := rw.switchToNewDay(today); err != nil {
			fmt.Fprintf(os.Stderr, "failed to switch to new day log: %v\n", err)
		}
	}

	// Check if size-based rotation is needed
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

// switchToNewDay closes current file and opens a new file for the new date.
func (rw *rotatingWriter) switchToNewDay(newDate string) error {
	// Close current file
	if rw.file != nil {
		rw.file.Close()
	}

	// Update date and open new file
	rw.currentDate = newDate
	filePath := rw.currentLogPath()

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open new day log file: %w", err)
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return fmt.Errorf("failed to stat new day log file: %w", err)
	}

	rw.file = f
	rw.currentSize = info.Size()

	// Clean up old logs after switching to new day
	go rw.cleanupOldLogs()

	return nil
}

func (rw *rotatingWriter) rotate() error {
	// Close current file
	if rw.file != nil {
		rw.file.Close()
	}

	currentPath := rw.currentLogPath()

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
	if _, err := os.Stat(currentPath); err == nil {
		os.Rename(currentPath, rw.backupPath(0))
	}

	// Open new file (same date, new file)
	filePath := rw.currentLogPath()
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open rotated log file: %w", err)
	}

	rw.file = f
	rw.currentSize = 0

	// Clean up old logs after rotation
	go rw.cleanupOldLogs()

	return nil
}

// backupPath returns the backup file path for the current date.
// Format: baseName-YYYY-MM-DD.ext.N (e.g., runner-2024-01-15.log.0)
func (rw *rotatingWriter) backupPath(index int) string {
	return fmt.Sprintf("%s.%d", rw.currentLogPath(), index)
}

// cleanupOldLogs removes old log files to keep total directory size under maxDirSize.
func (rw *rotatingWriter) cleanupOldLogs() {
	if rw.maxDirSize <= 0 {
		return
	}

	var files []logFileInfo
	var totalSize int64

	entries, err := os.ReadDir(rw.baseDir)
	if err != nil {
		return
	}

	// Pattern: baseName-*.ext or baseName-*.ext.N
	prefix := rw.baseName + "-"
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Check if file matches our log pattern
		if !isLogFile(name, prefix, rw.ext) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, logFileInfo{
			path:    filepath.Join(rw.baseDir, name),
			modTime: info.ModTime(),
			size:    info.Size(),
		})
		totalSize += info.Size()
	}

	// If under limit, nothing to do
	if totalSize <= rw.maxDirSize {
		return
	}

	// Sort by modification time (oldest first)
	sortLogFilesByTime(files)

	// Remove oldest files until we're under the limit
	for _, f := range files {
		if totalSize <= rw.maxDirSize {
			break
		}

		// Don't delete current log file
		if f.path == rw.currentLogPath() {
			continue
		}

		if err := os.Remove(f.path); err == nil {
			totalSize -= f.size
		}
	}
}

// isLogFile checks if a filename matches the log file pattern.
func isLogFile(name, prefix, ext string) bool {
	// Match: prefix + date + ext (e.g., runner-2024-01-15.log)
	// Or: prefix + date + ext + .N (e.g., runner-2024-01-15.log.0)
	if len(name) < len(prefix)+len(ext)+10 { // 10 = len("YYYY-MM-DD")
		return false
	}

	if name[:len(prefix)] != prefix {
		return false
	}

	// Check for date pattern after prefix
	rest := name[len(prefix):]
	if len(rest) < 10 {
		return false
	}

	// Validate date format (YYYY-MM-DD)
	dateStr := rest[:10]
	if _, err := time.Parse("2006-01-02", dateStr); err != nil {
		return false
	}

	// After date, should be ext or ext.N
	afterDate := rest[10:]
	if afterDate == ext {
		return true
	}

	// Check for .ext.N pattern
	if len(afterDate) > len(ext)+1 && afterDate[:len(ext)] == ext && afterDate[len(ext)] == '.' {
		return true
	}

	return false
}

// sortLogFilesByTime sorts log files by modification time (oldest first).
func sortLogFilesByTime(files []logFileInfo) {
	for i := 0; i < len(files)-1; i++ {
		for j := i + 1; j < len(files); j++ {
			if files[j].modTime.Before(files[i].modTime) {
				files[i], files[j] = files[j], files[i]
			}
		}
	}
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
