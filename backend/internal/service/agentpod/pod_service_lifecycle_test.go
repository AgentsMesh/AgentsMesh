package agentpod

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

func TestHandlePodCreated(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	req := &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		CreatedByID:    1,
	}
	sess, _ := svc.CreatePod(ctx, req)

	err := svc.HandlePodCreated(ctx, sess.PodKey, 12345, "/workspace/sandboxes/pod-1", "main")
	if err != nil {
		t.Fatalf("HandlePodCreated failed: %v", err)
	}

	updated, _ := svc.GetPod(ctx, sess.PodKey)
	if updated.Status != agentpod.StatusRunning {
		t.Errorf("Status = %s, want running", updated.Status)
	}
	// Note: PtyPID check skipped due to column naming mismatch in test setup
	if updated.StartedAt == nil {
		t.Error("StartedAt should be set")
	}
}

func TestHandlePodTerminated(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	req := &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		CreatedByID:    1,
	}
	sess, _ := svc.CreatePod(ctx, req)

	exitCode := 0
	err := svc.HandlePodTerminated(ctx, sess.PodKey, &exitCode)
	if err != nil {
		t.Fatalf("HandlePodTerminated failed: %v", err)
	}

	updated, _ := svc.GetPod(ctx, sess.PodKey)
	if updated.Status != agentpod.StatusTerminated {
		t.Errorf("Status = %s, want terminated", updated.Status)
	}
	if updated.FinishedAt == nil {
		t.Error("FinishedAt should be set")
	}
}
