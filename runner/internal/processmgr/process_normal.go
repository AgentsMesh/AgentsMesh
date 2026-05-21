package processmgr

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/anthropics/agentsmesh/runner/internal/safego"
)

// normalProcess is a cmd-backed handle without PTY. Stdio pipes are attached
// when Spec.PipeStdin/Stdout/Stderr is set.
type normalProcess struct {
	*cmdProcess
	stdinW  io.WriteCloser
	stdoutR io.ReadCloser
	stderrR io.ReadCloser
}

var _ Handle = (*normalProcess)(nil)

func startNormal(ctx context.Context, mgr *manager, spec Spec) (Handle, error) {
	// CommandContext binds ctx cancellation to a SIGKILL on the child. This
	// preserves the implicit cleanup callers got from
	// exec.CommandContext(ctx, ...) before they migrated onto processmgr —
	// without it, ctx.Done would silently leave the child running and the
	// only way to terminate would be an explicit Stop call.
	cmd := exec.CommandContext(ctx, spec.Command, spec.Args...) //nolint:gosec // command provided by trusted caller
	cmd.Env = spec.Env
	cmd.Dir = spec.Dir
	cmd.Stdin = spec.Stdin
	cmd.Stdout = spec.Stdout
	cmd.Stderr = spec.Stderr
	applyNewProcessGroup(cmd)

	stdinW, stdoutR, stderrR, err := attachPipes(cmd, spec)
	if err != nil {
		return nil, fmt.Errorf("processmgr: pipe %s: %w", spec.Owner, err)
	}

	if err := cmd.Start(); err != nil {
		closeAll(stdinW, stdoutR, stderrR)
		return nil, fmt.Errorf("processmgr: start %s: %w", spec.Owner, err)
	}

	p := &normalProcess{
		cmdProcess: &cmdProcess{
			baseProcess: newBaseProcess(spec.Owner, ModeNormal, cmd.Process.Pid),
			mgr:         mgr,
			cmd:         cmd,
			stopTimeout: mgr.opts.stopTimeoutFor(spec),
		},
		stdinW:  stdinW,
		stdoutR: stdoutR,
		stderrR: stderrR,
	}
	safego.Go("processmgr-reap-"+spec.Owner, func() { p.reapLoopBody(p, nil) })
	return p, nil
}

func (p *normalProcess) PTY() *os.File                { return nil }
func (p *normalProcess) StdinWriter() io.WriteCloser  { return p.stdinW }
func (p *normalProcess) StdoutReader() io.ReadCloser  { return p.stdoutR }
func (p *normalProcess) StderrReader() io.ReadCloser  { return p.stderrR }

// attachPipes wires up the io.Pipe pairs requested by Spec.PipeStdin/Stdout/
// Stderr. Returns the caller-side ends or nil for fields the caller did not
// request; cmd retains the child-side end via cmd.Stdin/etc.
func attachPipes(cmd *exec.Cmd, spec Spec) (io.WriteCloser, io.ReadCloser, io.ReadCloser, error) {
	var (
		stdinW  io.WriteCloser
		stdoutR io.ReadCloser
		stderrR io.ReadCloser
		err     error
	)
	if spec.PipeStdin {
		stdinW, err = cmd.StdinPipe()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("stdin pipe: %w", err)
		}
	}
	if spec.PipeStdout {
		stdoutR, err = cmd.StdoutPipe()
		if err != nil {
			closeAll(stdinW, nil, nil)
			return nil, nil, nil, fmt.Errorf("stdout pipe: %w", err)
		}
	}
	if spec.PipeStderr {
		stderrR, err = cmd.StderrPipe()
		if err != nil {
			closeAll(stdinW, stdoutR, nil)
			return nil, nil, nil, fmt.Errorf("stderr pipe: %w", err)
		}
	}
	return stdinW, stdoutR, stderrR, nil
}

func closeAll(closers ...io.Closer) {
	for _, c := range closers {
		if c != nil {
			_ = c.Close()
		}
	}
}
