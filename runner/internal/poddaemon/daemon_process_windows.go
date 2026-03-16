//go:build windows

package poddaemon

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/UserExistsError/conpty"

	"github.com/anthropics/agentsmesh/runner/internal/process"
)

// windowsDaemonProcess wraps ConPTY for Windows inside the daemon.
type windowsDaemonProcess struct {
	cpty *conpty.ConPty
}

// startDaemonProcess creates a new ConPTY process inside the daemon.
func startDaemonProcess(command string, args []string, workDir string, env []string, cols, rows int) (daemonProcess, error) {
	path, err := exec.LookPath(command)
	if err != nil {
		return nil, fmt.Errorf("command not found: %w", err)
	}

	cmdLine := buildWindowsCmdLine(path, args)

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
		return nil, fmt.Errorf("start conpty: %w", err)
	}
	return &windowsDaemonProcess{cpty: cpty}, nil
}

func (p *windowsDaemonProcess) Read(buf []byte) (int, error) {
	return p.cpty.Read(buf)
}

func (p *windowsDaemonProcess) Write(data []byte) (int, error) {
	return p.cpty.Write(data)
}

func (p *windowsDaemonProcess) Close() error {
	return p.cpty.Close()
}

func (p *windowsDaemonProcess) Resize(cols, rows int) error {
	return p.cpty.Resize(cols, rows)
}

func (p *windowsDaemonProcess) Pid() int {
	return p.cpty.Pid()
}

func (p *windowsDaemonProcess) Wait() (int, error) {
	exitCode, err := p.cpty.Wait(context.Background())
	return int(exitCode), err
}

func (p *windowsDaemonProcess) GracefulStop() error {
	_, err := p.cpty.Write([]byte{0x03}) // Ctrl+C
	return err
}

func (p *windowsDaemonProcess) Kill() error {
	_ = process.KillProcessTree(p.cpty.Pid())
	return p.cpty.Close()
}
