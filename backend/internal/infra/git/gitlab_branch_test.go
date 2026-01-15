package git

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestGitLabBranchOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("list branches", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"name": "main",
					"commit": map[string]interface{}{
						"id": "abc123",
					},
					"protected": true,
					"default":   true,
				},
				{
					"name": "develop",
					"commit": map[string]interface{}{
						"id": "def456",
					},
					"protected": false,
					"default":   false,
				},
			})
		})
		defer server.Close()

		branches, err := provider.ListBranches(ctx, "owner/repo")
		if err != nil {
			t.Fatalf("ListBranches failed: %v", err)
		}
		if len(branches) != 2 {
			t.Errorf("len(branches) = %d, want 2", len(branches))
		}
	})

	t.Run("get branch", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name": "main",
				"commit": map[string]interface{}{
					"id": "abc123",
				},
				"protected": true,
				"default":   true,
			})
		})
		defer server.Close()

		branch, err := provider.GetBranch(ctx, "owner/repo", "main")
		if err != nil {
			t.Fatalf("GetBranch failed: %v", err)
		}
		if branch.Name != "main" {
			t.Errorf("branch.Name = %s, want main", branch.Name)
		}
	})

	t.Run("get branch not found", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		defer server.Close()

		_, err := provider.GetBranch(ctx, "owner/repo", "nonexistent")
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("create branch", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name": "feature",
				"commit": map[string]interface{}{
					"id": "abc123",
				},
			})
		})
		defer server.Close()

		branch, err := provider.CreateBranch(ctx, "owner/repo", "feature", "main")
		if err != nil {
			t.Fatalf("CreateBranch failed: %v", err)
		}
		if branch.Name != "feature" {
			t.Errorf("branch.Name = %s, want feature", branch.Name)
		}
	})

	t.Run("delete branch", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})
		defer server.Close()

		err := provider.DeleteBranch(ctx, "owner/repo", "feature")
		if err != nil {
			t.Fatalf("DeleteBranch failed: %v", err)
		}
	})
}
