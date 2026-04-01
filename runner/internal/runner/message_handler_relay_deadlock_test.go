package runner

import (
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

// TestOnSubscribePod_NoDeadlockWhenVTBusy verifies that OnSubscribePod
// does not deadlock when VT's write lock is held by a concurrent Feed() call.
// The original bug: relayMu held → GetSnapshot() needs vt.mu → Feed() holds vt.mu → deadlock.
// The fix uses TryGetSnapshot outside relayMu, so this test must complete within the timeout.
func TestOnSubscribePod_NoDeadlockWhenVTBusy(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Create a real VT to exercise the lock contention path.
	terminal := vt.NewVirtualTerminal(80, 24, 1000)

	// Inject a mock factory so Connect/Start succeed without network I/O.
	var createdClient *relay.MockClient
	handler.relayClientFactory = func(url, podKey, token string, logger *slog.Logger) relay.RelayClient {
		mc := relay.NewMockClient(url)
		createdClient = mc
		return mc
	}

	pod := testNewPTYPod("pod-deadlock-1", terminal)
	pod.Status = PodStatusRunning
	store.Put(pod.PodKey, pod)

	// Continuously feed VT to hold vt.mu write lock under contention.
	stopFeed := make(chan struct{})
	go func() {
		data := []byte("hello world\r\n")
		for {
			select {
			case <-stopFeed:
				return
			default:
				terminal.Feed(data)
			}
		}
	}()
	defer close(stopFeed)

	// OnSubscribePod must complete within the timeout — a deadlock means failure.
	done := make(chan error, 1)
	go func() {
		done <- handler.OnSubscribePod(client.SubscribePodRequest{
			PodKey:      pod.PodKey,
			RelayURL:    "wss://relay.example.com",
			RunnerToken: "token-1",
		})
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("OnSubscribePod returned error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("deadlock detected: OnSubscribePod blocked for 3s")
	}

	// Verify the client was set up.
	rc := pod.GetRelayClient()
	if rc == nil {
		t.Fatal("expected relay client to be set after subscribe")
	}

	// SendSnapshot may or may not have been called (TryGetSnapshot can return nil
	// if the VT lock is busy), both outcomes are valid.
	_ = createdClient
}

// TestOnSubscribePod_ConcurrentSubscribes verifies that multiple concurrent
// subscribe PTY requests do not deadlock when VT is busy.
func TestOnSubscribePod_ConcurrentSubscribes(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	terminal := vt.NewVirtualTerminal(80, 24, 1000)

	// Factory that creates a fresh MockClient for each call.
	handler.relayClientFactory = func(url, podKey, token string, logger *slog.Logger) relay.RelayClient {
		return relay.NewMockClient(url)
	}

	pod := testNewPTYPod("pod-concurrent", terminal)
	pod.Status = PodStatusRunning
	store.Put(pod.PodKey, pod)

	// Continuous VT feed to create lock contention.
	stopFeed := make(chan struct{})
	go func() {
		data := []byte("output line\r\n")
		for {
			select {
			case <-stopFeed:
				return
			default:
				terminal.Feed(data)
			}
		}
	}()
	defer close(stopFeed)

	const concurrency = 10
	var wg sync.WaitGroup
	wg.Add(concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			defer wg.Done()
			_ = handler.OnSubscribePod(client.SubscribePodRequest{
				PodKey:      pod.PodKey,
				RelayURL:    "wss://relay.example.com",
				RunnerToken: "token-concurrent",
			})
		}()
	}

	// All goroutines must finish within the timeout.
	allDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(allDone)
	}()

	select {
	case <-allDone:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("deadlock detected: concurrent OnSubscribePod blocked for 5s")
	}

	// At least one subscribe must have succeeded.
	if pod.GetRelayClient() == nil {
		t.Error("expected at least one relay client to be set")
	}
}

