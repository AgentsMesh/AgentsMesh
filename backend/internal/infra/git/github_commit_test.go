package git

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestGitHubGetCommit(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"sha": "abc123",
				"commit": map[string]interface{}{
					"message": "Test commit",
					"author": map[string]interface{}{
						"name":  "Test Author",
						"email": "author@example.com",
						"date":  time.Now().Format(time.RFC3339),
					},
				},
			})
		})
		defer server.Close()

		commit, err := provider.GetCommit(ctx, "owner/repo", "abc123")
		if err != nil {
			t.Fatalf("GetCommit failed: %v", err)
		}
		if commit.SHA != "abc123" {
			t.Errorf("commit.SHA = %s, want abc123", commit.SHA)
		}
		if commit.Message != "Test commit" {
			t.Errorf("commit.Message = %s, want Test commit", commit.Message)
		}
	})

	t.Run("not found", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		defer server.Close()

		_, err := provider.GetCommit(ctx, "owner/repo", "nonexistent")
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestGitHubListCommits(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"sha": "abc123",
					"commit": map[string]interface{}{
						"message": "Commit 1",
						"author": map[string]interface{}{
							"name":  "Author 1",
							"email": "author1@example.com",
							"date":  time.Now().Format(time.RFC3339),
						},
					},
				},
				{
					"sha": "def456",
					"commit": map[string]interface{}{
						"message": "Commit 2",
						"author": map[string]interface{}{
							"name":  "Author 2",
							"email": "author2@example.com",
							"date":  time.Now().Format(time.RFC3339),
						},
					},
				},
			})
		})
		defer server.Close()

		commits, err := provider.ListCommits(ctx, "owner/repo", "main", 1, 20)
		if err != nil {
			t.Fatalf("ListCommits failed: %v", err)
		}
		if len(commits) != 2 {
			t.Errorf("len(commits) = %d, want 2", len(commits))
		}
	})
}

