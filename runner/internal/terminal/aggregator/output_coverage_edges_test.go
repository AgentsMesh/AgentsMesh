package aggregator

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type snapshotTimeoutWriter struct {
	flushStarted chan struct{}
	releaseFlush chan struct{}
	startOnce    sync.Once
}

func newSnapshotTimeoutWriter() *snapshotTimeoutWriter {
	return &snapshotTimeoutWriter{
		flushStarted: make(chan struct{}),
		releaseFlush: make(chan struct{}),
	}
}

func (w *snapshotTimeoutWriter) SendOutput([]byte) error { return nil }

func (w *snapshotTimeoutWriter) FlushOutput(context.Context) error {
	w.startOnce.Do(func() { close(w.flushStarted) })
	<-w.releaseFlush
	return nil
}

func (w *snapshotTimeoutWriter) IsConnected() bool { return true }

func bareOutputLane(capacity int) *outputLane {
	return &outputLane{
		name: "test", byteBudget: 1024, queue: make(chan queuedOutput, capacity),
		stopCh: make(chan struct{}), writer: newMockRelayWriter(true), generation: 1,
	}
}

func TestOutputSnapshotBoundaryDefensivePaths(t *testing.T) {
	var boundary *OutputSnapshotBoundary
	if boundary.wait(time.Second) || boundary.RecoveryRequired() || boundary.Commit() {
		t.Fatal("nil snapshot boundary reported success")
	}
	boundary.Abort(errors.New("ignored"))

	lane := bareOutputLane(1)
	invalid := &OutputSnapshotBoundary{
		lanes:   []snapshotLaneBoundary{{lane: lane, generation: 1}},
		barrier: &outputBarrier{},
	}
	if invalid.Commit() {
		t.Fatal("snapshot without pending lane committed")
	}

	if !(&outputBarrier{}).wait(time.Second) {
		t.Fatal("empty output barrier did not complete")
	}
	if (&outputBarrier{results: []*outputBarrierResult{newOutputBarrierResult()}}).wait(0) {
		t.Fatal("zero-timeout output barrier completed")
	}
	if (&outputBarrier{results: []*outputBarrierResult{newOutputBarrierResult()}}).wait(time.Millisecond) {
		t.Fatal("uncompleted output barrier beat timeout")
	}
}

func TestOutputLaneAdmissionDefensivePaths(t *testing.T) {
	lane := bareOutputLane(1)
	if !lane.enqueue(nil) {
		t.Fatal("empty output was rejected")
	}
	lane.stopped = true
	if lane.enqueue([]byte("stopped")) {
		t.Fatal("stopped lane accepted output")
	}
	result := lane.enqueueBarrier()
	if ok := <-result.done; ok {
		t.Fatal("stopped lane accepted barrier")
	}

	full := bareOutputLane(1)
	full.queue <- queuedOutput{generation: 1}
	if full.enqueue([]byte("frame")) {
		t.Fatal("physically full lane accepted output")
	}
	if !full.desynced || full.reason != OutputDesyncOverflow {
		t.Fatalf("full lane health = %+v", full.health())
	}

	barrierFull := bareOutputLane(1)
	barrierFull.queue <- queuedOutput{generation: 1}
	barrierResult := barrierFull.enqueueBarrier()
	if ok := <-barrierResult.done; ok || !barrierFull.desynced {
		t.Fatal("full lane barrier did not fail and desynchronize")
	}
}

func TestOutputLaneDeliveryAndCleanupEdges(t *testing.T) {
	lane := bareOutputLane(4)
	stale := newOutputBarrierResult()
	lane.deliver(queuedOutput{generation: 0, barrier: stale})
	if ok := <-stale.done; !ok {
		t.Fatal("stale barrier should be treated as drained")
	}

	lane.desynced = true
	rejected := newOutputBarrierResult()
	lane.deliver(queuedOutput{generation: 1, barrier: rejected})
	if ok := <-rejected.done; ok {
		t.Fatal("desynchronized lane completed a current barrier")
	}

	lane.queuedBytes = 1
	lane.releaseBytes(2)
	if lane.queuedBytes != 0 {
		t.Fatalf("negative queued bytes were not clamped: %d", lane.queuedBytes)
	}
	lane.stop()
	lane.stop()

	queued := bareOutputLane(4)
	queued.queuedBytes = len("data")
	queued.queue <- queuedOutput{data: []byte("data")}
	barrier := newOutputBarrierResult()
	queued.queue <- queuedOutput{barrier: barrier}
	queued.failQueuedBarriers()
	if queued.queuedBytes != 0 {
		t.Fatalf("queued bytes leaked after failure: %d", queued.queuedBytes)
	}
	if ok := <-barrier.done; ok {
		t.Fatal("queued barrier succeeded during failure cleanup")
	}
}

