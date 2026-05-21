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
//
// Note on processmgr: terminal/* deliberately does not go through
// processmgr.ModePTY because the Windows PTY path uses ConPTY (not creack/pty),
// and ConPTY's read/write surface is not an *os.File — incompatible with the
// processmgr.Handle.PTY() return type. The Wait() call below is already the
// single ownership point, and Setpgid below ensures kills reach grandchildren,
// so the zombie-leak class that processmgr fixes does not apply here.
type unixPTY struct {
	cmd     *exec.Cmd
	ptyFile *os.File
}

// startPTY creates and starts a PTY process on Unix.
func startPTY(command string, args []string, workDir string, env []string, cols, rows int) (ptyProcess, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = workDir
	cmd.Env = env
	cmd.SysProcAttr = &syscall.SysProcAttr{
		// pty.StartWithSize sets Setsid for us, which already makes this
		// process its own session and group leader. We keep this field
		// reachable so future changes that drop Setsid still get a process
		// group. Setting both is harmless.
		Setsid: true,
	}

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
	if p.cmd.Process == nil {
		return nil
	}
	// Negative PID targets the whole process group so shells that fork their
	// own children (claude, aider, etc.) get cleaned up too.
	if err := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL); err == nil {
		return nil
	}
	return p.cmd.Process.Kill()
}

func (p *unixPTY) GracefulStop() error {
	if p.cmd.Process == nil {
		return fmt.Errorf("process not started")
	}
	if err := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGTERM); err == nil {
		return nil
	}
	return p.cmd.Process.Signal(syscall.SIGTERM)
}
