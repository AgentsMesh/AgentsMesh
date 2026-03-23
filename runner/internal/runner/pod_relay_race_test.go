package runner

import (
	"sync"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

// TestPodRelayOperations_RaceDetection exercises concurrent relay operations
// to detect data races via `go test -race`.
// This test does not assert specific outcomes — its value is in triggering
// the race detector on interleaved read/write access to Pod.RelayClient.
func TestPodRelayOperations_RaceDetection(t *testing.T) {
	pod := &Pod{PodKey: "pod-race-detect", Status: PodStatusRunning}

	const goroutines = 20
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	// Mix of operations that touch relayMu from different angles.
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				switch i % 4 {
				case 0:
					// SetRelayClient
					mc := relay.NewMockClient("wss://relay.example.com")
					pod.SetRelayClient(mc)
				case 1:
					// GetRelayClient
					_ = pod.GetRelayClient()
				case 2:
					// DisconnectRelay
					pod.DisconnectRelay()
				case 3:
					// HasRelayClient
					_ = pod.HasRelayClient()
				}
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All operations completed without panic or deadlock.
	case <-time.After(10 * time.Second):
		t.Fatal("timeout: concurrent relay operations blocked for 10s")
	}
}

// TestPodRelayOperations_LockRelayRaceDetection exercises the LockRelay/UnlockRelay
// path used by OnSubscribePod's check-and-swap pattern concurrently with
// SetRelayClient and GetRelayClient.
func TestPodRelayOperations_LockRelayRaceDetection(t *testing.T) {
	pod := &Pod{PodKey: "pod-lock-race", Status: PodStatusRunning}

	const goroutines = 10
	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				switch i % 3 {
				case 0:
					// Simulate OnSubscribePod's check-and-swap pattern.
					pod.LockRelay()
					existing := pod.RelayClient
					if existing == nil {
						pod.RelayClient = relay.NewMockClient("wss://relay.example.com")
					}
					pod.UnlockRelay()
				case 1:
					// Concurrent read.
					_ = pod.GetRelayClient()
				case 2:
					// Concurrent disconnect.
					pod.DisconnectRelay()
				}
			}
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(10 * time.Second):
		t.Fatal("timeout: LockRelay race detection blocked for 10s")
	}
}
