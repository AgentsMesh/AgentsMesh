package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAckTracker_RegisterAndResolve(t *testing.T) {
	tracker := NewAckTracker()
	tracker.Register("pod-1")
	tracker.Resolve("pod-1")
	// No panic, no error — basic lifecycle works
}

func TestAckTracker_ResolveUnknownKey(t *testing.T) {
	tracker := NewAckTracker()
	tracker.Resolve("nonexistent") // should not panic
}

func TestAckTracker_ResolveIdempotent(t *testing.T) {
	tracker := NewAckTracker()
	tracker.Register("pod-1")
	tracker.Resolve("pod-1")
	tracker.Resolve("pod-1") // second call should not panic
}

func TestAckTracker_Remove(t *testing.T) {
	tracker := NewAckTracker()
	tracker.Register("pod-1")
	tracker.Remove("pod-1")
	// Subsequent Resolve should be a no-op
	tracker.Resolve("pod-1")
}

func TestAckTracker_RemoveUnknownKey(t *testing.T) {
	tracker := NewAckTracker()
	tracker.Remove("nonexistent") // should not panic
}

func TestInitReportCounter(t *testing.T) {
	pc, _, _, _ := setupPodEventHandlerDeps(t)

	// First report
	count := pc.incrementInitReportCount("pod-1")
	assert.Equal(t, 1, count)

	// Second report
	count = pc.incrementInitReportCount("pod-1")
	assert.Equal(t, 2, count)

	// Clear and re-increment
	pc.clearInitReportCount("pod-1")
	count = pc.incrementInitReportCount("pod-1")
	assert.Equal(t, 1, count)
}
