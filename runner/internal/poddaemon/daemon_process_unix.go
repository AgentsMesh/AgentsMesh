//go:build !windows

package poddaemon

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"github.com/creack/pty"
)

// unixDaemonProcess wraps a PTY-attached process on Unix.
type unixDaemonProcess struct {
	cmd     *exec.Cmd
	ptyFile *os.File
}

// startDaemonProcess creates a new PTY process inside the daemon.
//
// NOTE: We intentionally do NOT set Setpgid here. The daemon process itself
// is already a session leader (started with Setsid by startDaemon), so its
// children inherit the session. Setting Setpgid inside a Setsid session
// triggers "operation not permitted" in Docker containers with restricted
// seccomp profiles. Kill() falls back to direct process kill + PTY close,
// which sends SIGHUP to the child's process group.
func startDaemonProcess(command string, args []string, workDir string, env []string, cols, rows int) (daemonProcess, error) {
	cmd := exec.Command(command, args...)
	cmd.Dir = workDir
	cmd.Env = env

	winSize := &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	}
	ptmx, err := pty.StartWithSize(cmd, winSize)
	if err != nil {
		return nil, fmt.Errorf("start pty: %w", err)
	}
	return &unixDaemonProcess{cmd: cmd, ptyFile: ptmx}, nil
}

func (p *unixDaemonProcess) Read(buf []byte) (int, error) {
	return p.ptyFile.Read(buf)
}

func (p *unixDaemonProcess) Write(data []byte) (int, error) {
	return p.ptyFile.Write(data)
}

func (p *unixDaemonProcess) Close() error {
	return p.ptyFile.Close()
}

func (p *unixDaemonProcess) Resize(cols, rows int) error {
	return pty.Setsize(p.ptyFile, &pty.Winsize{
		Rows: uint16(rows),
		Cols: uint16(cols),
	})
}

func (p *unixDaemonProcess) Pid() int {
	if p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return 0
}

func (p *unixDaemonProcess) Wait() (int, error) {
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

func (p *unixDaemonProcess) GracefulStop() error {
	if p.cmd.Process != nil {
		return p.cmd.Process.Signal(syscall.SIGTERM)
	}
	return fmt.Errorf("process not started")
}

func (p *unixDaemonProcess) Kill() error {
	if p.cmd.Process != nil {
		// Try process-group kill first (works when child happens to be a PGID leader).
		// Falls back to killing the direct process. Closing the PTY fd afterwards
		// sends SIGHUP to any remaining children attached to the slave side.
		if err := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL); err != nil {
			return p.cmd.Process.Kill()
		}
		return nil
	}
	return nil
}
