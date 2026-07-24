package runner

import "github.com/anthropics/agentsmesh/runner/internal/terminal/vt"

// PodRuntime is one complete, immutable mode-specific runtime generation.
// Build paths assemble it privately and publish it to Pod in one transition.
type PodRuntime struct {
	IO         PodIO
	Relay      PodRelay
	vtProvider func() *vt.VirtualTerminal
}

func (r PodRuntime) valid() bool {
	return r.IO != nil && r.Relay != nil
}
