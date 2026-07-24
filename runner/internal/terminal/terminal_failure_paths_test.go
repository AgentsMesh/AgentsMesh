package terminal

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/config"
)

type failurePathPTY struct {
	readFn      func([]byte) (int, error)
	closeFn     func()
	resizeErr   error
	gracefulErr error
	killErr     error
	mu          sync.Mutex
	closeCalls  int
	killCalls   int
}

func (p *failurePathPTY) Read(data []byte) (int, error) {
	if p.readFn != nil {
		return p.readFn(data)
	}
	return 0, io.EOF
}
func (p *failurePathPTY) Write(data []byte) (int, error) { return len(data), nil }
func (p *failurePathPTY) Close() error {
	p.mu.Lock()
	p.closeCalls++
	p.mu.Unlock()
	if p.closeFn != nil {
		p.closeFn()
	}
	return nil
}
func (p *failurePathPTY) Resize(int, int) error           { return p.resizeErr }
func (p *failurePathPTY) GetSize() (int, int, error)      { return 80, 24, nil }
func (p *failurePathPTY) Pid() int                        { return 123 }
func (p *failurePathPTY) SetReadDeadline(time.Time) error { return nil }
func (p *failurePathPTY) Wait() (int, error)              { return 0, nil }
func (p *failurePathPTY) GracefulStop() error             { return p.gracefulErr }
func (p *failurePathPTY) Kill() error {
	p.mu.Lock()
	p.killCalls++
	p.mu.Unlock()
	return p.killErr
}

func bareFailureTerminal(proc ptyProcess) *Terminal {
	return &Terminal{
		proc: proc, doneCh: make(chan struct{}), readDoneCh: make(chan struct{}),
		readActiveCh: make(chan struct{}), readProgressCh: make(chan struct{}, 1),
		resumeCh: make(chan struct{}, 1),
	}
}

func TestTerminalStartRejectsAlreadyStarted(t *testing.T) {
	terminal, err := New(Options{Command: "fake"})
	if err != nil {
		t.Fatal(err)
	}
	terminal.proc = &failurePathPTY{}
	if err := terminal.Start(); err == nil || err.Error() != "terminal is already started" {
		t.Fatalf("Start = %v", err)
	}
}

func TestTerminalStopEscalatesAndTimesOut(t *testing.T) {
	proc := &failurePathPTY{gracefulErr: errors.New("graceful"), killErr: errors.New("kill")}
	terminal := bareFailureTerminal(proc)
	started := time.Now()
	terminal.Stop()
	if elapsed := time.Since(started); elapsed < gracefulStopTimeout+time.Second {
		t.Fatalf("Stop returned before both escalation timeouts: %v", elapsed)
	}
	if !terminal.closed || proc.killCalls != 1 || proc.closeCalls != 1 {
		t.Fatalf("escalation state: closed=%v kills=%d closes=%d", terminal.closed, proc.killCalls, proc.closeCalls)
	}
}

func TestTerminalDetachAlreadyClosed(t *testing.T) {
	terminal := bareFailureTerminal(&failurePathPTY{})
	terminal.closed = true
	terminal.Detach()
}

func TestTerminalDetachClosesReaderWithoutStoppingDaemon(t *testing.T) {
	proc := &failurePathPTY{}
	terminal := bareFailureTerminal(proc)

	terminal.Detach()

	if !terminal.stopping || !terminal.closed {
		t.Fatalf("detach state: stopping=%v closed=%v", terminal.stopping, terminal.closed)
	}
	if proc.closeCalls != 1 {
		t.Fatalf("detach close calls = %d, want 1", proc.closeCalls)
	}
	if proc.killCalls != 0 {
		t.Fatalf("detach kill calls = %d, want 0", proc.killCalls)
	}
}

func TestCleanupOldStackDumpsRemovesOnlyAncientDiagnostics(t *testing.T) {
	if err := os.MkdirAll(config.TempBaseDir(), 0755); err != nil {
		t.Fatal(err)
	}
	stamp := time.Now().UnixNano()
	old := filepath.Join(config.TempBaseDir(), "coverage-old-"+time.Unix(0, stamp).Format("150405.000000000")+".stacks")
	nonStack := old + ".txt"
	directory := old + "-dir.stacks"
	for _, path := range []string{old, nonStack} {
		if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Remove(path) })
	}
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Remove(directory) })
	ancient := time.Unix(1, 0)
	if err := os.Chtimes(old, ancient, ancient); err != nil {
		t.Fatal(err)
	}

	CleanupOldStackDumps(50 * 365 * 24 * time.Hour)
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Fatalf("ancient stack dump retained: %v", err)
	}
	if _, err := os.Stat(nonStack); err != nil {
		t.Fatalf("non-stack fixture removed: %v", err)
	}
}

