package terminal

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// PTYLogger records PTY output to files for debugging.
// It captures both raw PTY output and SmartAggregator processed output.
type PTYLogger struct {
	podKey  string
	logDir  string
	rawFile *os.File // raw.log - original PTY output with ANSI sequences
	aggFile *os.File // aggregated.log - SmartAggregator output
	mu      sync.Mutex
	closed  bool
}

// NewPTYLogger creates a new PTY logger for a pod.
// Creates directory: {baseDir}/{podKey}/
// Creates files: raw.log, aggregated.log
func NewPTYLogger(baseDir, podKey string) (*PTYLogger, error) {
	logDir := filepath.Join(baseDir, podKey)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	rawFile, err := os.Create(filepath.Join(logDir, "raw.log"))
	if err != nil {
		return nil, fmt.Errorf("failed to create raw.log: %w", err)
	}

	aggFile, err := os.Create(filepath.Join(logDir, "aggregated.log"))
	if err != nil {
		rawFile.Close()
		return nil, fmt.Errorf("failed to create aggregated.log: %w", err)
	}

	// Write header with timestamp
	header := fmt.Sprintf("# PTY Log for pod %s\n# Started: %s\n\n",
		podKey, time.Now().Format(time.RFC3339))
	rawFile.WriteString(header)
	aggFile.WriteString(header)

	return &PTYLogger{
		podKey:  podKey,
		logDir:  logDir,
		rawFile: rawFile,
		aggFile: aggFile,
	}, nil
}

// WriteRaw writes raw PTY output (before aggregation).
func (l *PTYLogger) WriteRaw(data []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	_, err := l.rawFile.Write(data)
	return err
}

// WriteAggregated writes SmartAggregator processed output.
func (l *PTYLogger) WriteAggregated(data []byte) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	// Add separator between flushes for readability
	l.aggFile.WriteString(fmt.Sprintf("\n--- flush at %s (%d bytes) ---\n",
		time.Now().Format("15:04:05.000"), len(data)))
	_, err := l.aggFile.Write(data)
	return err
}

// LogDir returns the log directory path.
func (l *PTYLogger) LogDir() string {
	return l.logDir
}

// Close closes all log files.
func (l *PTYLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true

	footer := fmt.Sprintf("\n# Ended: %s\n", time.Now().Format(time.RFC3339))
	l.rawFile.WriteString(footer)
	l.aggFile.WriteString(footer)

	var errs []error
	if err := l.rawFile.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := l.aggFile.Close(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("failed to close log files: %v", errs)
	}
	return nil
}
