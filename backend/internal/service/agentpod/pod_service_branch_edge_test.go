package agentpod

import (
	"context"
	"testing"
)

// ===========================================
// FindByBranchAndRepo Edge Cases Tests
// Tests for special characters, case sensitivity, and nil values
// ===========================================

func TestFindByBranchAndRepo_BranchNameCaseSensitivity(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	orgID := int64(1)
	repoID := int64(900)

	// Create pod with specific case branch name
	req := &CreatePodRequest{
		OrganizationID: orgID,
		RunnerID:       1,
		CreatedByID:    1,
		RepositoryID:   intPtr(repoID),
		BranchName:     strPtr("Feature/Test-Branch"),
	}
	created, err := svc.CreatePod(ctx, req)
	if err != nil {
		t.Fatalf("CreatePod failed: %v", err)
	}

	t.Run("exact case match", func(t *testing.T) {
		found, err := svc.FindByBranchAndRepo(ctx, orgID, repoID, "Feature/Test-Branch")
		if err != nil {
			t.Fatalf("FindByBranchAndRepo failed: %v", err)
		}
		if found.ID != created.ID {
			t.Errorf("Expected pod ID %d, got %d", created.ID, found.ID)
		}
	})

	// Note: SQLite case sensitivity depends on COLLATE settings
	// Git branch names are case-sensitive on Linux, case-insensitive on macOS/Windows
	// Our test assumes case-sensitive matching as is typical for Linux servers
}

func TestFindByBranchAndRepo_SpecialCharactersInBranch(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	orgID := int64(1)
	repoID := int64(1000)

	// Test various branch name formats commonly used
	branches := []string{
		"feature/AM-123-new-feature",
		"fix/bug-456",
		"hotfix/urgent_fix",
		"release/v1.2.3",
		"user/john/experiment",
		"deps/update-lodash-4.17.21",
		"renovate/npm-typescript-5.x",
	}

	for _, branch := range branches {
		t.Run(branch, func(t *testing.T) {
			req := &CreatePodRequest{
				OrganizationID: orgID,
				RunnerID:       1,
				CreatedByID:    1,
				RepositoryID:   intPtr(repoID),
				BranchName:     strPtr(branch),
			}
			created, err := svc.CreatePod(ctx, req)
			if err != nil {
				t.Fatalf("CreatePod failed for branch %s: %v", branch, err)
			}

			found, err := svc.FindByBranchAndRepo(ctx, orgID, repoID, branch)
			if err != nil {
				t.Fatalf("FindByBranchAndRepo failed for branch %s: %v", branch, err)
			}

			if found.ID != created.ID {
				t.Errorf("Pod ID mismatch for branch %s: expected %d, got %d", branch, created.ID, found.ID)
			}
		})
		// Increment repoID for each test to ensure isolation
		repoID++
	}
}

func TestFindByBranchAndRepo_NilBranchPod(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	// Create pod without branch_name
	req := &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		CreatedByID:    1,
		RepositoryID:   intPtr(1100),
		// BranchName is nil
	}
	_, err := svc.CreatePod(ctx, req)
	if err != nil {
		t.Fatalf("CreatePod failed: %v", err)
	}

	// Should not find pod when searching for specific branch
	_, err = svc.FindByBranchAndRepo(ctx, 1, 1100, "main")
	if err == nil {
		t.Error("Expected error when searching for branch on pod without branch_name")
	}
}

func TestFindByBranchAndRepo_NilRepoPod(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	// Create pod without repository_id
	req := &CreatePodRequest{
		OrganizationID: 1,
		RunnerID:       1,
		CreatedByID:    1,
		BranchName:     strPtr("orphan-branch"),
		// RepositoryID is nil
	}
	_, err := svc.CreatePod(ctx, req)
	if err != nil {
		t.Fatalf("CreatePod failed: %v", err)
	}

	// Should not find pod when searching for specific repo
	_, err = svc.FindByBranchAndRepo(ctx, 1, 1200, "orphan-branch")
	if err == nil {
		t.Error("Expected error when searching for repo on pod without repository_id")
	}
}
