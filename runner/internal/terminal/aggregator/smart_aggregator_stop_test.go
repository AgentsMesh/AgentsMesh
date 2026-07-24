package aggregator

import (
	"reflect"
	"testing"
	"time"
)

func TestSmartAggregatorStopDrainsBeforeRelayIsCleared(t *testing.T) {
	relay := newBlockingFirstRelayWriter()
	agg := NewSmartAggregator(func() float64 { return 0 }, WithSmartBaseDelay(time.Hour))
	agg.SetOutputDestination(OutputDestinationCloud, relay)

	released := false
	defer func() {
		if !released {
			close(relay.releaseFirst)
		}
		agg.Stop()
	}()

	agg.Write([]byte("first"))
	agg.Flush()
	select {
	case <-relay.firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first flush did not reach relay")
	}
	agg.Write([]byte("second"))

	teardownDone := make(chan struct{})
	go func() {
		agg.Stop()
		agg.RemoveOutputDestination(OutputDestinationCloud)
		close(teardownDone)
	}()
	select {
	case <-teardownDone:
		t.Error("teardown cleared relay before queued output drained")
	case <-time.After(100 * time.Millisecond):
	}

	close(relay.releaseFirst)
	released = true
	select {
	case <-teardownDone:
	case <-time.After(time.Second):
		t.Fatal("teardown did not finish after relay writer unblocked")
	}
	if got, want := relay.sentCalls(), []string{"first", "second"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("teardown flush calls = %v, want %v", got, want)
	}
	if len(agg.router.Health()) != 0 {
		t.Fatal("teardown did not clear relay after draining output")
	}
}

func TestSmartAggregatorStopDrainTimeoutIsBounded(t *testing.T) {
	relay := newBlockingFirstRelayWriter()
	agg := NewSmartAggregator(func() float64 { return 0 }, WithSmartBaseDelay(time.Hour))
	agg.drainTimeout = 30 * time.Millisecond
	agg.SetOutputDestination(OutputDestinationCloud, relay)
	agg.Write([]byte("first"))

	released := false
	defer func() {
		if !released {
			close(relay.releaseFirst)
		}
	}()
	startedAt := time.Now()
	agg.Stop()
	if elapsed := time.Since(startedAt); elapsed > time.Second {
		t.Fatalf("Stop blocked past its bounded drain timeout: %v", elapsed)
	}
	select {
	case <-relay.firstStarted:
	case <-time.After(time.Second):
		t.Fatal("blocked relay writer never received final flush")
	}

	clearDone := make(chan struct{})
	go func() {
		agg.RemoveOutputDestination(OutputDestinationCloud)
		close(clearDone)
	}()
	select {
	case <-clearDone:
	case <-time.After(time.Second):
		t.Fatal("relay clear blocked after the output drain timed out")
	}
	if len(agg.router.Health()) != 0 {
		t.Fatal("relay clear did not take effect after the output drain timed out")
	}

	close(relay.releaseFirst)
	released = true
	waitForBlockingRelayCalls(t, relay, 1)
}

func waitForBlockingRelayCalls(t *testing.T, relay *blockingFirstRelayWriter, want int) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if len(relay.sentCalls()) == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("relay call count = %d, want %d", len(relay.sentCalls()), want)
}
