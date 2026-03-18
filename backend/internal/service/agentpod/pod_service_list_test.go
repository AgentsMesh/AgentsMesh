package agentpod

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

func TestListPods(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	// Create multiple pods
	for i := 0; i < 5; i++ {
		req := &CreatePodRequest{
			OrganizationID: 1,
			RunnerID:       1,
			CreatedByID:    int64(i + 1),
		}
		svc.CreatePod(ctx, req)
	}

	t.Run("list all", func(t *testing.T) {
		pods, total, err := svc.ListPods(ctx, 1, nil, 0, 10, 0)
		if err != nil {
			t.Fatalf("ListPods failed: %v", err)
		}
		if total != 5 {
			t.Errorf("Total = %d, want 5", total)
		}
		if len(pods) != 5 {
			t.Errorf("Pods count = %d, want 5", len(pods))
		}
	})

	t.Run("list with pagination", func(t *testing.T) {
		pods, total, err := svc.ListPods(ctx, 1, nil, 0, 2, 0)
		if err != nil {
			t.Fatalf("ListPods failed: %v", err)
		}
		if total != 5 {
			t.Errorf("Total = %d, want 5", total)
		}
		if len(pods) != 2 {
			t.Errorf("Pods count = %d, want 2", len(pods))
		}
	})

	t.Run("list with single status filter", func(t *testing.T) {
		pods, _, err := svc.ListPods(ctx, 1, []string{agentpod.StatusInitializing}, 0, 10, 0)
		if err != nil {
			t.Fatalf("ListPods failed: %v", err)
		}
		if len(pods) != 5 {
			t.Errorf("Pods count = %d, want 5", len(pods))
		}
	})

	// Update some pods to different statuses for multi-status test
	allPods, _, _ := svc.ListPods(ctx, 1, nil, 0, 10, 0)
	if len(allPods) >= 3 {
		svc.UpdatePodStatus(ctx, allPods[0].PodKey, agentpod.StatusRunning)
		svc.UpdatePodStatus(ctx, allPods[1].PodKey, agentpod.StatusTerminated)
	}

	t.Run("list with multiple status filter", func(t *testing.T) {
		pods, total, err := svc.ListPods(ctx, 1, []string{agentpod.StatusRunning, agentpod.StatusInitializing}, 0, 10, 0)
		if err != nil {
			t.Fatalf("ListPods failed: %v", err)
		}
		// 1 running + 3 still initializing = 4
		if total != 4 {
			t.Errorf("Total = %d, want 4", total)
		}
		if len(pods) != 4 {
			t.Errorf("Pods count = %d, want 4", len(pods))
		}
	})

	t.Run("list with non-matching status filter", func(t *testing.T) {
		pods, total, err := svc.ListPods(ctx, 1, []string{agentpod.StatusPaused}, 0, 10, 0)
		if err != nil {
			t.Fatalf("ListPods failed: %v", err)
		}
		if total != 0 {
			t.Errorf("Total = %d, want 0", total)
		}
		if len(pods) != 0 {
			t.Errorf("Pods count = %d, want 0", len(pods))
		}
	})

	t.Run("list filtered by creator", func(t *testing.T) {
		// CreatedByID 1 should match exactly 1 pod
		pods, total, err := svc.ListPods(ctx, 1, nil, 1, 10, 0)
		if err != nil {
			t.Fatalf("ListPods failed: %v", err)
		}
		if total != 1 {
			t.Errorf("Total = %d, want 1", total)
		}
		if len(pods) != 1 {
			t.Errorf("Pods count = %d, want 1", len(pods))
		}
	})

	t.Run("list filtered by creator with status", func(t *testing.T) {
		// CreatedByID 1 pod was updated to running status above
		pods, total, err := svc.ListPods(ctx, 1, []string{agentpod.StatusRunning}, 1, 10, 0)
		if err != nil {
			t.Fatalf("ListPods failed: %v", err)
		}
		if total != 1 {
			t.Errorf("Total = %d, want 1", total)
		}
		if len(pods) != 1 {
			t.Errorf("Pods count = %d, want 1", len(pods))
		}
	})

	t.Run("list filtered by creator with non-matching status", func(t *testing.T) {
		// CreatedByID 1 pod is running, not initializing
		pods, total, err := svc.ListPods(ctx, 1, []string{agentpod.StatusInitializing}, 1, 10, 0)
		if err != nil {
			t.Fatalf("ListPods failed: %v", err)
		}
		if total != 0 {
			t.Errorf("Total = %d, want 0", total)
		}
		if len(pods) != 0 {
			t.Errorf("Pods count = %d, want 0", len(pods))
		}
	})

	t.Run("list filtered by non-existent creator", func(t *testing.T) {
		pods, total, err := svc.ListPods(ctx, 1, nil, 999, 10, 0)
		if err != nil {
			t.Fatalf("ListPods failed: %v", err)
		}
		if total != 0 {
			t.Errorf("Total = %d, want 0", total)
		}
		if len(pods) != 0 {
			t.Errorf("Pods count = %d, want 0", len(pods))
		}
	})
}

