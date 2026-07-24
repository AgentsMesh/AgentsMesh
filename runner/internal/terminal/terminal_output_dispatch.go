package terminal

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

const (
	handlerBlockedThreshold   = 5 * time.Second
	handlerSlowWarnThreshold  = 50 * time.Millisecond
	handlerSlowErrorThreshold = 1 * time.Second
)

func (t *Terminal) dispatchOutput(data []byte, readCount int) {
	log := logger.TerminalTrace()
	log.Trace("PTY read SUCCESS",
		"label", t.label,
		"read_num", readCount,
		"bytes", len(data))

	t.mu.Lock()
	handler := t.onOutput
	t.mu.Unlock()
	if handler == nil {
		log.Warn("No output handler set", "label", t.label, "read_num", readCount)
		return
	}

	start := time.Now()
	watchdogDone := make(chan struct{})
	go t.watchOutputHandler(watchdogDone, start, readCount, len(data))
	handler(data)
	close(watchdogDone)

	elapsed := time.Since(start)
	if elapsed > handlerSlowErrorThreshold {
		logger.Terminal().Error("PTY output handler extremely slow",
			"label", t.label,
			"read_num", readCount,
			"bytes", len(data),
			"handler_time", elapsed)
	} else if elapsed > handlerSlowWarnThreshold {
		log.Warn("PTY output handler slow",
			"label", t.label,
			"read_num", readCount,
			"bytes", len(data),
			"handler_time", elapsed)
	}
}

func (t *Terminal) watchOutputHandler(done <-chan struct{}, started time.Time, readCount, byteCount int) {
	timer := time.NewTimer(handlerBlockedThreshold)
	defer timer.Stop()
	select {
	case <-done:
		return
	case <-timer.C:
	}

	stackBuf := make([]byte, 64*1024)
	stackLen := runtime.Stack(stackBuf, true)
	dumpPath := ""
	stackDumpDir := config.TempBaseDir()
	_ = os.MkdirAll(stackDumpDir, 0755)
	dumpFile := filepath.Join(stackDumpDir, fmt.Sprintf("blocked-%s-%d.stacks",
		t.label, time.Now().Unix()))
	if err := os.WriteFile(dumpFile, stackBuf[:stackLen], 0644); err == nil {
		dumpPath = dumpFile
	}

	logger.Terminal().Error("PTY output handler BLOCKED - possible deadlock",
		"label", t.label,
		"read_num", readCount,
		"bytes", byteCount,
		"blocked_for", time.Since(started),
		"goroutine_dump", dumpPath)
}
