package updater

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// ProgressCallback is called with download progress updates.
type ProgressCallback func(downloaded, total int64)

// ProgressWriter wraps an io.Writer and tracks progress.
type ProgressWriter struct {
	writer   io.Writer
	total    int64
	current  int64
	callback ProgressCallback
	mu       sync.Mutex
}

// NewProgressWriter creates a new progress tracking writer.
func NewProgressWriter(w io.Writer, total int64, callback ProgressCallback) *ProgressWriter {
	return &ProgressWriter{
		writer:   w,
		total:    total,
		callback: callback,
	}
}

// Write implements io.Writer.
func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.writer.Write(p)
	if n > 0 {
		pw.mu.Lock()
		pw.current += int64(n)
		if pw.callback != nil {
			pw.callback(pw.current, pw.total)
		}
		pw.mu.Unlock()
	}
	return n, err
}

// Progress returns the current progress as bytes downloaded and total.
func (pw *ProgressWriter) Progress() (int64, int64) {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	return pw.current, pw.total
}

// ConsoleProgress provides console-based progress display.
type ConsoleProgress struct {
	startTime time.Time
	lastPrint time.Time
	width     int
	mu        sync.Mutex
}

// NewConsoleProgress creates a new console progress display.
func NewConsoleProgress() *ConsoleProgress {
	return &ConsoleProgress{
		startTime: time.Now(),
		width:     40, // Progress bar width
	}
}

// Update prints progress to console.
func (cp *ConsoleProgress) Update(downloaded, total int64) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	now := time.Now()
	// Throttle updates to avoid console spam
	if now.Sub(cp.lastPrint) < 100*time.Millisecond && downloaded < total {
		return
	}
	cp.lastPrint = now

	// Calculate percentage
	var percent float64
	if total > 0 {
		percent = float64(downloaded) / float64(total) * 100
	}

	// Calculate speed
	elapsed := now.Sub(cp.startTime).Seconds()
	var speed float64
	if elapsed > 0 {
		speed = float64(downloaded) / elapsed
	}

	// Calculate ETA
	var eta string
	if speed > 0 && total > downloaded {
		remaining := float64(total-downloaded) / speed
		eta = formatDuration(time.Duration(remaining) * time.Second)
	} else if downloaded >= total {
		eta = "done"
	} else {
		eta = "calculating..."
	}

	// Build progress bar
	filled := int(percent / 100 * float64(cp.width))
	if filled > cp.width {
		filled = cp.width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", cp.width-filled)

	// Format sizes
	downloadedStr := formatBytes(downloaded)
	totalStr := formatBytes(total)
	speedStr := formatBytes(int64(speed)) + "/s"

	// Print progress line
	fmt.Printf("\r[%s] %5.1f%% %s/%s %s ETA: %s",
		bar, percent, downloadedStr, totalStr, speedStr, eta)

	// Print newline when done
	if downloaded >= total {
		fmt.Println()
	}
}

// formatBytes formats bytes in human-readable format.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatDuration formats duration in human-readable format.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
}

// SpinnerProgress provides a simple spinner for indeterminate progress.
type SpinnerProgress struct {
	message string
	frames  []string
	current int
	done    chan struct{}
	stopped bool
	mu      sync.Mutex
}

// NewSpinnerProgress creates a new spinner progress display.
func NewSpinnerProgress(message string) *SpinnerProgress {
	return &SpinnerProgress{
		message: message,
		frames:  []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		done:    make(chan struct{}),
	}
}

// Start starts the spinner animation.
func (sp *SpinnerProgress) Start() {
	go func() {
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-sp.done:
				return
			case <-ticker.C:
				sp.mu.Lock()
				frame := sp.frames[sp.current%len(sp.frames)]
				sp.current++
				sp.mu.Unlock()
				fmt.Printf("\r%s %s", frame, sp.message)
			}
		}
	}()
}

// Stop stops the spinner and clears the line.
func (sp *SpinnerProgress) Stop() {
	sp.mu.Lock()
	if sp.stopped {
		sp.mu.Unlock()
		return
	}
	sp.stopped = true
	sp.mu.Unlock()

	close(sp.done)
	fmt.Printf("\r%s\r", strings.Repeat(" ", len(sp.message)+3))
}

// StopWithMessage stops the spinner and prints a final message.
func (sp *SpinnerProgress) StopWithMessage(message string) {
	sp.mu.Lock()
	if sp.stopped {
		sp.mu.Unlock()
		return
	}
	sp.stopped = true
	sp.mu.Unlock()

	close(sp.done)
	fmt.Printf("\r%s\n", message)
}
