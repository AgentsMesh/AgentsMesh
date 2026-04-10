package runner

import (
	"context"
	"fmt"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

// errorInjectingPodStore wraps a PodStore and allows injecting errors
// into GetByKeyAndRunner to simulate transient DB failures.
type errorInjectingPodStore struct {
	PodStore
	getByKeyAndRunnerFn func(ctx context.Context, podKey string, runnerID int64) (*agentpod.Pod, error)
}

func (r *errorInjectingPodStore) GetByKeyAndRunner(ctx context.Context, podKey string, runnerID int64) (*agentpod.Pod, error) {
	if r.getByKeyAndRunnerFn != nil {
		return r.getByKeyAndRunnerFn(ctx, podKey, runnerID)
	}
	return r.PodStore.GetByKeyAndRunner(ctx, podKey, runnerID)
}

func TestReconcilePods_TransientDBError_NoTerminate(t *testing.T) {
	db, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)
	defer cm.Close()

	errStore := &errorInjectingPodStore{
		PodStore: podStore,
		getByKeyAndRunnerFn: func(_ context.Context, _ string, _ int64) (*agentpod.Pod, error) {
			return nil, fmt.Errorf("connection refused")
		},
	}

	logger := newTestLogger()
	pc := NewPodCoordinator(errStore, runnerRepo, cm, tr, hb, logger)

	mockSender := &MockCommandSender{}
	pc.SetCommandSender(mockSender)

	// Create a runner
	r := &runner.Runner{OrganizationID: 1, NodeID: "transient-err-node", Status: "online"}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	ctx := context.Background()
	reportedPods := map[string]bool{"transient-pod-1": true}

	// Send many heartbeats — transient error should never trigger terminate
	for i := 0; i < orphanMissThreshold*2; i++ {
		pc.reconcilePods(ctx, r.ID, reportedPods)
	}

	assert.Equal(t, 0, mockSender.TerminatePodCalls,
		"transient DB error must not trigger terminate")
}

func TestReconcilePods_NotFound_WithEvidence(t *testing.T) {
	pc, _, _, db := setupPodEventHandlerDeps(t)

	mockSender := &MockCommandSender{}
	pc.SetCommandSender(mockSender)

	// Create a runner
	r := &runner.Runner{OrganizationID: 1, NodeID: "evidence-node", Status: "online"}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	ctx := context.Background()
	// Report a pod that does NOT exist in DB
	reportedPods := map[string]bool{"ghost-pod-1": true}

	// Before threshold: no terminate should be sent
	for i := 0; i < orphanMissThreshold-1; i++ {
		pc.reconcilePods(ctx, r.ID, reportedPods)
	}
	assert.Equal(t, 0, mockSender.TerminatePodCalls,
		"should not terminate before evidence threshold")

	// At threshold: terminate should be sent
	pc.reconcilePods(ctx, r.ID, reportedPods)
	assert.Equal(t, 1, mockSender.TerminatePodCalls,
		"should terminate after evidence threshold")
}

// TestReconcilePods_MixedTransientAndNotFound verifies that transient DB errors
// do NOT contribute to the miss count, and only consecutive NotFound errors
// accumulate toward the terminate threshold.
func TestReconcilePods_MixedTransientAndNotFound(t *testing.T) {
	db, cm, tr, hb, podStore, runnerRepo := setupPodCoordinatorDeps(t)
	defer cm.Close()

	callCount := 0
	errStore := &errorInjectingPodStore{
		PodStore: podStore,
		getByKeyAndRunnerFn: func(_ context.Context, _ string, _ int64) (*agentpod.Pod, error) {
			callCount++
			// Alternate: NotFound, transient, NotFound, transient, ...
			if callCount%2 == 1 {
				return nil, gorm.ErrRecordNotFound
			}
			return nil, fmt.Errorf("connection timeout")
		},
	}

	logger := newTestLogger()
	pc := NewPodCoordinator(errStore, runnerRepo, cm, tr, hb, logger)

	mockSender := &MockCommandSender{}
	pc.SetCommandSender(mockSender)

	r := &runner.Runner{OrganizationID: 1, NodeID: "mixed-err-node", Status: "online"}
	if err := db.Create(r).Error; err != nil {
		t.Fatalf("failed to create runner: %v", err)
	}

	ctx := context.Background()
	reportedPods := map[string]bool{"mixed-pod-1": true}

	// With alternating errors and threshold=3, only odd-numbered heartbeats
	// (NotFound) increment the miss count. We need 2*threshold - 1 heartbeats
	// for the miss count to reach threshold:
	// HB1: NotFound → miss=1, HB2: transient → skip
	// HB3: NotFound → miss=2, HB4: transient → skip
	// HB5: NotFound → miss=3 → terminate!

	// After 4 heartbeats (miss count = 2, below threshold): no terminate
	for i := 0; i < 4; i++ {
		pc.reconcilePods(ctx, r.ID, reportedPods)
	}
	assert.Equal(t, 0, mockSender.TerminatePodCalls,
		"transient errors should not contribute to miss count")

	// 5th heartbeat (NotFound): miss count reaches threshold → terminate
	pc.reconcilePods(ctx, r.ID, reportedPods)
	assert.Equal(t, 1, mockSender.TerminatePodCalls,
		"should terminate after enough NotFound evidence despite interleaved transient errors")
}
