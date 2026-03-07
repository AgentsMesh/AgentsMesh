package agentpod

import (
	"context"
	"testing"
	"time"
)

// ===========================================
// FindByBranchAndRepo Basic Tests
// ===========================================

func TestFindByBranchAndRepo(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	t.Run("find existing pod by branch and repo", func(t *testing.T) {
		// Create a pod with branch_name and repository_id
		req := &CreatePodRequest{
			OrganizationID: 1,
			RunnerID:       1,
			CreatedByID:    1,
			RepositoryID:   intPtr(100),
			BranchName:     strPtr("feature/AM-123-new-feature"),
		}
		created, err := svc.CreatePod(ctx, req)
		if err != nil {
			t.Fatalf("CreatePod failed: %v", err)
		}

		// Find the pod
		found, err := svc.FindByBranchAndRepo(ctx, 1, 100, "feature/AM-123-new-feature")
		if err != nil {
			t.Fatalf("FindByBranchAndRepo failed: %v", err)
		}

		if found.ID != created.ID {
			t.Errorf("Pod ID mismatch: expected %d, got %d", created.ID, found.ID)
		}
		if found.PodKey != created.PodKey {
			t.Errorf("PodKey mismatch: expected %s, got %s", created.PodKey, found.PodKey)
		}
	})

	t.Run("not found - wrong org", func(t *testing.T) {
		req := &CreatePodRequest{
			OrganizationID: 1,
			RunnerID:       1,
			CreatedByID:    1,
			RepositoryID:   intPtr(200),
			BranchName:     strPtr("fix/bug-456"),
		}
		_, err := svc.CreatePod(ctx, req)
		if err != nil {
			t.Fatalf("CreatePod failed: %v", err)
		}

		// Search with wrong organization ID
		_, err = svc.FindByBranchAndRepo(ctx, 999, 200, "fix/bug-456")
		if err == nil {
			t.Error("Expected error for wrong org ID")
		}
	})

	t.Run("not found - wrong repo", func(t *testing.T) {
		req := &CreatePodRequest{
			OrganizationID: 1,
			RunnerID:       1,
			CreatedByID:    1,
			RepositoryID:   intPtr(300),
			BranchName:     strPtr("fix/bug-789"),
		}
		_, err := svc.CreatePod(ctx, req)
		if err != nil {
			t.Fatalf("CreatePod failed: %v", err)
		}

		// Search with wrong repository ID
		_, err = svc.FindByBranchAndRepo(ctx, 1, 999, "fix/bug-789")
		if err == nil {
			t.Error("Expected error for wrong repo ID")
		}
	})

	t.Run("not found - wrong branch", func(t *testing.T) {
		req := &CreatePodRequest{
			OrganizationID: 1,
			RunnerID:       1,
			CreatedByID:    1,
			RepositoryID:   intPtr(400),
			BranchName:     strPtr("feature/specific-branch"),
		}
		_, err := svc.CreatePod(ctx, req)
		if err != nil {
			t.Fatalf("CreatePod failed: %v", err)
		}

		// Search with wrong branch name
		_, err = svc.FindByBranchAndRepo(ctx, 1, 400, "feature/wrong-branch")
		if err == nil {
			t.Error("Expected error for wrong branch name")
		}
	})

	t.Run("not found - non-existent combination", func(t *testing.T) {
		_, err := svc.FindByBranchAndRepo(ctx, 9999, 9999, "non-existent-branch")
		if err == nil {
			t.Error("Expected error for non-existent combination")
		}
	})
}

func TestFindByBranchAndRepo_ReturnsMostRecent(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	orgID := int64(1)
	repoID := int64(500)
	branchName := "feature/AM-999-test"

	// Create first pod
	req1 := &CreatePodRequest{
		OrganizationID: orgID,
		RunnerID:       1,
		CreatedByID:    1,
		RepositoryID:   intPtr(repoID),
		BranchName:     strPtr(branchName),
	}
	pod1, err := svc.CreatePod(ctx, req1)
	if err != nil {
		t.Fatalf("CreatePod (first) failed: %v", err)
	}

	// Create second pod with same branch/repo
	req2 := &CreatePodRequest{
		OrganizationID: orgID,
		RunnerID:       1,
		CreatedByID:    1,
		RepositoryID:   intPtr(repoID),
		BranchName:     strPtr(branchName),
	}
	pod2, err := svc.CreatePod(ctx, req2)
	if err != nil {
		t.Fatalf("CreatePod (second) failed: %v", err)
	}

	// Explicitly set created_at timestamps to ensure ordering
	// pod1 is older, pod2 is newer
	oldTime := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	newTime := time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC)
	db.Exec("UPDATE pods SET created_at = ? WHERE id = ?", oldTime, pod1.ID)
	db.Exec("UPDATE pods SET created_at = ? WHERE id = ?", newTime, pod2.ID)

	// Find should return the most recent (pod2)
	found, err := svc.FindByBranchAndRepo(ctx, orgID, repoID, branchName)
	if err != nil {
		t.Fatalf("FindByBranchAndRepo failed: %v", err)
	}

	if found.ID != pod2.ID {
		t.Errorf("Expected most recent pod (ID=%d), got ID=%d", pod2.ID, found.ID)
	}
	if found.ID == pod1.ID {
		t.Error("Should not return the older pod")
	}
}
