package runner

import (
	"fmt"

	"github.com/anthropics/agentsmesh/runner/internal/poddaemon"
)

// restartDeadPerpetualDaemon re-creates a daemon session for a perpetual pod
// whose daemon died. Uses the existing sandbox and state to spawn a new daemon.
func (r *Runner) restartDeadPerpetualDaemon(state *poddaemon.PodDaemonState) (*Pod, error) {
	_, updatedState, err := r.podDaemonManager.CreateSession(poddaemon.CreateOpts{
		PodKey:         state.PodKey,
		Agent:          state.Agent,
		Command:        state.Command,
		Args:           state.Args,
		WorkDir:        state.WorkDir,
		Env:            state.Env,
		Cols:           state.Cols,
		Rows:           state.Rows,
		SandboxPath:    state.SandboxPath,
		RepositoryURL:  state.RepositoryURL,
		Branch:         state.Branch,
		TicketSlug:     state.TicketSlug,
		VTHistoryLimit: state.VTHistoryLimit,
		Perpetual:      true,
	})
	if err != nil {
		return nil, fmt.Errorf("create daemon session: %w", err)
	}

	// recoverSingleSession will AttachSession (new TCP conn). The CreateSession
	// connection is implicitly replaced by daemon's single-client model.
	return r.recoverSingleSession(updatedState)
}