// TestOnSubscribePod_RaceConditionTwoSubscribers verifies that when two
// goroutines concurrently complete Phase 2 (Connect + Start), only one wins
// the Phase 3 pointer swap and the loser's client is properly stopped.
func TestOnSubscribePod_RaceConditionTwoSubscribers(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Track all created clients to verify cleanup.
	var clientsMu sync.Mutex
	var allClients []*relay.MockClient

	handler.relayClientFactory = func(url, podKey, token string, logger *slog.Logger) relay.RelayClient {
		mc := relay.NewMockClient(url)
		clientsMu.Lock()
		allClients = append(allClients, mc)
		clientsMu.Unlock()
		return mc
	}

	pod := &Pod{
		PodKey: "pod-race",
		Status: PodStatusRunning,
	}
	store.Put(pod.PodKey, pod)

	var wg sync.WaitGroup
	wg.Add(2)

	// Two subscribers with different relay URLs race to set the client.
	for i := 0; i < 2; i++ {
		i := i
		go func() {
			defer wg.Done()
			_ = handler.OnSubscribePod(client.SubscribePodRequest{
				PodKey:      pod.PodKey,
				RelayURL:    "wss://relay.example.com",
				RunnerToken: "token-" + string(rune('A'+i)),
			})
		}()
	}

	allDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(allDone)
	}()

	select {
	case <-allDone:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("deadlock detected: two concurrent subscribers blocked for 5s")
	}

	// Exactly one client should be active.
	if pod.GetRelayClient() == nil {
		t.Fatal("expected a relay client to be set")
	}

	// Verify no leaked clients: every client that lost the race should have Stop() called.
	clientsMu.Lock()
	defer clientsMu.Unlock()

	var stoppedCount int32
	for _, mc := range allClients {
		if mc.StopCalled {
			stoppedCount++
		}
	}

	// With 2 clients created, at most 1 should remain (the winner).
	// The loser(s) must have been stopped.
	activeCount := int32(len(allClients)) - stoppedCount
	if activeCount > 1 {
		t.Errorf("relay client leak: %d clients active (expected at most 1), %d total created, %d stopped",
			activeCount, len(allClients), stoppedCount)
	}
}

// TestOnSubscribePod_SnapshotSentOnSuccess verifies that Send(MsgTypeSnapshot, ...)
// is called when VT lock is available and the subscribe succeeds.
func TestOnSubscribePod_SnapshotSentOnSuccess(t *testing.T) {
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{cfg: &config.Config{}}
	handler := NewRunnerMessageHandler(runner, store, mockConn)

	terminal := vt.NewVirtualTerminal(80, 24, 1000)
	// Feed some content so snapshot is non-nil.
	terminal.Feed([]byte("Hello, World!\r\n"))

	var snapshotCalls atomic.Int32
	handler.relayClientFactory = func(url, podKey, token string, logger *slog.Logger) relay.RelayClient {
		mc := relay.NewMockClient(url)
		// Wrap Send to count snapshot calls atomically.
		return &snapshotTrackingClient{MockClient: mc, calls: &snapshotCalls}
	}

	comps := &PTYComponents{VirtualTerminal: terminal}
	pod := &Pod{
		PodKey: "pod-snapshot",
		Status: PodStatusRunning,
	}
	pod.IO = NewPTYPodIO(pod.PodKey, comps, PTYPodIODeps{})
	pod.Relay = NewPTYPodRelay(pod.PodKey, pod.IO, comps)
	store.Put(pod.PodKey, pod)

	err := handler.OnSubscribePod(client.SubscribePodRequest{
		PodKey:      pod.PodKey,
		RelayURL:    "wss://relay.example.com",
		RunnerToken: "token-snap",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Without VT contention, TryGetSnapshot should succeed and Send(MsgTypeSnapshot) should be called.
	if snapshotCalls.Load() == 0 {
		t.Error("expected Send(MsgTypeSnapshot) to be called when VT lock is available")
	}
}

// snapshotTrackingClient wraps MockClient to track Send(MsgTypeSnapshot) calls atomically.
type snapshotTrackingClient struct {
	*relay.MockClient
	calls *atomic.Int32
}

func (s *snapshotTrackingClient) Send(msgType byte, payload []byte) error {
	if msgType == relay.MsgTypeSnapshot {
		s.calls.Add(1)
	}
	return s.MockClient.Send(msgType, payload)
}
