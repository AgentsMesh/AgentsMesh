package runner

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/runner/internal/terminal/detector"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

func TestPod_GetOrCreateStateDetector(t *testing.T) {
	vterminal := vt.NewVirtualTerminal(80, 24, 1000)

	pod := &Pod{
		PodKey:     "test-pod",
		vtProvider: func() *vt.VirtualTerminal { return vterminal },
	}
	defer pod.StopStateDetector()

	// First call should create detector
	sd := pod.GetOrCreateStateDetector()
	require.NotNil(t, sd)

	// Second call should return same instance
	sd2 := pod.GetOrCreateStateDetector()
	assert.Equal(t, sd, sd2)
}

func TestPod_GetOrCreateStateDetector_NoVirtualTerminal(t *testing.T) {
	pod := &Pod{
		PodKey: "test-pod",
	}

	// Should return nil when no VirtualTerminal
	sd := pod.GetOrCreateStateDetector()
	assert.Nil(t, sd)
}

func TestPod_SubscribeStateChange(t *testing.T) {
	vterminal := vt.NewVirtualTerminal(80, 24, 1000)

	pod := &Pod{
		PodKey:     "test-pod",
		vtProvider: func() *vt.VirtualTerminal { return vterminal },
	}
	defer pod.StopStateDetector()

	var mu sync.Mutex
	var events []detector.StateChangeEvent

	// Subscribe
	ok := pod.SubscribeStateChange("test-subscriber", func(event detector.StateChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, event)
	})
	assert.True(t, ok, "Subscribe should succeed")

	// Trigger state change via NotifyStateDetectorWithScreen
	pod.NotifyStateDetectorWithScreen(100, []string{"test"})

	// Wait for async callback
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	require.Len(t, events, 1)
	assert.Equal(t, detector.StateExecuting, events[0].NewState)
}

func TestPod_SubscribeStateChange_NoVirtualTerminal(t *testing.T) {
	pod := &Pod{
		PodKey: "test-pod",
	}

	// Should return false when no VirtualTerminal
	ok := pod.SubscribeStateChange("test", func(event detector.StateChangeEvent) {})
	assert.False(t, ok, "Subscribe should fail without VirtualTerminal")
}

func TestPod_UnsubscribeStateChange(t *testing.T) {
	vterminal := vt.NewVirtualTerminal(80, 24, 1000)

	pod := &Pod{
		PodKey:     "test-pod",
		vtProvider: func() *vt.VirtualTerminal { return vterminal },
	}
	defer pod.StopStateDetector()

	var mu sync.Mutex
	var events []detector.StateChangeEvent

	// Subscribe
	pod.SubscribeStateChange("test-subscriber", func(event detector.StateChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		events = append(events, event)
	})

	// Unsubscribe
	pod.UnsubscribeStateChange("test-subscriber")

	// Trigger state change
	pod.NotifyStateDetectorWithScreen(100, []string{"test"})

	// Wait for potential async callback
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, events, 0, "Unsubscribed callback should not be called")
}

func TestPod_UnsubscribeStateChange_NoDetector(t *testing.T) {
	pod := &Pod{
		PodKey: "test-pod",
	}

	// Should not panic when no detector exists
	pod.UnsubscribeStateChange("test")
}

func TestPod_MultipleSubscribers(t *testing.T) {
	vterminal := vt.NewVirtualTerminal(80, 24, 1000)

	pod := &Pod{
		PodKey:     "test-pod",
		vtProvider: func() *vt.VirtualTerminal { return vterminal },
	}
	defer pod.StopStateDetector()

	var mu sync.Mutex
	sub1Events := []detector.StateChangeEvent{}
	sub2Events := []detector.StateChangeEvent{}

	// Subscribe two subscribers
	pod.SubscribeStateChange("sub1", func(event detector.StateChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		sub1Events = append(sub1Events, event)
	})
	pod.SubscribeStateChange("sub2", func(event detector.StateChangeEvent) {
		mu.Lock()
		defer mu.Unlock()
		sub2Events = append(sub2Events, event)
	})

	// Trigger state change
	pod.NotifyStateDetectorWithScreen(100, []string{"test"})

	// Wait for async callbacks
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	assert.Len(t, sub1Events, 1, "sub1 should receive event")
	assert.Len(t, sub2Events, 1, "sub2 should receive event")
}

func TestPod_NotifyStateDetectorWithScreen(t *testing.T) {
	vterminal := vt.NewVirtualTerminal(80, 24, 1000)

	pod := &Pod{
		PodKey:     "test-pod",
		vtProvider: func() *vt.VirtualTerminal { return vterminal },
	}

	// Create detector first
	sd := pod.GetOrCreateStateDetector()
	require.NotNil(t, sd)
	defer pod.StopStateDetector()

	// Notify should transition to Executing
	pod.NotifyStateDetectorWithScreen(100, []string{"$ "})
	assert.Equal(t, detector.StateExecuting, sd.GetState())
}

func TestPod_NotifyStateDetectorWithScreen_NoDetector(t *testing.T) {
	pod := &Pod{
		PodKey: "test-pod",
	}

	// Should not panic when no detector
	pod.NotifyStateDetectorWithScreen(100, []string{"test"})
}

func TestPod_NotifyStateDetectorWithScreen_NilScreenLines(t *testing.T) {
	vterminal := vt.NewVirtualTerminal(80, 24, 1000)

	pod := &Pod{
		PodKey:     "test-pod",
		vtProvider: func() *vt.VirtualTerminal { return vterminal },
	}
	defer pod.StopStateDetector()

	// Initialize detector
	_ = pod.GetOrCreateStateDetector()

	// Should not panic with nil screen lines
	pod.NotifyStateDetectorWithScreen(100, nil)
}

func TestPod_StopStateDetector(t *testing.T) {
	vterminal := vt.NewVirtualTerminal(80, 24, 1000)

	pod := &Pod{
		PodKey:     "test-pod",
		vtProvider: func() *vt.VirtualTerminal { return vterminal },
	}

	// Create detector
	sd := pod.GetOrCreateStateDetector()
	require.NotNil(t, sd)

	// Stop
	pod.StopStateDetector()

	// Detector should be nil after stop
	pod.stateDetectorMu.RLock()
	assert.Nil(t, pod.stateDetector)
	pod.stateDetectorMu.RUnlock()
}

func TestPod_StopStateDetector_NoDetector(t *testing.T) {
	pod := &Pod{
		PodKey: "test-pod",
	}

	// Should not panic when no detector
	pod.StopStateDetector()
}

func TestPod_ConcurrentSubscribeUnsubscribe(t *testing.T) {
	vterminal := vt.NewVirtualTerminal(80, 24, 1000)

	pod := &Pod{
		PodKey:     "test-pod",
		vtProvider: func() *vt.VirtualTerminal { return vterminal },
	}
	defer pod.StopStateDetector()

	var wg sync.WaitGroup

	// Concurrent subscribe/unsubscribe operations
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			subID := "sub-" + string(rune('A'+id))
			for j := 0; j < 50; j++ {
				pod.SubscribeStateChange(subID, func(event detector.StateChangeEvent) {})
				pod.NotifyStateDetectorWithScreen(10, []string{"test"})
				pod.UnsubscribeStateChange(subID)
			}
		}(i)
	}

	wg.Wait()
	// Test passes if no race conditions
}
