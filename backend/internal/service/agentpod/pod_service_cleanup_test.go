package agentpod

import (
	"context"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

func TestCleanupStalePods(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	// Create a disconnected pod
	req := &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		CreatedByID:    1,
	}
	sess, _ := svc.CreatePod(ctx, req)
	svc.UpdatePodStatus(ctx, sess.PodKey, agentpod.StatusRunning)
	svc.MarkDisconnected(ctx, sess.PodKey)

	// Update last_activity to be old
	db.Exec("UPDATE pods SET last_activity = ? WHERE pod_key = ?",
		time.Now().Add(-48*time.Hour), sess.PodKey)

	count, err := svc.CleanupStalePods(ctx, 24) // 24 hours
	if err != nil {
		t.Fatalf("CleanupStalePods failed: %v", err)
	}
	if count != 1 {
		t.Errorf("Cleaned up count = %d, want 1", count)
	}

	updated, _ := svc.GetPod(ctx, sess.PodKey)
	if updated.Status != agentpod.StatusTerminated {
		t.Errorf("Status = %s, want terminated", updated.Status)
	}
}

func TestReconcilePods(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	// Create pods
	var keys []string
	for i := 0; i < 3; i++ {
		req := &CreatePodRequest{
			OrganizationID: 1,
			RunnerID:       1,
			CreatedByID:    1,
		}
		sess, _ := svc.CreatePod(ctx, req)
		svc.UpdatePodStatus(ctx, sess.PodKey, agentpod.StatusRunning)
		keys = append(keys, sess.PodKey)
	}

	// Only report 2 of 3 pods in heartbeat
	err := svc.ReconcilePods(ctx, 1, keys[:2])
	if err != nil {
		t.Fatalf("ReconcilePods failed: %v", err)
	}

	// Third pod should be marked as orphaned
	orphaned, _ := svc.GetPod(ctx, keys[2])
	if orphaned.Status != agentpod.StatusOrphaned {
		t.Errorf("Status = %s, want orphaned", orphaned.Status)
	}
}
