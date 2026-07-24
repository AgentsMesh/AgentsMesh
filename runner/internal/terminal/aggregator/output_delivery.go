package aggregator

import (
	"context"
	"errors"
	"sync"
	"time"
)

const (
	defaultOutputQueueBytes = 1 << 20
	defaultOutputQueueItems = 1024
	OutputDestinationCloud  = "cloud"
	OutputDestinationLocal  = "local"
)

// RelayWriter is one ordered PTY output destination.
type RelayWriter interface {
	SendOutput(data []byte) error
	FlushOutput(ctx context.Context) error
	IsConnected() bool
}

// OutputDesyncReason explains why a destination can no longer consume deltas.
type OutputDesyncReason string

const (
	OutputDesyncOverflow     OutputDesyncReason = "overflow"
	OutputDesyncSendError    OutputDesyncReason = "send_error"
	OutputDesyncFlushError   OutputDesyncReason = "flush_error"
	OutputDesyncDisconnected OutputDesyncReason = "disconnected"
	OutputDesyncSnapshot     OutputDesyncReason = "snapshot_failed"
	OutputDesyncBuffer       OutputDesyncReason = "aggregation_buffer_overflow"
)

var errSnapshotBarrierTimeout = errors.New("PTY output snapshot barrier timed out")

// OutputHealth is a point-in-time destination delivery state.
type OutputHealth struct {
	Destination     string
	Desynced        bool
	SnapshotPending bool
	Reason          OutputDesyncReason
	Err             error
	QueuedBytes     int
	ByteBudget      int
}

// OutputHealthHandler receives transitions into a desynchronized state.
// The handler must arrange a destination snapshot before marking it resynced.
type OutputHealthHandler func(OutputHealth)

type outputBarrierResult struct {
	done   chan bool
	ctx    context.Context
	cancel context.CancelFunc
	once   sync.Once
}

func newOutputBarrierResult() *outputBarrierResult {
	ctx, cancel := context.WithCancel(context.Background())
	return &outputBarrierResult{
		done:   make(chan bool, 1),
		ctx:    ctx,
		cancel: cancel,
	}
}

func (r *outputBarrierResult) complete(ok bool) {
	r.once.Do(func() {
		r.done <- ok
		r.cancel()
	})
}

type outputBarrier struct {
	results []*outputBarrierResult
}

type snapshotLaneBoundary struct {
	lane       *outputLane
	generation uint64
}

// OutputSnapshotBoundary is a prepared delivery cut. The caller must send the
// snapshot while holding its stream-ordering lock, then Commit on success or
// Abort on failure. Either terminal operation may be called only once.
type OutputSnapshotBoundary struct {
	once             sync.Once
	lanes            []snapshotLaneBoundary
	barrier          *outputBarrier
	recoveryRequired bool
	committed        bool
}

func (b *OutputSnapshotBoundary) wait(timeout time.Duration) bool {
	return b != nil && b.barrier.wait(timeout)
}

// RecoveryRequired reports whether the lane was desynchronized at the cut.
// A requester-private snapshot cannot commit such a destination-wide recovery.
func (b *OutputSnapshotBoundary) RecoveryRequired() bool {
	return b != nil && b.recoveryRequired
}

// Commit makes the snapshot the new baseline and reopens delta delivery.
func (b *OutputSnapshotBoundary) Commit() bool {
	if b == nil {
		return false
	}
	b.once.Do(func() {
		b.committed = true
		for _, boundary := range b.lanes {
			if !boundary.lane.commitSnapshot(boundary.generation) {
				b.committed = false
			}
		}
	})
	return b.committed
}

// Abort keeps every participating destination desynchronized.
func (b *OutputSnapshotBoundary) Abort(err error) {
	if b == nil {
		return
	}
	b.once.Do(func() {
		for _, boundary := range b.lanes {
			boundary.lane.abortSnapshot(boundary.generation, err)
		}
	})
}

func (b *outputBarrier) wait(timeout time.Duration) bool {
	if len(b.results) == 0 {
		return true
	}
	if timeout <= 0 {
		return false
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	defer func() {
		for _, result := range b.results {
			result.cancel()
		}
	}()
	ok := true
	// A failed lane still participates in the drain cut. Wait for every later
	// lane until the common deadline so teardown cannot race an in-flight send.
	for _, result := range b.results {
		select {
		case resultOK := <-result.done:
			if !resultOK {
				ok = false
			}
		case <-timer.C:
			return false
		}
	}
	return ok
}
