package agentpod

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

func TestUpdatePodStatus(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	// Create a pod
	req := &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		CreatedByID:    1,
	}
	sess, _ := svc.CreatePod(ctx, req)

	tests := []struct {
		name       string
		status     string
		checkField string
	}{
		{"to running", agentpod.StatusRunning, "started_at"},
		{"to terminated", agentpod.StatusTerminated, "finished_at"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh pod for each test
			sess, _ = svc.CreatePod(ctx, req)

			err := svc.UpdatePodStatus(ctx, sess.PodKey, tt.status)
			if err != nil {
				t.Errorf("UpdatePodStatus failed: %v", err)
			}

			// Verify
			updated, _ := svc.GetPod(ctx, sess.PodKey)
			if updated.Status != tt.status {
				t.Errorf("Status = %s, want %s", updated.Status, tt.status)
			}
		})
	}

	t.Run("non-existent pod", func(t *testing.T) {
		err := svc.UpdatePodStatus(ctx, "non-existent", agentpod.StatusRunning)
		if err == nil {
			t.Error("Expected error for non-existent pod")
		}
	})
}

func TestMarkDisconnected(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	req := &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		CreatedByID:    1,
	}
	sess, _ := svc.CreatePod(ctx, req)
	svc.UpdatePodStatus(ctx, sess.PodKey, agentpod.StatusRunning)

	err := svc.MarkDisconnected(ctx, sess.PodKey)
	if err != nil {
		t.Fatalf("MarkDisconnected failed: %v", err)
	}

	updated, _ := svc.GetPod(ctx, sess.PodKey)
	if updated.Status != agentpod.StatusDisconnected {
		t.Errorf("Status = %s, want disconnected", updated.Status)
	}
}

func TestMarkReconnected(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	req := &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		CreatedByID:    1,
	}
	sess, _ := svc.CreatePod(ctx, req)
	svc.UpdatePodStatus(ctx, sess.PodKey, agentpod.StatusRunning)
	svc.MarkDisconnected(ctx, sess.PodKey)

	err := svc.MarkReconnected(ctx, sess.PodKey)
	if err != nil {
		t.Fatalf("MarkReconnected failed: %v", err)
	}

	updated, _ := svc.GetPod(ctx, sess.PodKey)
	if updated.Status != agentpod.StatusRunning {
		t.Errorf("Status = %s, want running", updated.Status)
	}
}
