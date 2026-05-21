//go:build !windows

package processmgr

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/anthropics/agentsmesh/runner/internal/safego"
	"github.com/creack/pty"
)

// ptyProcess is a cmd-backed handle that exposes a PTY master fd to the
// caller. Lifecycle is identical to normalProcess; the PTY fd is closed in
// the onWait hook just before doneCh fires so that callers reading from
// PTY() are not racing the kernel reaping.
type ptyProcess struct {
	*cmdProcess
	ptyFile *os.File
}

var _ Handle = (*ptyProcess)(nil)

func startPTY(ctx context.Context, mgr *manager, spec Spec) (Handle, error) {
	cmd := exec.Command(spec.Command, spec.Args...) //nolint:gosec
	cmd.Env = spec.Env
	cmd.Dir = spec.Dir

	ptyFile, err := pty.StartWithSize(cmd, spec.PTYSize)
	if err != nil {
		return nil, fmt.Errorf("processmgr: pty start %s: %w", spec.Owner, err)
	}

	p := &ptyProcess{
		cmdProcess: &cmdProcess{
			baseProcess: newBaseProcess(spec.Owner, ModePTY, cmd.Process.Pid),
			mgr:         mgr,
			cmd:         cmd,
			stopTimeout: mgr.opts.stopTimeoutFor(spec),
		},
		ptyFile: ptyFile,
	}
	safego.Go("processmgr-reap-pty-"+spec.Owner, func() {
		p.reapLoopBody(p, func() { _ = p.ptyFile.Close() })
	})
	return p, nil
}

func (p *ptyProcess) PTY() *os.File { return p.ptyFile }
