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

	for i := 0; i < 5; i++ {
		req := &CreatePodRequest{OrganizationID: 1, RunnerID: 1, CreatedByID: int64(i + 1)}
		svc.CreatePod(ctx, req)
	}

	t.Run("list all", func(t *testing.T) {
		pods, total, err := svc.ListPods(ctx, 1, agentpod.PodListQuery{Limit: 10})
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
		pods, total, err := svc.ListPods(ctx, 1, agentpod.PodListQuery{Limit: 2})
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
		pods, _, err := svc.ListPods(ctx, 1, agentpod.PodListQuery{
			Statuses: []string{agentpod.StatusInitializing}, Limit: 10,
		})
		if err != nil {
			t.Fatalf("ListPods failed: %v", err)
		}
		if len(pods) != 5 {
			t.Errorf("Pods count = %d, want 5", len(pods))
		}
	})

	allPods, _, _ := svc.ListPods(ctx, 1, agentpod.PodListQuery{Limit: 10})
	if len(allPods) >= 3 {
		svc.UpdatePodStatus(ctx, allPods[0].PodKey, agentpod.StatusRunning)
		svc.UpdatePodStatus(ctx, allPods[1].PodKey, agentpod.StatusTerminated)
	}

	t.Run("list with multiple status filter", func(t *testing.T) {
		pods, total, err := svc.ListPods(ctx, 1, agentpod.PodListQuery{
			Statuses: []string{agentpod.StatusRunning, agentpod.StatusInitializing}, Limit: 10,
		})
		if err != nil {
			t.Fatalf("ListPods failed: %v", err)
		}
		if total != 4 {
			t.Errorf("Total = %d, want 4", total)
		}
		if len(pods) != 4 {
			t.Errorf("Pods count = %d, want 4", len(pods))
		}
	})

	t.Run("list with non-matching status filter", func(t *testing.T) {
		pods, total, err := svc.ListPods(ctx, 1, agentpod.PodListQuery{
			Statuses: []string{agentpod.StatusPaused}, Limit: 10,
		})
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
		pods, total, err := svc.ListPods(ctx, 1, agentpod.PodListQuery{CreatedByID: 1, Limit: 10})
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
		pods, total, err := svc.ListPods(ctx, 1, agentpod.PodListQuery{
			Statuses: []string{agentpod.StatusRunning}, CreatedByID: 1, Limit: 10,
		})
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
		pods, total, err := svc.ListPods(ctx, 1, agentpod.PodListQuery{
			Statuses: []string{agentpod.StatusInitializing}, CreatedByID: 1, Limit: 10,
		})
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
		pods, total, err := svc.ListPods(ctx, 1, agentpod.PodListQuery{CreatedByID: 999, Limit: 10})
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

	for i := 0; i < 3; i++ {
		req := &CreatePodRequest{OrganizationID: 1, RunnerID: 1, CreatedByID: int64(i + 1)}
		sess, _ := svc.CreatePod(ctx, req)
		if i == 0 {
			svc.UpdatePodStatus(ctx, sess.PodKey, agentpod.StatusRunning)
		} else if i == 1 {
			svc.UpdatePodStatus(ctx, sess.PodKey, agentpod.StatusTerminated)
		}
	}

	pods, err := svc.ListActivePods(ctx, 1)
	if err != nil {
		t.Fatalf("ListActivePods failed: %v", err)
	}
	if len(pods) != 2 {
		t.Errorf("Active pods count = %d, want 2", len(pods))
	}
}

func TestListByRunner(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		req := &CreatePodRequest{OrganizationID: 1, RunnerID: 1, CreatedByID: 1}
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
		req := &CreatePodRequest{OrganizationID: 1, RunnerID: 1, CreatedByID: 1, TicketID: &ticketID}
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
	for i := 0; i < 3; i++ {
		req := &CreatePodRequest{OrganizationID: 1, RunnerID: 1, CreatedByID: 1, TicketID: &ticketID}
		svc.CreatePod(ctx, req)
	}
	req := &CreatePodRequest{OrganizationID: 1, RunnerID: 1, CreatedByID: 1}
	svc.CreatePod(ctx, req)

	pods, err := svc.GetPodsByTicket(ctx, ticketID)
	if err != nil {
		t.Fatalf("GetPodsByTicket failed: %v", err)
	}
	if len(pods) != 3 {
		t.Errorf("Pods count = %d, want 3", len(pods))
	}
}

func TestListActiveResumedBy(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	source, err := svc.CreatePod(ctx, &CreatePodRequest{OrganizationID: 1, RunnerID: 1, CreatedByID: 1})
	if err != nil {
		t.Fatalf("CreatePod(source) failed: %v", err)
	}
	if err := svc.UpdatePodStatus(ctx, source.PodKey, agentpod.StatusTerminated); err != nil {
		t.Fatalf("UpdatePodStatus failed: %v", err)
	}

	resumed, err := svc.CreatePod(ctx, &CreatePodRequest{
		OrganizationID: 1, RunnerID: 1, CreatedByID: 1, SourcePodKey: source.PodKey,
	})
	if err != nil {
		t.Fatalf("CreatePod(resume) failed: %v", err)
	}

	t.Run("active resume pod is reported against its source", func(t *testing.T) {
		m, err := svc.ListActiveResumedBy(ctx, []string{source.PodKey})
		if err != nil {
			t.Fatalf("ListActiveResumedBy failed: %v", err)
		}
		if m[source.PodKey] != resumed.PodKey {
			t.Errorf("resumed_by = %q, want %q", m[source.PodKey], resumed.PodKey)
		}
	})

	t.Run("terminated resume pod frees the slot", func(t *testing.T) {
		if err := svc.UpdatePodStatus(ctx, resumed.PodKey, agentpod.StatusTerminated); err != nil {
			t.Fatalf("UpdatePodStatus failed: %v", err)
		}
		m, err := svc.ListActiveResumedBy(ctx, []string{source.PodKey})
		if err != nil {
			t.Fatalf("ListActiveResumedBy failed: %v", err)
		}
		if len(m) != 0 {
			t.Errorf("map = %v, want empty", m)
		}
	})

	t.Run("empty input short-circuits", func(t *testing.T) {
		m, err := svc.ListActiveResumedBy(ctx, nil)
		if err != nil {
			t.Fatalf("ListActiveResumedBy failed: %v", err)
		}
		if len(m) != 0 {
			t.Errorf("map = %v, want empty", m)
		}
	})
}
