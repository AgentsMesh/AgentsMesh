package processmgr

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
)

func (m *manager) startReaper() {
	safego.Go("processmgr-reaper", func() { m.reaperLoop(m.ctx) })
}

// reaperLoop is the safety-net wait4(WNOHANG) sweep. Anything it finds is a
// bug somewhere else — runner_zombie_reaped_total should stay at zero in
// steady state. The interval is configured via Options.ReaperInterval.
func (m *manager) reaperLoop(ctx context.Context) {
	t := time.NewTicker(m.opts.ReaperInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if n := reapOrphans(); n > 0 {
				observeZombieReaped(n)
				logger.Runner().Warn("processmgr: reaped untracked zombies",
					"count", n,
					"hint", "some Start() path is not going through processmgr — inspect /debug/processes")
			}
		}
	}
}
