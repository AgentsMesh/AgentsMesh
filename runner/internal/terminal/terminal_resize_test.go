package terminal

import (
	"io"
	"sync"
	"testing"
	"time"
)

type recordingResizePTY struct {
	mu          sync.Mutex
	cols        int
	rows        int
	resizeCalls [][2]int
}

func newRecordingResizePTY(cols, rows int) *recordingResizePTY {
	return &recordingResizePTY{
		cols: cols,
		rows: rows,
	}
}

func (p *recordingResizePTY) Read([]byte) (int, error)        { return 0, io.EOF }
func (p *recordingResizePTY) Write(b []byte) (int, error)     { return len(b), nil }
func (p *recordingResizePTY) Close() error                    { return nil }
func (p *recordingResizePTY) Pid() int                        { return 1 }
func (p *recordingResizePTY) SetReadDeadline(time.Time) error { return nil }
func (p *recordingResizePTY) Wait() (int, error)              { return 0, nil }
func (p *recordingResizePTY) Kill() error                     { return nil }
func (p *recordingResizePTY) GracefulStop() error             { return nil }

func (p *recordingResizePTY) Resize(cols, rows int) error {
	p.mu.Lock()
	p.cols = cols
	p.rows = rows
	p.resizeCalls = append(p.resizeCalls, [2]int{cols, rows})
	p.mu.Unlock()
	return nil
}

func (p *recordingResizePTY) GetSize() (int, int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cols, p.rows, nil
}

func (p *recordingResizePTY) calls() [][2]int {
	p.mu.Lock()
	defer p.mu.Unlock()
	result := make([][2]int, len(p.resizeCalls))
	copy(result, p.resizeCalls)
	return result
}

func resizeTestTerminal(proc *recordingResizePTY) *Terminal {
	return &Terminal{proc: proc, cols: 80, rows: 24}
}

func TestTerminalResizeSameSizeIsNoOp(t *testing.T) {
	proc := newRecordingResizePTY(80, 24)
	term := resizeTestTerminal(proc)

	if err := term.Resize(80, 24); err != nil {
		t.Fatalf("same-size Resize returned error: %v", err)
	}
	if calls := proc.calls(); len(calls) != 0 {
		t.Fatalf("same-size Resize reached real PTY: calls=%v", calls)
	}

	if err := term.Resize(100, 30); err != nil {
		t.Fatalf("Resize returned error: %v", err)
	}
	if term.cols != 100 || term.rows != 30 {
		t.Fatalf("authoritative geometry not updated: got %dx%d", term.cols, term.rows)
	}
}
