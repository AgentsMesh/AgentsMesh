package aggregator

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type gateRelayWriter struct {
	connected atomic.Bool
	started   chan struct{}
	release   chan struct{}
	startOnce sync.Once
	blockOnce sync.Once

	mu   sync.Mutex
	data []byte
}

type disconnectAfterAcceptWriter struct {
	checks atomic.Int32
}

type flushGateRelayWriter struct {
	flushStarted chan struct{}
	releaseFlush chan struct{}
	startOnce    sync.Once
	data         atomic.Int64
}

func (w *disconnectAfterAcceptWriter) SendOutput([]byte) error { return nil }
func (w *disconnectAfterAcceptWriter) FlushOutput(context.Context) error {
	return nil
}
func (w *disconnectAfterAcceptWriter) IsConnected() bool {
	return w.checks.Add(1) == 1
}

func newFlushGateRelayWriter() *flushGateRelayWriter {
	return &flushGateRelayWriter{
		flushStarted: make(chan struct{}),
		releaseFlush: make(chan struct{}),
	}
}

func (w *flushGateRelayWriter) SendOutput(data []byte) error {
	w.data.Add(int64(len(data)))
	return nil
}

func (w *flushGateRelayWriter) FlushOutput(ctx context.Context) error {
	w.startOnce.Do(func() { close(w.flushStarted) })
	select {
	case <-w.releaseFlush:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *flushGateRelayWriter) IsConnected() bool { return true }

func newGateRelayWriter() *gateRelayWriter {
	w := &gateRelayWriter{started: make(chan struct{}), release: make(chan struct{})}
	w.connected.Store(true)
	return w
}

func (w *gateRelayWriter) SendOutput(data []byte) error {
	w.startOnce.Do(func() { close(w.started) })
	w.blockOnce.Do(func() { <-w.release })
	w.mu.Lock()
	w.data = append(w.data, data...)
	w.mu.Unlock()
	return nil
}

func (w *gateRelayWriter) FlushOutput(context.Context) error { return nil }

func (w *gateRelayWriter) IsConnected() bool { return w.connected.Load() }

func (w *gateRelayWriter) getData() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return string(w.data)
}

func TestOutputRouterDestinationsAreIsolatedAndOrdered(t *testing.T) {
	slow := newGateRelayWriter()
	fast := newMockRelayWriter(true)
	router := newOutputRouter(1024, 16)
	router.SetDestination(OutputDestinationCloud, fast)
	router.SetDestination(OutputDestinationLocal, slow)
	defer router.stop()

	router.enqueue([]byte("one"))
	select {
	case <-slow.started:
	case <-time.After(time.Second):
		t.Fatal("slow destination did not start")
	}
	router.enqueue([]byte("two"))

	waitForData(t, fast, "onetwo")
	if got := slow.getData(); got != "" {
		t.Fatalf("blocked destination unexpectedly completed: %q", got)
	}

	close(slow.release)
	if !router.barrier().wait(time.Second) {
		t.Fatal("router did not drain after blocked destination resumed")
	}
	if got := slow.getData(); got != "onetwo" {
		t.Fatalf("slow destination order = %q, want %q", got, "onetwo")
	}
}

func TestOutputBarrierWaitsForDestinationSocketFlush(t *testing.T) {
	writer := newFlushGateRelayWriter()
	router := newOutputRouter(1024, 16)
	router.SetDestination(OutputDestinationCloud, writer)
	defer router.stop()

	if !router.enqueue([]byte("tail")) {
		t.Fatal("tail output was not accepted")
	}
	waitDone := make(chan bool, 1)
	go func() { waitDone <- router.barrier().wait(time.Second) }()
	select {
	case <-writer.flushStarted:
	case <-time.After(time.Second):
		t.Fatal("barrier did not reach downstream flush")
	}
	if got := writer.data.Load(); got != int64(len("tail")) {
		t.Fatalf("SendOutput bytes=%d, want %d", got, len("tail"))
	}
	select {
	case ok := <-waitDone:
		t.Fatalf("barrier completed before downstream flush: %v", ok)
	case <-time.After(20 * time.Millisecond):
	}

	close(writer.releaseFlush)
	select {
	case ok := <-waitDone:
		if !ok {
			t.Fatal("barrier failed after downstream flush completed")
		}
	case <-time.After(time.Second):
		t.Fatal("barrier did not complete after downstream flush")
	}
}

func TestOutputRouterOverflowStopsDeltasUntilSnapshotCommit(t *testing.T) {
	slow := newGateRelayWriter()
	router := newOutputRouter(8, 8)
	healthEvents := make(chan OutputHealth, 1)
	router.SetHealthHandler(func(health OutputHealth) { healthEvents <- health })
	router.SetDestination(OutputDestinationLocal, slow)
	defer router.stop()

	router.enqueue([]byte("aaaa"))
	select {
	case <-slow.started:
	case <-time.After(time.Second):
		t.Fatal("blocked send did not start")
	}
	router.enqueue([]byte("bbbb"))
	router.enqueue([]byte("x"))

	select {
	case health := <-healthEvents:
		if health.Destination != OutputDestinationLocal || health.Reason != OutputDesyncOverflow {
			t.Fatalf("overflow health = %+v", health)
		}
	case <-time.After(time.Second):
		t.Fatal("overflow did not emit destination health")
	}

	boundary, found := router.prepareSnapshot(OutputDestinationLocal)
	if !found {
		t.Fatal("local destination missing during snapshot preparation")
	}
	close(slow.release)
	if !boundary.wait(time.Second) {
		t.Fatal("snapshot boundary did not discard old-generation queue")
	}
	if err := slow.SendOutput([]byte("SNAP")); err != nil {
		t.Fatalf("snapshot send: %v", err)
	}
	if !boundary.Commit() {
		t.Fatal("snapshot boundary commit failed")
	}

	router.enqueue([]byte("delta"))
	if !router.barrier().wait(time.Second) {
		t.Fatal("post-snapshot delta did not drain")
	}
	if got := slow.getData(); got != "aaaaSNAPdelta" {
		t.Fatalf("destination stream = %q, want old in-flight prefix + snapshot + delta", got)
	}
}