func TestOutputLaneStateDefensivePaths(t *testing.T) {
	lane := bareOutputLane(1)
	lane.markDesynced(0, OutputDesyncSendError, errors.New("stale"))
	if lane.desynced {
		t.Fatal("stale generation desynchronized lane")
	}
	lane.stopped = true
	lane.forceDesync(OutputDesyncSendError, errors.New("stopped"))
	if lane.desynced {
		t.Fatal("stopped lane was force-desynchronized")
	}
	result, generation, recovery := lane.beginSnapshot(2)
	if generation != 0 || recovery {
		t.Fatalf("stopped snapshot = generation %d recovery %v", generation, recovery)
	}
	if ok := <-result.done; ok {
		t.Fatal("stopped snapshot barrier succeeded")
	}
	if lane.commitSnapshot(1) {
		t.Fatal("stopped snapshot committed")
	}
	lane.abortSnapshot(1, errors.New("ignored"))

	full := bareOutputLane(1)
	full.queue <- queuedOutput{generation: 1}
	result, generation, _ = full.beginSnapshot(2)
	if generation != 1 || !full.desynced {
		t.Fatalf("full snapshot lane state: generation=%d health=%+v", generation, full.health())
	}
	if ok := <-result.done; ok {
		t.Fatal("full snapshot barrier succeeded")
	}

	callback := make(chan OutputHealth, 1)
	notify := bareOutputLane(1)
	notify.onDesync = func(health OutputHealth) { callback <- health }
	health, changed := notify.desyncLocked(OutputDesyncBuffer, errors.New("buffer"))
	notify.notifyDesync(health, changed)
	select {
	case got := <-callback:
		if got.Reason != OutputDesyncBuffer {
			t.Fatalf("callback health = %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("desync callback did not run")
	}
	if _, changed := notify.desyncLocked(OutputDesyncSendError, nil); changed {
		t.Fatal("second desync reported a transition")
	}
}

func TestOutputRouterAndSmartAggregatorGuardPaths(t *testing.T) {
	router := newOutputRouter(0, 0)
	if router.byteBudget != defaultOutputQueueBytes || router.itemLimit != defaultOutputQueueItems {
		t.Fatalf("router defaults = bytes %d items %d", router.byteBudget, router.itemLimit)
	}
	if boundary, found := router.prepareSnapshot("missing"); found || boundary != nil {
		t.Fatal("missing destination prepared a snapshot")
	}
	router.SetDestination("", newMockRelayWriter(true))
	router.SetDestination("invalid", nil)

	agg := NewSmartAggregator(func() float64 { return 0 })
	agg.Stop()
	if agg.FlushAndWait(time.Second) {
		t.Fatal("stopped aggregator flushed")
	}
	if boundary, ok := agg.PrepareSnapshot("missing", time.Second); ok || boundary != nil {
		t.Fatal("stopped aggregator prepared snapshot")
	}
	agg.SetOutputDestination("cloud", newMockRelayWriter(true))
	agg.RemoveOutputDestination("cloud")

	done := make(chan struct{})
	close(done)
	if !waitForOutputDrain(done, 0) {
		t.Fatal("closed output drain was not observed")
	}
	if waitForOutputDrain(make(chan struct{}), time.Millisecond) {
		t.Fatal("open output drain beat timeout")
	}
}

func TestSmartAggregatorSnapshotBoundaryPublicContract(t *testing.T) {
	agg := NewSmartAggregator(func() float64 { return 0 })
	agg.SetOutputDestination("z-local", newMockRelayWriter(true))
	agg.SetOutputDestination("a-cloud", newMockRelayWriter(true))
	defer agg.Stop()

	health := agg.OutputHealth()
	if len(health) != 2 || health[0].Destination != "a-cloud" || health[1].Destination != "z-local" {
		t.Fatalf("OutputHealth is not destination sorted: %+v", health)
	}
	if !agg.FlushAndWait(0) {
		t.Fatal("default drain timeout did not flush healthy destinations")
	}
	if boundary, ok := agg.PrepareSnapshot("missing", time.Second); ok || boundary != nil {
		t.Fatal("missing destination prepared a snapshot")
	}

	boundary, ok := agg.PrepareSnapshot("a-cloud", 0)
	if !ok || boundary == nil {
		t.Fatal("healthy destination did not prepare a snapshot with the default timeout")
	}
	if boundary.RecoveryRequired() {
		t.Fatal("healthy destination unexpectedly required recovery")
	}
	if !boundary.Commit() {
		t.Fatal("healthy snapshot boundary did not commit")
	}
	agg.RemoveOutputDestination("z-local")
	if got := agg.OutputHealth(); len(got) != 1 || got[0].Destination != "a-cloud" {
		t.Fatalf("destination removal left unexpected health: %+v", got)
	}
}

func TestSmartAggregatorSnapshotBarrierTimeoutDesynchronizesDestination(t *testing.T) {
	writer := newSnapshotTimeoutWriter()
	agg := NewSmartAggregator(func() float64 { return 0 })
	agg.SetOutputDestination(OutputDestinationCloud, writer)
	defer func() {
		close(writer.releaseFlush)
		agg.Stop()
	}()

	boundary, ok := agg.PrepareSnapshot(OutputDestinationCloud, time.Millisecond)
	if ok || boundary != nil {
		t.Fatal("blocked downstream flush crossed the snapshot timeout")
	}
	select {
	case <-writer.flushStarted:
	case <-time.After(time.Second):
		t.Fatal("snapshot barrier never reached the downstream flush")
	}
	health := agg.OutputHealth()
	if len(health) != 1 || !health[0].Desynced || health[0].SnapshotPending {
		t.Fatalf("snapshot timeout did not leave a recoverable destination state: %+v", health)
	}
	if health[0].Reason != OutputDesyncSnapshot || !errors.Is(health[0].Err, errSnapshotBarrierTimeout) {
		t.Fatalf("snapshot timeout health = %+v", health[0])
	}
}
