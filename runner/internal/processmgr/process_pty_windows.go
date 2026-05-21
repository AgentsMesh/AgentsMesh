//go:build windows

package processmgr

import (
	"context"
	"errors"
)

// startPTY on Windows is intentionally unimplemented for the initial rollout.
// terminal/terminal_pty_windows.go uses conpty directly (not exec.Cmd), so
// it is not eligible for ModePTY in its current form. When terminal/* moves
// onto processmgr we will wire conpty here and update Handle.PTY accordingly.
func startPTY(ctx context.Context, mgr *manager, spec Spec) (Handle, error) {
	return nil, errors.New("processmgr: ModePTY on Windows is not yet supported")
}
