package aggregator

import (
	"fmt"
	"sort"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// SetHealthHandler installs the observable desynchronization callback.
func (r *OutputRouter) SetHealthHandler(handler OutputHealthHandler) {
	r.healthMu.Lock()
	r.healthHandler = handler
	r.healthMu.Unlock()
}

// Health returns a stable, destination-sorted state snapshot.
func (r *OutputRouter) Health() []OutputHealth {
	lanes := r.snapshotLanes()
	health := make([]OutputHealth, 0, len(lanes))
	for _, lane := range lanes {
		health = append(health, lane.health())
	}
	sort.Slice(health, func(i, j int) bool {
		return health[i].Destination < health[j].Destination
	})
	return health
}

func (r *OutputRouter) desynchronizeAll(reason OutputDesyncReason, err error) {
	for _, lane := range r.snapshotLanes() {
		lane.forceDesync(reason, err)
	}
}

func (r *OutputRouter) emitDesync(health OutputHealth) {
	attrs := []any{
		"destination", health.Destination,
		"reason", health.Reason,
		"queued_bytes", health.QueuedBytes,
		"byte_budget", health.ByteBudget,
	}
	if health.Err != nil {
		attrs = append(attrs, "error", health.Err)
	}
	logger.Terminal().Error("PTY output destination desynchronized", attrs...)

	r.healthMu.RLock()
	handler := r.healthHandler
	r.healthMu.RUnlock()
	if handler != nil {
		callOutputHealthHandler(handler, health)
	}
}

func callOutputHealthHandler(handler OutputHealthHandler, health OutputHealth) {
	defer logOutputCallbackPanic("health")
	handler(health)
}

func logOutputCallbackPanic(kind string) {
	if recovered := recover(); recovered != nil {
		logger.Terminal().Error("PTY output callback panicked", "kind", kind, "panic", fmt.Sprint(recovered))
	}
}
