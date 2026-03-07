package agentpod

import (
	"context"
	"testing"
)

// ===========================================
// FindByBranchAndRepo Isolation Tests
// Tests for isolation between branches, repos, and orgs
// ===========================================

func TestFindByBranchAndRepo_DifferentBranches(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	orgID := int64(1)
	repoID := int64(600)

	// Create pods with different branches on same repo
	branches := []string{"main", "develop", "feature/test-1", "feature/test-2"}
	podsByBranch := make(map[string]int64)

	for _, branch := range branches {
		req := &CreatePodRequest{
			OrganizationID: orgID,
			RunnerID:       1,
			CreatedByID:    1,
			RepositoryID:   intPtr(repoID),
			BranchName:     strPtr(branch),
		}
		pod, err := svc.CreatePod(ctx, req)
		if err != nil {
			t.Fatalf("CreatePod for branch %s failed: %v", branch, err)
		}
		podsByBranch[branch] = pod.ID
	}

	// Each branch should find its own pod
	for _, branch := range branches {
		found, err := svc.FindByBranchAndRepo(ctx, orgID, repoID, branch)
		if err != nil {
			t.Errorf("FindByBranchAndRepo for branch %s failed: %v", branch, err)
			continue
		}

		expectedID := podsByBranch[branch]
		if found.ID != expectedID {
			t.Errorf("Branch %s: expected pod ID %d, got %d", branch, expectedID, found.ID)
		}
	}
}

func TestFindByBranchAndRepo_DifferentRepos(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	orgID := int64(1)
	branchName := "main"

	// Create pods with same branch on different repos
	repoIDs := []int64{700, 701, 702}
	podsByRepo := make(map[int64]int64)

	for _, repoID := range repoIDs {
		req := &CreatePodRequest{
			OrganizationID: orgID,
			RunnerID:       1,
			CreatedByID:    1,
			RepositoryID:   intPtr(repoID),
			BranchName:     strPtr(branchName),
		}
		pod, err := svc.CreatePod(ctx, req)
		if err != nil {
			t.Fatalf("CreatePod for repo %d failed: %v", repoID, err)
		}
		podsByRepo[repoID] = pod.ID
	}

	// Each repo should find its own pod
	for _, repoID := range repoIDs {
		found, err := svc.FindByBranchAndRepo(ctx, orgID, repoID, branchName)
		if err != nil {
			t.Errorf("FindByBranchAndRepo for repo %d failed: %v", repoID, err)
			continue
		}

		expectedID := podsByRepo[repoID]
		if found.ID != expectedID {
			t.Errorf("Repo %d: expected pod ID %d, got %d", repoID, expectedID, found.ID)
		}
	}
}

func TestFindByBranchAndRepo_DifferentOrgs(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	repoID := int64(800)
	branchName := "feature/shared-branch-name"

	// Create pods with same branch/repo on different orgs
	orgIDs := []int64{1, 2, 3}
	podsByOrg := make(map[int64]int64)

	for _, orgID := range orgIDs {
		req := &CreatePodRequest{
			OrganizationID: orgID,
			RunnerID:       1,
			CreatedByID:    1,
			RepositoryID:   intPtr(repoID),
			BranchName:     strPtr(branchName),
		}
		pod, err := svc.CreatePod(ctx, req)
		if err != nil {
			t.Fatalf("CreatePod for org %d failed: %v", orgID, err)
		}
		podsByOrg[orgID] = pod.ID
	}

	// Each org should find its own pod
	for _, orgID := range orgIDs {
		found, err := svc.FindByBranchAndRepo(ctx, orgID, repoID, branchName)
		if err != nil {
			t.Errorf("FindByBranchAndRepo for org %d failed: %v", orgID, err)
			continue
		}

		expectedID := podsByOrg[orgID]
		if found.ID != expectedID {
			t.Errorf("Org %d: expected pod ID %d, got %d", orgID, expectedID, found.ID)
		}
	}
}
