package logger

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsLogFile(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		prefix   string
		ext      string
		want     bool
	}{
		{
			name:     "valid date log file",
			filename: "runner-2024-01-15.log",
			prefix:   "runner-",
			ext:      ".log",
			want:     true,
		},
		{
			name:     "valid date log file with backup",
			filename: "runner-2024-01-15.log.0",
			prefix:   "runner-",
			ext:      ".log",
			want:     true,
		},
		{
			name:     "valid date log file with higher backup number",
			filename: "runner-2024-01-15.log.5",
			prefix:   "runner-",
			ext:      ".log",
			want:     true,
		},
		{
			name:     "invalid - wrong prefix",
			filename: "other-2024-01-15.log",
			prefix:   "runner-",
			ext:      ".log",
			want:     false,
		},
		{
			name:     "invalid - wrong extension",
			filename: "runner-2024-01-15.txt",
			prefix:   "runner-",
			ext:      ".log",
			want:     false,
		},
		{
			name:     "invalid - no date",
			filename: "runner.log",
			prefix:   "runner-",
			ext:      ".log",
			want:     false,
		},
		{
			name:     "invalid - wrong date format",
			filename: "runner-01-15-2024.log",
			prefix:   "runner-",
			ext:      ".log",
			want:     false,
		},
		{
			name:     "invalid - too short",
			filename: "runner-.log",
			prefix:   "runner-",
			ext:      ".log",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsLogFile(tt.filename, tt.prefix, tt.ext)
			if got != tt.want {
				t.Errorf("IsLogFile(%q, %q, %q) = %v, want %v",
					tt.filename, tt.prefix, tt.ext, got, tt.want)
			}
		})
	}
}

func TestSortLogFilesByTime(t *testing.T) {
	now := time.Now()
	files := []logFileInfo{
		{path: "c.log", modTime: now.Add(-1 * time.Hour), size: 100},
		{path: "a.log", modTime: now.Add(-3 * time.Hour), size: 100},
		{path: "b.log", modTime: now.Add(-2 * time.Hour), size: 100},
	}

	sortLogFilesByTime(files)

	// Should be sorted oldest first
	if files[0].path != "a.log" || files[1].path != "b.log" || files[2].path != "c.log" {
		t.Errorf("sortLogFilesByTime did not sort correctly: got %v", files)
	}
}

func TestRotatingWriter_DateBasedFilename(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	rw, err := newRotatingWriter(logPath, DefaultMaxFileSize, DefaultMaxBackups, DefaultMaxDirSize)
	if err != nil {
		t.Fatalf("newRotatingWriter failed: %v", err)
	}
	defer rw.Close()

	// Check that current file has date in name
	today := time.Now().Format("2006-01-02")
	expectedPath := filepath.Join(tmpDir, "test-"+today+".log")

	if rw.currentLogPath() != expectedPath {
		t.Errorf("currentLogPath() = %q, want %q", rw.currentLogPath(), expectedPath)
	}

	// Write some data
	_, err = rw.Write([]byte("test log entry\n"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Log file does not exist at expected path: %s", expectedPath)
	}
}

func TestRotatingWriter_SizeRotation(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	// Use small max size for testing
	maxSize := int64(100)
	rw, err := newRotatingWriter(logPath, maxSize, 3, DefaultMaxDirSize)
	if err != nil {
		t.Fatalf("newRotatingWriter failed: %v", err)
	}
	defer rw.Close()

	// Write enough data to trigger rotation
	data := make([]byte, 60)
	for i := range data {
		data[i] = 'x'
	}
	data[59] = '\n'

	// First write (60 bytes)
	_, err = rw.Write(data)
	if err != nil {
		t.Fatalf("Write 1 failed: %v", err)
	}

	// Second write should trigger rotation (60+60 > 100)
	_, err = rw.Write(data)
	if err != nil {
		t.Fatalf("Write 2 failed: %v", err)
	}

	// Check that backup file exists
	today := time.Now().Format("2006-01-02")
	backupPath := filepath.Join(tmpDir, "test-"+today+".log.0")

	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		t.Errorf("Backup file does not exist at: %s", backupPath)
	}
}

func TestRotatingWriter_DirectoryCleanup(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some old log files manually
	oldFiles := []struct {
		name string
		size int
		age  time.Duration
	}{
		{"test-2024-01-01.log", 100 * 1024 * 1024, 72 * time.Hour}, // 100MB, 3 days old
		{"test-2024-01-02.log", 100 * 1024 * 1024, 48 * time.Hour}, // 100MB, 2 days old
		{"test-2024-01-03.log", 100 * 1024 * 1024, 24 * time.Hour}, // 100MB, 1 day old
		{"test-2024-01-04.log", 100 * 1024 * 1024, 12 * time.Hour}, // 100MB, 12 hours old
		{"test-2024-01-05.log", 100 * 1024 * 1024, 6 * time.Hour},  // 100MB, 6 hours old
		{"test-2024-01-06.log", 100 * 1024 * 1024, 1 * time.Hour},  // 100MB, 1 hour old
	}

	now := time.Now()
	for _, f := range oldFiles {
		path := filepath.Join(tmpDir, f.name)
		data := make([]byte, f.size)
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		// Set modification time
		modTime := now.Add(-f.age)
		if err := os.Chtimes(path, modTime, modTime); err != nil {
			t.Fatalf("Failed to set file time: %v", err)
		}
	}

	logPath := filepath.Join(tmpDir, "test.log")
	// Max dir size 250MB - should keep ~2-3 files
	rw, err := newRotatingWriter(logPath, DefaultMaxFileSize, DefaultMaxBackups, 250*1024*1024)
	if err != nil {
		t.Fatalf("newRotatingWriter failed: %v", err)
	}
	defer rw.Close()

	// Give cleanup goroutine time to run
	time.Sleep(100 * time.Millisecond)

	// Count remaining files
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}

	var totalSize int64
	var logCount int
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if IsLogFile(entry.Name(), "test-", ".log") {
			totalSize += info.Size()
			logCount++
		}
	}

	// Total size should be under 250MB (allowing for the new empty file)
	if totalSize > 260*1024*1024 {
		t.Errorf("Directory size %d exceeds max %d", totalSize, 250*1024*1024)
	}

	t.Logf("After cleanup: %d files, total size: %d bytes", logCount, totalSize)
}

func TestInit(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	cfg := Config{
		Level:       "debug",
		FilePath:    logPath,
		Format:      "text",
		MaxFileSize: DefaultMaxFileSize,
		MaxBackups:  DefaultMaxBackups,
		MaxDirSize:  DefaultMaxDirSize,
	}

	err := Init(cfg)
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	defer Close()

	// Log something
	Default().Info("test message")

	// Verify file was created with date
	today := time.Now().Format("2006-01-02")
	expectedPath := filepath.Join(tmpDir, "test-"+today+".log")

	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("Log file not created at: %s", expectedPath)
	}
}
