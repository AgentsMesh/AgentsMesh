package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

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
		if !IsLogFile(name, prefix, rw.ext) {
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

// IsLogFile checks if a filename matches the rotated log file pattern.
// Pattern: prefix + YYYY-MM-DD + ext (e.g., runner-2024-01-15.log)
// Or: prefix + YYYY-MM-DD + ext + .N (e.g., runner-2024-01-15.log.0)
func IsLogFile(name, prefix, ext string) bool {
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

// rotate performs size-based log rotation within the current day.
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