func TestListActivePods(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	// Create pods with different statuses
	for i := 0; i < 3; i++ {
		req := &CreatePodRequest{
			OrganizationID: 1,
			RunnerID:       1,
			CreatedByID:    int64(i + 1),
		}
		sess, _ := svc.CreatePod(ctx, req)
		if i == 0 {
			svc.UpdatePodStatus(ctx, sess.PodKey, agentpod.StatusRunning)
		} else if i == 1 {
			svc.UpdatePodStatus(ctx, sess.PodKey, agentpod.StatusTerminated)
		}
		// i == 2 remains initializing (active)
	}

	pods, err := svc.ListActivePods(ctx, 1)
	if err != nil {
		t.Fatalf("ListActivePods failed: %v", err)
	}

	// Should have 2 active pods (running and initializing)
	if len(pods) != 2 {
		t.Errorf("Active pods count = %d, want 2", len(pods))
	}
}

func TestListByRunner(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	// Create pods
	for i := 0; i < 3; i++ {
		req := &CreatePodRequest{
			OrganizationID: 1,
			RunnerID:       1,
			CreatedByID:    1,
		}
		sess, _ := svc.CreatePod(ctx, req)
		if i == 0 {
			svc.UpdatePodStatus(ctx, sess.PodKey, agentpod.StatusRunning)
		}
	}

	t.Run("all pods", func(t *testing.T) {
		pods, err := svc.ListByRunner(ctx, 1, "")
		if err != nil {
			t.Fatalf("ListByRunner failed: %v", err)
		}
		if len(pods) != 3 {
			t.Errorf("Pods count = %d, want 3", len(pods))
		}
	})

	t.Run("running pods only", func(t *testing.T) {
		pods, err := svc.ListByRunner(ctx, 1, agentpod.StatusRunning)
		if err != nil {
			t.Fatalf("ListByRunner failed: %v", err)
		}
		if len(pods) != 1 {
			t.Errorf("Pods count = %d, want 1", len(pods))
		}
	})
}

func TestListByTicket(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	ticketID := int64(100)
	for i := 0; i < 2; i++ {
		req := &CreatePodRequest{
			OrganizationID: 1,
			RunnerID:       1,
			CreatedByID:    1,
			TicketID:       &ticketID,
		}
		svc.CreatePod(ctx, req)
	}

	pods, err := svc.ListByTicket(ctx, ticketID)
	if err != nil {
		t.Fatalf("ListByTicket failed: %v", err)
	}
	if len(pods) != 2 {
		t.Errorf("Pods count = %d, want 2", len(pods))
	}
}

func TestGetPodsByTicket(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	ticketID := int64(42)
	// Create pods for ticket
	for i := 0; i < 3; i++ {
		req := &CreatePodRequest{
			OrganizationID: 1,
			RunnerID:       1,
			CreatedByID:    1,
			TicketID:       &ticketID,
		}
		svc.CreatePod(ctx, req)
	}

	// Create pod without ticket
	req := &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		CreatedByID:    1,
	}
	svc.CreatePod(ctx, req)

	pods, err := svc.GetPodsByTicket(ctx, ticketID)
	if err != nil {
		t.Fatalf("GetPodsByTicket failed: %v", err)
	}
	if len(pods) != 3 {
		t.Errorf("Pods count = %d, want 3", len(pods))
	}
}
