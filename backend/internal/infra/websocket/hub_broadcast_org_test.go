package websocket

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHubBroadcastToOrg(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	client1 := mockClient(1, 100) // Events channel
	client2 := mockClient(2, 100) // Events channel
	client3 := mockClient(3, 200) // Different org

	hub.Register(client1)
	hub.Register(client2)
	hub.Register(client3)
	time.Sleep(20 * time.Millisecond)

	hub.BroadcastToOrg(100, []byte(`{"event":"test"}`))

	// client1 and client2 should receive
	select {
	case <-client1.send:
	case <-time.After(100 * time.Millisecond):
		t.Error("client1 didn't receive message")
	}

	select {
	case <-client2.send:
	case <-time.After(100 * time.Millisecond):
		t.Error("client2 didn't receive message")
	}

	// client3 should NOT receive
	select {
	case <-client3.send:
		t.Error("client3 should not receive message")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestHubSendToUser(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	client1 := mockClient(123, 1) // Events for user 123
	client2 := mockClient(123, 1) // Same user
	client3 := mockClient(456, 1) // Different user

	hub.Register(client1)
	hub.Register(client2)
	hub.Register(client3)
	time.Sleep(20 * time.Millisecond)

	hub.SendToUser(123, []byte(`{"notification":"hello"}`))

	// client1 and client2 should receive
	select {
	case <-client1.send:
	case <-time.After(100 * time.Millisecond):
		t.Error("client1 didn't receive message")
	}

	select {
	case <-client2.send:
	case <-time.After(100 * time.Millisecond):
		t.Error("client2 didn't receive message")
	}

	// client3 should NOT receive
	select {
	case <-client3.send:
		t.Error("client3 should not receive message")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestHubBroadcastToOrgEmptyOrg(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Broadcast to non-existent org should not panic
	hub.BroadcastToOrg(99999, []byte(`{"test":"data"}`))
}

func TestHubSendToUserEmptyUser(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	// Send to non-existent user should not panic
	hub.SendToUser(99999, []byte(`{"test":"data"}`))
}

// Regression: broadcasting to an org while clients are concurrently
// unregistered must not panic. Pre-fix, BroadcastToOrg sent to client.send
// AFTER releasing the read lock, racing handleUnregister's close(client.send)
// → "send on closed channel". The fix sends under the read lock so the RWMutex
// serializes send-vs-close. No panic (which would crash the test binary) is the
// core assertion; the count check confirms the teardown actually ran.
func TestHubBroadcastToOrgConcurrentUnregisterNoPanic(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	const orgID int64 = 777
	const n = 50
	clients := make([]*Client, 0, n)
	for i := 0; i < n; i++ {
		c := mockClient(int64(i+1), orgID)
		hub.Register(c)
		clients = append(clients, c)
		// Drain like the Connect server-stream so the 256-slot send buffer never
		// fills. This keeps `client.send <- data` actually executing (instead of
		// falling to the unregister default branch), maximizing the window where
		// a broadcast send races handleUnregister's close — i.e. it genuinely
		// exercises the bug the fix targets. range exits when the channel closes.
		go func(cl *Client) {
			for range cl.Outbound() { //nolint:revive // intentional drain
			}
		}(c)
	}
	require.Eventually(t, func() bool { return hub.GetOrgClientCount(orgID) == n },
		2*time.Second, time.Millisecond, "clients did not all register")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 2000; i++ {
			hub.BroadcastToOrg(orgID, []byte(`{"event":"x"}`))
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for _, c := range clients {
			hub.Unregister(c)
			time.Sleep(50 * time.Microsecond)
		}
	}()
	wg.Wait()

	// No panic reached here. Confirm the async teardown drained every client.
	require.Eventually(t, func() bool { return hub.GetOrgClientCount(orgID) == 0 },
		2*time.Second, time.Millisecond, "clients were not all unregistered")
}