func TestOutputRouterSendErrorDesynchronizesDestination(t *testing.T) {
	relay := newMockRelayWriter(true)
	relay.sendErr = errors.New("write failed")
	router := newOutputRouter(1024, 16)
	router.SetDestination(OutputDestinationCloud, relay)
	defer router.stop()

	router.enqueue([]byte("broken"))
	if router.barrier().wait(time.Second) {
		t.Fatal("barrier succeeded after destination send error")
	}
	health := router.Health()
	if len(health) != 1 || !health[0].Desynced || health[0].Reason != OutputDesyncSendError {
		t.Fatalf("send-error health = %+v", health)
	}
}

func TestOutputBarrierWaitsForHealthyDestinationAfterAnotherFails(t *testing.T) {
	failed := newOutputLane(
		OutputDestinationCloud,
		newMockRelayWriter(true),
		1,
		1024,
		16,
		nil,
	)
	healthyWriter := newGateRelayWriter()
	healthy := newOutputLane(
		OutputDestinationLocal,
		healthyWriter,
		1,
		1024,
		16,
		nil,
	)
	defer failed.stop()
	defer healthy.stop()
	var releaseOnce sync.Once
	releaseHealthy := func() { releaseOnce.Do(func() { close(healthyWriter.release) }) }
	defer releaseHealthy()

	failed.forceDesync(OutputDesyncSendError, errors.New("cloud failed"))
	failedResult := failed.enqueueBarrier()
	if !healthy.enqueue([]byte("healthy")) {
		t.Fatal("healthy destination rejected output")
	}
	select {
	case <-healthyWriter.started:
	case <-time.After(time.Second):
		t.Fatal("healthy destination did not begin its gated send")
	}
	healthyResult := healthy.enqueueBarrier()

	done := make(chan bool, 1)
	go func() {
		done <- (&outputBarrier{results: []*outputBarrierResult{
			failedResult,
			healthyResult,
		}}).wait(time.Second)
	}()

	select {
	case got := <-done:
		t.Fatalf("barrier returned %v before the healthy destination drained", got)
	case <-time.After(50 * time.Millisecond):
	}

	releaseHealthy()
	select {
	case got := <-done:
		if got {
			t.Fatal("barrier succeeded despite the failed destination")
		}
	case <-time.After(time.Second):
		t.Fatal("barrier did not return after the healthy destination drained")
	}
	if got := healthyWriter.getData(); got != "healthy" {
		t.Fatalf("healthy destination data = %q, want %q", got, "healthy")
	}
}

func TestOutputRouterInactiveLocalDestinationDoesNotDesynchronize(t *testing.T) {
	local := newMockRelayWriter(false)
	router := newOutputRouter(1024, 16)
	router.SetDestination(OutputDestinationLocal, local)
	defer router.stop()

	router.enqueue([]byte("not observed"))
	if !router.barrier().wait(time.Second) {
		t.Fatal("inactive local destination did not reach its intentional-drop boundary")
	}
	health := router.Health()
	if len(health) != 1 || health[0].Desynced || health[0].QueuedBytes != 0 {
		t.Fatalf("inactive local health = %+v", health)
	}
}

func TestOutputRouterDisconnectAfterAcceptDesynchronizesDestination(t *testing.T) {
	router := newOutputRouter(1024, 16)
	router.SetDestination(OutputDestinationCloud, &disconnectAfterAcceptWriter{})
	defer router.stop()

	router.enqueue([]byte("accepted"))
	if router.barrier().wait(time.Second) {
		t.Fatal("barrier succeeded after accepted destination disconnected")
	}
	health := router.Health()
	if len(health) != 1 || health[0].Reason != OutputDesyncDisconnected {
		t.Fatalf("disconnect health = %+v", health)
	}
}

func TestOutputRouterDesyncedDestinationDoesNotBlockHealthyDestination(t *testing.T) {
	slow := newGateRelayWriter()
	fast := newMockRelayWriter(true)
	router := newOutputRouter(8, 8)
	healthEvents := make(chan OutputHealth, 2)
	router.SetHealthHandler(func(health OutputHealth) { healthEvents <- health })
	router.SetDestination(OutputDestinationCloud, fast)
	router.SetDestination(OutputDestinationLocal, slow)
	defer router.stop()

	router.enqueue([]byte("aaaa"))
	select {
	case <-slow.started:
	case <-time.After(time.Second):
		t.Fatal("blocked destination did not start")
	}
	router.enqueue([]byte("bbbb"))
	waitForData(t, fast, "aaaabbbb")
	router.enqueue([]byte("x"))

	select {
	case state := <-healthEvents:
		if state.Destination != OutputDestinationLocal || state.Reason != OutputDesyncOverflow {
			t.Fatalf("desync health = %+v", state)
		}
	case <-time.After(time.Second):
		t.Fatal("slow destination did not desynchronize")
	}

	router.enqueue([]byte("healthy"))
	waitForData(t, fast, "aaaabbbbxhealthy")
	close(slow.release)
}

func waitForData(t *testing.T, relay *mockRelayWriter, want string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if got := string(relay.getData()); got == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("relay data = %q, want %q", relay.getData(), want)
}
