package updater

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestProgressWriter(t *testing.T) {
	t.Run("writes data and tracks progress", func(t *testing.T) {
		var buf bytes.Buffer
		var lastDownloaded, lastTotal int64

		pw := NewProgressWriter(&buf, 100, func(downloaded, total int64) {
			lastDownloaded = downloaded
			lastTotal = total
		})

		n, err := pw.Write([]byte("hello"))
		assert.NoError(t, err)
		assert.Equal(t, 5, n)
		assert.Equal(t, "hello", buf.String())
		assert.Equal(t, int64(5), lastDownloaded)
		assert.Equal(t, int64(100), lastTotal)

		n, err = pw.Write([]byte(" world"))
		assert.NoError(t, err)
		assert.Equal(t, 6, n)
		assert.Equal(t, int64(11), lastDownloaded)
	})

	t.Run("progress returns current state", func(t *testing.T) {
		var buf bytes.Buffer
		pw := NewProgressWriter(&buf, 100, nil)

		pw.Write([]byte("test"))
		downloaded, total := pw.Progress()
		assert.Equal(t, int64(4), downloaded)
		assert.Equal(t, int64(100), total)
	})
}

func TestProgressWriter_NilCallback(t *testing.T) {
	var buf bytes.Buffer
	pw := NewProgressWriter(&buf, 100, nil)

	n, err := pw.Write([]byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, "test", buf.String())

	downloaded, total := pw.Progress()
	assert.Equal(t, int64(4), downloaded)
	assert.Equal(t, int64(100), total)
}

func TestProgressWriter_MultipleWrites(t *testing.T) {
	var buf bytes.Buffer
	var progressHistory []int64

	pw := NewProgressWriter(&buf, 100, func(downloaded, total int64) {
		progressHistory = append(progressHistory, downloaded)
	})

	pw.Write([]byte("hello"))
	pw.Write([]byte(" "))
	pw.Write([]byte("world"))

	assert.Equal(t, "hello world", buf.String())
	assert.Equal(t, []int64{5, 6, 11}, progressHistory)

	downloaded, total := pw.Progress()
	assert.Equal(t, int64(11), downloaded)
	assert.Equal(t, int64(100), total)
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
		{1125899906842624, "1.0 PB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{0, "0s"},
		{30 * time.Second, "30s"},
		{59 * time.Second, "59s"},
		{60 * time.Second, "1m0s"},
		{61 * time.Second, "1m1s"},
		{90 * time.Second, "1m30s"},
		{59*time.Minute + 59*time.Second, "59m59s"},
		{60 * time.Minute, "1h0m"},
		{61 * time.Minute, "1h1m"},
		{3661 * time.Second, "1h1m"},
		{24 * time.Hour, "24h0m"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConsoleProgress(t *testing.T) {
	cp := NewConsoleProgress()
	assert.NotNil(t, cp)
	assert.Equal(t, 40, cp.width)

	cp.Update(50, 100)
	cp.Update(100, 100)
}

func TestSpinnerProgress(t *testing.T) {
	sp := NewSpinnerProgress("Loading...")
	assert.NotNil(t, sp)
	assert.Equal(t, "Loading...", sp.message)
	assert.Len(t, sp.frames, 10)
}
