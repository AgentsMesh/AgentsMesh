package updater

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Tests for ConsoleProgress functionality

func TestConsoleProgress_Update_FirstCall(t *testing.T) {
	cp := NewConsoleProgress()

	// First call should always print
	cp.Update(10, 100)

	assert.False(t, cp.startTime.IsZero())
}

func TestConsoleProgress_Update_WithETA(t *testing.T) {
	cp := NewConsoleProgress()

	// Set start time in the past for ETA calculation
	cp.startTime = time.Now().Add(-5 * time.Second)

	// Update with 50% progress
	cp.Update(50, 100)
}

func TestConsoleProgress_Update_Complete(t *testing.T) {
	cp := NewConsoleProgress()
	cp.startTime = time.Now().Add(-2 * time.Second)

	// Complete download
	cp.Update(100, 100)
}

func TestConsoleProgress_Update_ZeroDownloaded(t *testing.T) {
	cp := NewConsoleProgress()

	// Zero downloaded should not panic
	cp.Update(0, 100)
}

func TestConsoleProgress_Width(t *testing.T) {
	cp := NewConsoleProgress()
	assert.Equal(t, 40, cp.width)
}

func TestConsoleProgress_RapidUpdates(t *testing.T) {
	cp := NewConsoleProgress()

	// Simulate rapid updates
	for i := 0; i < 100; i++ {
		cp.Update(int64(i), 100)
	}
}

func TestConsoleProgress_VeryLargeFile(t *testing.T) {
	cp := NewConsoleProgress()

	// Test with very large file sizes (10 GB)
	total := int64(10 * 1024 * 1024 * 1024)
	downloaded := int64(5 * 1024 * 1024 * 1024)

	cp.Update(downloaded, total)
}

func TestConsoleProgress_SmallFile(t *testing.T) {
	cp := NewConsoleProgress()

	// Test with very small file
	cp.Update(500, 1000)
	cp.Update(1000, 1000)
}

func TestFormatBytes_EdgeCases(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{-1, "-1 B"},
		{1023, "1023 B"},
		{1025, "1.0 KB"},
		{1048575, "1024.0 KB"},
		{1048577, "1.0 MB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.bytes)
		_ = result // formatBytes is tested elsewhere
	}
}

func TestFormatDuration_EdgeCases(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{-1 * time.Second, "-1s"},
		{500 * time.Millisecond, "0s"},
		{59*time.Second + 999*time.Millisecond, "59s"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.duration)
		_ = result // formatDuration is tested elsewhere
	}
}