func TestCleanupOldStackDumpsMissingDirectoryIsSafe(t *testing.T) {
	cleanupOldStackDumps(filepath.Join(t.TempDir(), "missing"), time.Hour)
}

func TestReadOutputBackpressureAndReplacementPaths(t *testing.T) {
	t.Run("closed while paused", func(t *testing.T) {
		terminal := bareFailureTerminal(&failurePathPTY{})
		terminal.readPaused = true
		terminal.closed = true
		terminal.readOutput()
	})

	t.Run("continues while paused until close", func(t *testing.T) {
		terminal := bareFailureTerminal(&failurePathPTY{})
		terminal.readPaused = true
		go func() {
			time.Sleep(150 * time.Millisecond)
			terminal.mu.Lock()
			terminal.closed = true
			terminal.mu.Unlock()
		}()
		terminal.readOutput()
	})

	t.Run("ordered read observes replacement", func(t *testing.T) {
		terminal := bareFailureTerminal(&failurePathPTY{})
		terminal.readOrder = closingReadLocker{terminal: terminal}
		terminal.readOutput()
	})

	t.Run("fatal error after expected process exit", func(t *testing.T) {
		terminal := bareFailureTerminal(&failurePathPTY{readFn: func([]byte) (int, error) {
			return 0, errors.New("read failed")
		}})
		terminal.processExited = true
		terminal.readOutput()
	})
}

type closingReadLocker struct{ terminal *Terminal }

func (l closingReadLocker) Lock() {
	l.terminal.mu.Lock()
	l.terminal.closed = true
	l.terminal.mu.Unlock()
}
func (closingReadLocker) Unlock() {}

func TestDrainReadOutputIdleAndProgressRace(t *testing.T) {
	t.Run("not started", func(t *testing.T) {
		bareFailureTerminal(&failurePathPTY{}).drainReadOutput()
	})

	t.Run("idle closes PTY and joins reader", func(t *testing.T) {
		done := make(chan struct{})
		var once sync.Once
		proc := &failurePathPTY{closeFn: func() { once.Do(func() { close(done) }) }}
		terminal := bareFailureTerminal(proc)
		terminal.readStarted = true
		terminal.readDoneCh = done
		close(terminal.readActiveCh)
		terminal.drainReadOutput()
		if proc.closeCalls != 1 {
			t.Fatalf("idle drain close calls = %d", proc.closeCalls)
		}
	})

	t.Run("progress at timer boundary extends drain", func(t *testing.T) {
		terminal := bareFailureTerminal(&failurePathPTY{})
		terminal.readStarted = true
		close(terminal.readActiveCh)
		var gate sync.Mutex
		gate.Lock()
		terminal.readOrder = &gate
		drained := make(chan struct{})
		go func() {
			terminal.drainReadOutput()
			close(drained)
		}()
		time.Sleep(readDrainGrace + 30*time.Millisecond)
		terminal.mu.Lock()
		terminal.readProgress++
		terminal.mu.Unlock()
		gate.Unlock()
		time.Sleep(20 * time.Millisecond)
		close(terminal.readDoneCh)
		select {
		case <-drained:
		case <-time.After(time.Second):
			t.Fatal("progress-extended drain did not finish")
		}
	})

	timer := time.NewTimer(time.Millisecond)
	<-timer.C
	resetDrainTimer(timer)
	if !timer.Stop() {
		t.Fatal("reset timer was not active")
	}
}

func TestTerminalResizePropagatesPTYError(t *testing.T) {
	want := errors.New("resize")
	terminal := bareFailureTerminal(&failurePathPTY{resizeErr: want})
	terminal.cols, terminal.rows = 80, 24
	if err := terminal.Resize(100, 30); !errors.Is(err, want) {
		t.Fatalf("Resize = %v, want %v", err, want)
	}
}
