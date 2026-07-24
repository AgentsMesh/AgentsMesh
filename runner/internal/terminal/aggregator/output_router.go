// Package aggregator provides terminal output aggregation and delivery.
package aggregator

import (
	"sync"
)

// OutputRouter owns one bounded FIFO lane per destination. Replacing a writer
// keeps the destination lane, so an in-flight old-generation send cannot be
// overtaken by newer output for the same destination. Different destinations
// never share a worker.
type OutputRouter struct {
	mu           sync.RWMutex
	destinations map[string]*outputLane
	generation   uint64
	byteBudget   int
	itemLimit    int

	healthMu      sync.RWMutex
	healthHandler OutputHealthHandler
}

// NewOutputRouter creates an output router with a 1 MiB budget per destination.
func NewOutputRouter() *OutputRouter {
	return newOutputRouter(defaultOutputQueueBytes, defaultOutputQueueItems)
}

func newOutputRouter(byteBudget, itemLimit int) *OutputRouter {
	if byteBudget <= 0 {
		byteBudget = defaultOutputQueueBytes
	}
	if itemLimit <= 0 {
		itemLimit = defaultOutputQueueItems
	}
	return &OutputRouter{
		destinations: make(map[string]*outputLane),
		byteBudget:   byteBudget,
		itemLimit:    itemLimit,
	}
}

func (r *OutputRouter) enqueue(data []byte) bool {
	if len(data) == 0 {
		return true
	}
	lanes := r.snapshotLanes()
	ok := true
	for _, lane := range lanes {
		if !lane.enqueue(data) {
			ok = false
		}
	}
	return ok
}

// barrier completes after all data accepted before it has either finished its
// destination send or been rejected as stale/desynchronized.
func (r *OutputRouter) barrier() *outputBarrier {
	lanes := r.snapshotLanes()
	barrier := &outputBarrier{results: make([]*outputBarrierResult, 0, len(lanes))}
	for _, lane := range lanes {
		barrier.results = append(barrier.results, lane.enqueueBarrier())
	}
	return barrier
}

func (r *OutputRouter) prepareSnapshot(destination string) (*OutputSnapshotBoundary, bool) {
	r.mu.Lock()
	lane := r.destinations[destination]
	if lane == nil {
		r.mu.Unlock()
		return nil, false
	}
	r.generation++
	resetGeneration := r.generation
	boundary := &OutputSnapshotBoundary{
		lanes:   make([]snapshotLaneBoundary, 0, 1),
		barrier: &outputBarrier{results: make([]*outputBarrierResult, 0, 1)},
	}
	result, generation, recoveryRequired := lane.beginSnapshot(resetGeneration)
	boundary.recoveryRequired = recoveryRequired
	boundary.lanes = append(boundary.lanes, snapshotLaneBoundary{
		lane: lane, generation: generation,
	})
	boundary.barrier.results = append(boundary.barrier.results, result)
	r.mu.Unlock()
	return boundary, true
}

func (r *OutputRouter) snapshotLanes() []*outputLane {
	r.mu.RLock()
	defer r.mu.RUnlock()
	lanes := make([]*outputLane, 0, len(r.destinations))
	for _, lane := range r.destinations {
		lanes = append(lanes, lane)
	}
	return lanes
}

// SetDestination replaces one logical sink without disturbing other sinks.
func (r *OutputRouter) SetDestination(name string, writer RelayWriter) {
	if name == "" || writer == nil {
		return
	}
	r.mu.Lock()
	r.generation++
	generation := r.generation
	if lane := r.destinations[name]; lane != nil {
		lane.replaceWriter(writer, generation)
		r.mu.Unlock()
		return
	}
	r.destinations[name] = newOutputLane(name, writer, generation, r.byteBudget, r.itemLimit, r.emitDesync)
	r.mu.Unlock()
}

// RemoveDestination stops one sink while all other sinks continue independently.
func (r *OutputRouter) RemoveDestination(name string) {
	r.mu.Lock()
	removed := r.destinations[name]
	delete(r.destinations, name)
	r.mu.Unlock()
	if removed != nil {
		removed.stop()
	}
}

func (r *OutputRouter) stop() {
	r.mu.Lock()
	lanes := r.destinations
	r.destinations = make(map[string]*outputLane)
	r.mu.Unlock()
	for _, lane := range lanes {
		lane.stop()
	}
}
