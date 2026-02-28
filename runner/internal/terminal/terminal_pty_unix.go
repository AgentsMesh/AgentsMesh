//go:build !windows

package terminal

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/creack/pty"
)

// unixPTY wraps creack/pty and exec.Cmd for Unix platforms.
type unixPTY struct {
	cmd     *exec.Cmd
	ptyFile *os.File
}

// startPTY creates and starts a PTY process on Unix.
func startPTY(command string, args []string, workDir string, env []string, cols, rows int) (ptyProcess, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = workDir
	cmd.Env = env

	winSize := &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	}
	ptmx, err := pty.StartWithSize(cmd, winSize)
	if err != nil {
		return nil, err
	}

	return &unixPTY{cmd: cmd, ptyFile: ptmx}, nil
}

func (p *unixPTY) Read(buf []byte) (int, error) {
	return p.ptyFile.Read(buf)
}

func (p *unixPTY) Write(data []byte) (int, error) {
	return p.ptyFile.Write(data)
}

func (p *unixPTY) Close() error {
	return p.ptyFile.Close()
}

func (p *unixPTY) Resize(cols, rows int) error {
	return pty.Setsize(p.ptyFile, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
}

func (p *unixPTY) GetSize() (int, int, error) {
	size, err := pty.GetsizeFull(p.ptyFile)
	if err != nil {
		return 0, 0, err
	}
	return int(size.Cols), int(size.Rows), nil
}

func (p *unixPTY) Pid() int {
	if p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return 0
}

func (p *unixPTY) SetReadDeadline(t time.Time) error {
	return p.ptyFile.SetReadDeadline(t)
}

func (p *unixPTY) Wait() (int, error) {
	err := p.cmd.Wait()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode(), nil
		}
		return -1, err
	}
	return 0, nil
}

func (p *unixPTY) Kill() error {
	if p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return nil
}

func (p *unixPTY) GracefulStop() error {
	if p.cmd.Process != nil {
		return p.cmd.Process.Signal(syscall.SIGTERM)
	}
	return fmt.Errorf("process not started")
}
