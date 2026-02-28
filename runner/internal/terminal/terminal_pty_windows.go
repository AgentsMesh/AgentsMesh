//go:build windows

package terminal

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/UserExistsError/conpty"
)

// windowsPTY wraps ConPTY for Windows platforms.
type windowsPTY struct {
	cpty *conpty.ConPty
	cols int
	rows int

	readCh    chan readResult
	closedCh  chan struct{}
	closeOnce sync.Once

	readDeadline time.Time
	deadlineMu   sync.Mutex
}

type readResult struct {
	data []byte
	n    int
	err  error
}

func startPTY(command string, args []string, workDir string, env []string, cols, rows int) (ptyProcess, error) {
	path, err := exec.LookPath(command)
	if err != nil {
		return nil, fmt.Errorf("command not found: %w", err)
	}

	cmdLine := buildCommandLine(path, args)

	opts := []conpty.ConPtyOption{
		conpty.ConPtyDimensions(cols, rows),
	}
	if workDir != "" {
		opts = append(opts, conpty.ConPtyWorkDir(workDir))
	}
	if len(env) > 0 {
		opts = append(opts, conpty.ConPtyEnv(env))
	}

	cpty, err := conpty.Start(cmdLine, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to start conpty: %w", err)
	}

	wp := &windowsPTY{
		cpty:     cpty,
		cols:     cols,
		rows:     rows,
		readCh:   make(chan readResult, 4),
		closedCh: make(chan struct{}),
	}

	go wp.backgroundReader()

	return wp, nil
}

func buildCommandLine(path string, args []string) string {
	parts := make([]string, 0, 1+len(args))
	parts = append(parts, quoteWinArg(path))
	for _, a := range args {
		parts = append(parts, quoteWinArg(a))
	}
	return strings.Join(parts, " ")
}

// quoteWinArg quotes a Windows command line argument if needed.
func quoteWinArg(s string) string {
	if s == "" {
		return `""`
	}
	needsQuote := false
	for _, c := range s {
		if c == ' ' || c == '\t' || c == '"' {
			needsQuote = true
			break
		}
	}
	if !needsQuote {
		return s
	}
	var buf strings.Builder
	buf.WriteByte('"')
	nSlash := 0
	for _, c := range s {
		switch {
		case c == '\\':
			nSlash++
		case c == '"':
			for i := 0; i < nSlash; i++ {
				buf.WriteByte('\\')
			}
			buf.WriteByte('\\')
			buf.WriteByte('"')
			nSlash = 0
		default:
			for i := 0; i < nSlash; i++ {
				buf.WriteByte('\\')
			}
			buf.WriteRune(c)
			nSlash = 0
		}
	}
	for i := 0; i < nSlash; i++ {
		buf.WriteByte('\\')
	}
	buf.WriteByte('"')
	return buf.String()
}

func (w *windowsPTY) backgroundReader() {
	buf := make([]byte, 4096)
	for {
		n, err := w.cpty.Read(buf)
		var result readResult
		if n > 0 {
			result.data = make([]byte, n)
			copy(result.data, buf[:n])
			result.n = n
		}
		if err != nil {
			result.err = err
		}
		select {
		case w.readCh <- result:
		case <-w.closedCh:
			return
		}
		if err != nil {
			return
		}
	}
}

func (w *windowsPTY) Read(p []byte) (int, error) {
	w.deadlineMu.Lock()
	deadline := w.readDeadline
	w.deadlineMu.Unlock()

	var timer *time.Timer
	var timerCh <-chan time.Time

	if !deadline.IsZero() {
		d := time.Until(deadline)
		if d <= 0 {
			return 0, os.ErrDeadlineExceeded
		}
		timer = time.NewTimer(d)
		defer timer.Stop()
		timerCh = timer.C
	}

	select {
	case r, ok := <-w.readCh:
		if !ok {
			return 0, io.EOF
		}
		if r.n > 0 {
			n := copy(p, r.data[:r.n])
			return n, r.err
		}
		return 0, r.err
	case <-timerCh:
		return 0, os.ErrDeadlineExceeded
	case <-w.closedCh:
		return 0, io.EOF
	}
}

func (w *windowsPTY) Write(data []byte) (int, error) {
	return w.cpty.Write(data)
}

func (w *windowsPTY) Close() error {
	var err error
	w.closeOnce.Do(func() {
		close(w.closedCh)
		err = w.cpty.Close()
	})
	return err
}

func (w *windowsPTY) Resize(cols, rows int) error {
	w.cols = cols
	w.rows = rows
	return w.cpty.Resize(cols, rows)
}

func (w *windowsPTY) GetSize() (int, int, error) {
	return w.cols, w.rows, nil
}

func (w *windowsPTY) Pid() int {
	return w.cpty.Pid()
}

func (w *windowsPTY) SetReadDeadline(t time.Time) error {
	w.deadlineMu.Lock()
	w.readDeadline = t
	w.deadlineMu.Unlock()
	return nil
}

func (w *windowsPTY) Wait() (int, error) {
	exitCode, err := w.cpty.Wait(context.Background())
	return int(exitCode), err
}

func (w *windowsPTY) Kill() error {
	return w.Close()
}

func (w *windowsPTY) GracefulStop() error {
	_, err := w.cpty.Write([]byte{0x03})
	return err
}
