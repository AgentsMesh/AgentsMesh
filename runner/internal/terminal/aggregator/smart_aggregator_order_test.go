package aggregator

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"
)

type blockingFirstRelayWriter struct {
	firstStarted  chan struct{}
	releaseFirst  chan struct{}
	secondStarted chan struct{}
	firstOnce     sync.Once
	secondOnce    sync.Once

	mu    sync.Mutex
	calls []string
}

type relayWriterBlockedBeforeSend struct {
	connectionChecked chan struct{}
	releaseCheck      chan struct{}
	sendCalled        chan struct{}
	checkOnce         sync.Once
	sendOnce          sync.Once

	mu    sync.Mutex
	calls []string
}

func newRelayWriterBlockedBeforeSend() *relayWriterBlockedBeforeSend {
	return &relayWriterBlockedBeforeSend{
		connectionChecked: make(chan struct{}),
		releaseCheck:      make(chan struct{}),
		sendCalled:        make(chan struct{}),
	}
}

func (r *relayWriterBlockedBeforeSend) IsConnected() bool {
	r.checkOnce.Do(func() { close(r.connectionChecked) })
	<-r.releaseCheck
	return true
}

func (r *relayWriterBlockedBeforeSend) SendOutput(data []byte) error {
	r.mu.Lock()
	r.calls = append(r.calls, string(data))
	r.mu.Unlock()
	r.sendOnce.Do(func() { close(r.sendCalled) })
	return nil
}

func (r *relayWriterBlockedBeforeSend) FlushOutput(context.Context) error { return nil }

func (r *relayWriterBlockedBeforeSend) sentCalls() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...)
}

func newBlockingFirstRelayWriter() *blockingFirstRelayWriter {
	return &blockingFirstRelayWriter{
		firstStarted:  make(chan struct{}),
		releaseFirst:  make(chan struct{}),
		secondStarted: make(chan struct{}),
	}
}

func (r *blockingFirstRelayWriter) IsConnected() bool { return true }

func (r *blockingFirstRelayWriter) SendOutput(data []byte) error {
	value := string(data)
	switch value {
	case "first":
		r.firstOnce.Do(func() { close(r.firstStarted) })
		<-r.releaseFirst
	}

	r.mu.Lock()
	r.calls = append(r.calls, value)
	r.mu.Unlock()
	if value == "second" {
		r.secondOnce.Do(func() { close(r.secondStarted) })
	}
	return nil
}

func (r *blockingFirstRelayWriter) FlushOutput(context.Context) error { return nil }

func (r *blockingFirstRelayWriter) sentCalls() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...)
}

func TestSmartAggregatorFlushesRouteInFIFOOrder(t *testing.T) {
	relay := newBlockingFirstRelayWriter()
	agg := NewSmartAggregator(
		func() float64 { return 0 },
		WithSmartBaseDelay(time.Hour),
	)
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

	// Keep the first network write blocked while enqueueing another flush. With
	// one goroutine per flush, the second SendOutput overtakes it deterministically.
	agg.Write([]byte("second"))
	agg.Flush()
	select {
	case <-relay.secondStarted:
		t.Error("second flush reached relay before the first write completed")
	case <-time.After(100 * time.Millisecond):
	}

	close(relay.releaseFirst)
	released = true
	select {
	case <-relay.secondStarted:
	case <-time.After(time.Second):
		t.Fatal("second flush did not reach relay after the first write completed")
	}

	if got, want := relay.sentCalls(), []string{"first", "second"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("flush route order = %v, want %v", got, want)
	}
}

func TestSmartAggregatorDropsQueuedOutputFromReplacedRelayGeneration(t *testing.T) {
	oldRelay := newBlockingFirstRelayWriter()
	newRelay := newMockRelayWriter(true)
	agg := NewSmartAggregator(
		func() float64 { return 0 },
		WithSmartBaseDelay(time.Hour),
	)
	agg.SetOutputDestination(OutputDestinationCloud, oldRelay)

	released := false
	defer func() {
		if !released {
			close(oldRelay.releaseFirst)
		}
		agg.Stop()
	}()

	agg.Write([]byte("first"))
	agg.Flush()
	select {
	case <-oldRelay.firstStarted:
	case <-time.After(time.Second):
		t.Fatal("first flush did not reach old relay")
	}

	// Keep the old generation blocked so stale output remains queued while the
	// relay is replaced. Only bytes enqueued after replacement may reach newRelay.
	agg.Write([]byte("stale"))
	agg.Flush()
	replacementDone := make(chan struct{})
	go func() {
		agg.SetOutputDestination(OutputDestinationCloud, newRelay)
		close(replacementDone)
	}()
	select {
	case <-replacementDone:
	case <-time.After(time.Second):
		t.Fatal("relay replacement blocked on the active old send")
	}

	agg.Write([]byte("fresh"))
	agg.Flush()
	assertRelayHasNoDataFor(t, newRelay, 100*time.Millisecond,
		"new-generation output overtook the active old send")

	close(oldRelay.releaseFirst)
	released = true
	waitForRelayData(t, newRelay, "fresh")

	if got := string(newRelay.getData()); got != "fresh" {
		t.Fatalf("new relay received cross-generation output %q, want %q", got, "fresh")
	}
	if got, want := oldRelay.sentCalls(), []string{"first"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("old relay calls = %v, want %v", got, want)
	}
}

func TestSmartAggregatorRelayReplacementDoesNotCancelOwnedSend(t *testing.T) {
	oldRelay := newRelayWriterBlockedBeforeSend()
	newRelay := newMockRelayWriter(true)
	agg := NewSmartAggregator(
		func() float64 { return 0 },
		WithSmartBaseDelay(time.Hour),
	)
	agg.SetOutputDestination(OutputDestinationCloud, oldRelay)

	released := false
	defer func() {
		if !released {
			close(oldRelay.releaseCheck)
		}
		agg.Stop()
	}()

	agg.Write([]byte("old-generation"))
	agg.Flush()
	select {
	case <-oldRelay.connectionChecked:
		// routeQueued has selected the old generation and is immediately before
		// SendOutput, blocked inside IsConnected.
	case <-time.After(time.Second):
		t.Fatal("old relay was not selected for queued output")
	}

	replacementDone := make(chan struct{})
	go func() {
		agg.SetOutputDestination(OutputDestinationCloud, newRelay)
		close(replacementDone)
	}()

	select {
	case <-replacementDone:
	case <-time.After(time.Second):
		t.Fatal("relay replacement blocked on an old writer call before SendOutput")
	}

	agg.Write([]byte("new-generation"))
	agg.Flush()
	assertRelayHasNoDataFor(t, newRelay, 100*time.Millisecond,
		"new-generation output overtook the owned old writer call")

	close(oldRelay.releaseCheck)
	released = true
	select {
	case <-oldRelay.sendCalled:
	case <-time.After(time.Second):
		t.Fatal("approved old send did not complete")
	}
	if got, want := oldRelay.sentCalls(), []string{"old-generation"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("old relay calls = %v, want %v", got, want)
	}

	waitForRelayData(t, newRelay, "new-generation")
}

func assertRelayHasNoDataFor(t *testing.T, relay *mockRelayWriter, duration time.Duration, message string) {
	t.Helper()
	deadline := time.Now().Add(duration)
	for time.Now().Before(deadline) {
		if got := relay.getData(); len(got) != 0 {
			t.Fatalf("%s: got %q", message, got)
		}
		time.Sleep(time.Millisecond)
	}
}

func waitForRelayData(t *testing.T, relay *mockRelayWriter, want string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if string(relay.getData()) == want {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("relay data = %q, want %q", relay.getData(), want)
}
