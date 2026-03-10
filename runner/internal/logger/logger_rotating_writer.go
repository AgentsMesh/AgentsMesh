package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

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
