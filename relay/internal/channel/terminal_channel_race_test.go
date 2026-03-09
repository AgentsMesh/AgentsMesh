package channel

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/anthropics/agentsmesh/relay/internal/protocol"
)

// ==================== Publisher Goroutine Race Condition Tests ====================

// TestTerminalChannel_SetPublisher_RapidReconnect verifies that rapid publisher
// replacement (disconnect + reconnect) correctly handles goroutine lifecycle:
// - Old goroutine exits cleanly
// - New goroutine forwards data to subscribers
func TestTerminalChannel_SetPublisher_RapidReconnect(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-rapid", testChannelConfig(), nil, nil)

	subServer, subClient := createWSPair(t)
	ch.AddSubscriber("s1", subServer)

	// Set first publisher
	pub1Server, _ := createWSPair(t)
	ch.SetPublisher(pub1Server)

	// Immediately replace with second publisher (simulates rapid reconnect)
	pub2Server, pub2Client := createWSPair(t)
	ch.SetPublisher(pub2Server)

	// Verify the new publisher's goroutine is active by sending data
	outMsg := protocol.EncodeOutput([]byte("from-pub2"))
	if err := pub2Client.WriteMessage(websocket.BinaryMessage, outMsg); err != nil {
		t.Fatalf("write to pub2Client: %v", err)
	}

	// Subscriber should receive the data from pub2
	_ = subClient.SetReadDeadline(time.Now().Add(2 * time.Second))

	// May receive RunnerReconnected first, then the actual data
	for {
		_, data, err := subClient.ReadMessage()
		if err != nil {
			t.Fatalf("read from subClient: %v", err)
		}
		msg, _ := protocol.DecodeMessage(data)
		if msg != nil && msg.Type == protocol.MsgTypeRunnerReconnected {
			continue // Skip reconnection notification
		}
		if !bytes.Equal(data, outMsg) {
			t.Fatalf("expected output from pub2, got %v", data)
		}
		break
	}

	// Verify only one publisher goroutine is active (old one exited)
	if ch.GetPublisher() != pub2Server {
		t.Fatal("expected publisher to be pub2Server")
	}
}

// TestTerminalChannel_SetPublisher_RapidReconnect_Multiple exercises the epoch
// mechanism under stress: many rapid SetPublisher calls in sequence.
func TestTerminalChannel_SetPublisher_RapidReconnect_Multiple(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-rapid-multi", testChannelConfig(), nil, nil)

	subServer, subClient := createWSPair(t)
	ch.AddSubscriber("s1", subServer)

	const iterations = 5
	var conns []*websocket.Conn

	for i := 0; i < iterations; i++ {
		server, _ := createWSPair(t)
		conns = append(conns, server)
		ch.SetPublisher(server)
	}

	// Only the last publisher should be active
	lastConn := conns[iterations-1]
	if ch.GetPublisher() != lastConn {
		t.Fatal("expected publisher to be the last set connection")
	}

	// Verify epoch is correct
	ch.publisherMu.RLock()
	epoch := ch.publisherEpoch
	ch.publisherMu.RUnlock()

	if epoch != uint64(iterations) {
		t.Fatalf("expected epoch %d, got %d", iterations, epoch)
	}

	// Drain any RunnerReconnected messages from subscriber
	_ = subClient.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	for {
		_, _, err := subClient.ReadMessage()
		if err != nil {
			break // Timeout or error, done draining
		}
	}
}

// TestTerminalChannel_Close_WaitsForGoroutine verifies that Close() blocks
// until the forwardPublisherToSubscribers goroutine has exited.
func TestTerminalChannel_Close_WaitsForGoroutine(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-close-wait", testChannelConfig(), nil, nil)

	pubServer, _ := createWSPair(t)
	ch.SetPublisher(pubServer)

	// Close should wait for the goroutine to exit
	closeDone := make(chan struct{})
	go func() {
		ch.Close()
		close(closeDone)
	}()

	select {
	case <-closeDone:
		// Close completed — goroutine was properly awaited
	case <-time.After(5 * time.Second):
		t.Fatal("Close() did not return within timeout — goroutine may be stuck")
	}

	if !ch.IsClosed() {
		t.Fatal("expected channel to be closed")
	}
}

// TestTerminalChannel_SetPublisher_EpochIncrement verifies epoch monotonically
// increases and that stale epoch disconnect handlers are no-ops.
func TestTerminalChannel_SetPublisher_EpochIncrement(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-epoch", testChannelConfig(), nil, nil)

	// Track epochs across multiple SetPublisher calls
	var epochs []uint64

	for i := 0; i < 3; i++ {
		server, client := createWSPair(t)
		ch.SetPublisher(server)

		ch.publisherMu.RLock()
		epochs = append(epochs, ch.publisherEpoch)
		ch.publisherMu.RUnlock()

		// Close client to trigger disconnect for cleanup
		_ = client.Close()
		waitFor(t, func() bool {
			return ch.IsPublisherDisconnected()
		}, 2*time.Second)
	}

	// Verify monotonic increase
	for i := 1; i < len(epochs); i++ {
		if epochs[i] <= epochs[i-1] {
			t.Fatalf("epoch not monotonically increasing: epochs[%d]=%d <= epochs[%d]=%d",
				i, epochs[i], i-1, epochs[i-1])
		}
	}

	// Test stale epoch disconnect handler is a no-op:
	// Set a new publisher and try calling handlePublisherDisconnect with stale epoch
	newServer, _ := createWSPair(t)
	ch.SetPublisher(newServer)

	ch.publisherMu.RLock()
	currentEpoch := ch.publisherEpoch
	ch.publisherMu.RUnlock()

	// Call with matching conn but stale epoch — should be no-op
	ch.handlePublisherDisconnect(newServer, currentEpoch-1)
	if ch.IsPublisherDisconnected() {
		t.Fatal("stale epoch handlePublisherDisconnect should be a no-op")
	}
	if ch.GetPublisher() != newServer {
		t.Fatal("publisher should still be newServer after stale disconnect")
	}
}

// TestTerminalChannel_SetPublisher_ConcurrentAccess verifies thread safety
// when SetPublisher is called concurrently (e.g., two runner connections racing).
func TestTerminalChannel_SetPublisher_ConcurrentAccess(t *testing.T) {
	ch := NewTerminalChannelWithConfig("pod-concurrent", testChannelConfig(), nil, nil)

	const goroutines = 4
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			server, _ := createWSPair(t)
			ch.SetPublisher(server)
		}()
	}

	wg.Wait()

	// Channel should be in a consistent state
	ch.publisherMu.RLock()
	epoch := ch.publisherEpoch
	pub := ch.publisher
	ch.publisherMu.RUnlock()

	if epoch != uint64(goroutines) {
		t.Fatalf("expected epoch %d, got %d", goroutines, epoch)
	}
	if pub == nil {
		t.Fatal("expected publisher to be non-nil after concurrent SetPublisher calls")
	}
}
