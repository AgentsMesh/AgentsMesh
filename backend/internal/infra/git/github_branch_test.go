package git

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestGitHubListBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		callCount := 0
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if r.URL.Path == "/repos/owner/repo/branches" {
				json.NewEncoder(w).Encode([]map[string]interface{}{
					{
						"name": "main",
						"commit": map[string]interface{}{
							"sha": "abc123",
						},
						"protected": true,
					},
					{
						"name": "develop",
						"commit": map[string]interface{}{
							"sha": "def456",
						},
						"protected": false,
					},
				})
			} else if r.URL.Path == "/repos/owner/repo" {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"default_branch": "main",
					"created_at":     time.Now().Format(time.RFC3339),
					"updated_at":     time.Now().Format(time.RFC3339),
				})
			}
		})
		defer server.Close()

		branches, err := provider.ListBranches(ctx, "owner/repo")
		if err != nil {
			t.Fatalf("ListBranches failed: %v", err)
		}
		if len(branches) != 2 {
			t.Errorf("len(branches) = %d, want 2", len(branches))
		}
		// Check default branch
		for _, b := range branches {
			if b.Name == "main" && !b.Default {
				t.Error("main branch should be default")
			}
		}
	})
}

func TestGitHubGetBranch(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/repos/owner/repo/branches/main" {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"name": "main",
					"commit": map[string]interface{}{
						"sha": "abc123",
					},
					"protected": true,
				})
			} else if r.URL.Path == "/repos/owner/repo" {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"default_branch": "main",
					"created_at":     time.Now().Format(time.RFC3339),
					"updated_at":     time.Now().Format(time.RFC3339),
				})
			}
		})
		defer server.Close()

		branch, err := provider.GetBranch(ctx, "owner/repo", "main")
		if err != nil {
			t.Fatalf("GetBranch failed: %v", err)
		}
		if branch.Name != "main" {
			t.Errorf("branch.Name = %s, want main", branch.Name)
		}
		if !branch.Protected {
			t.Error("branch should be protected")
		}
	})

	t.Run("not found", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		defer server.Close()

		_, err := provider.GetBranch(ctx, "owner/repo", "nonexistent")
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestGitHubCreateBranch(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" && r.URL.Path == "/repos/owner/repo/git/refs/heads/main" {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"ref": "refs/heads/main",
					"object": map[string]interface{}{
						"sha": "abc123",
					},
				})
			} else if r.Method == "POST" && r.URL.Path == "/repos/owner/repo/git/refs" {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"ref": "refs/heads/feature",
					"object": map[string]interface{}{
						"sha": "abc123",
					},
				})
			}
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
}

func TestGitHubDeleteBranch(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "DELETE" {
				t.Errorf("unexpected method: %s", r.Method)
			}
			w.WriteHeader(http.StatusNoContent)
		})
		defer server.Close()

		err := provider.DeleteBranch(ctx, "owner/repo", "feature")
		if err != nil {
			t.Fatalf("DeleteBranch failed: %v", err)
		}
	})
}
