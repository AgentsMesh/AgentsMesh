package runner

import (
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/terminal"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

// PTYComponents holds PTY-specific infrastructure shared between PTYPodIO and PTYPodRelay.
// Both abstractions receive a pointer to the same instance, eliminating field duplication.
type PTYComponents struct {
	Terminal        *terminal.Terminal
	VirtualTerminal *vt.VirtualTerminal
	Aggregator      *aggregator.SmartAggregator
	PTYLogger       *aggregator.PTYLogger

	// readMu is acquired before a PTY read and held through its output handler.
	// Geometry and teardown acquire it first, so bytes already owned by the read
	// loop cannot cross either lifecycle boundary. Lock order is readMu, streamMu.
	readMu sync.Mutex

	// streamMu defines the ordering boundary shared by processed PTY output,
	// snapshots, and geometry changes. Snapshot delivery may hold it for one
	// bounded sink barrier so pre-cut deltas cannot arrive after the baseline.
	streamMu sync.Mutex
}
