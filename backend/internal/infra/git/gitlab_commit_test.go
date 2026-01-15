package git

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestGitLabCommitOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("get commit", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":           "abc123",
				"message":      "Test commit",
				"author_name":  "Test Author",
				"author_email": "author@example.com",
				"created_at":   time.Now().Format(time.RFC3339),
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
	})

	t.Run("get commit not found", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		defer server.Close()

		_, err := provider.GetCommit(ctx, "owner/repo", "nonexistent")
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("list commits", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":           "abc123",
					"message":      "Commit 1",
					"author_name":  "Author 1",
					"author_email": "author1@example.com",
					"created_at":   time.Now().Format(time.RFC3339),
				},
				{
					"id":           "def456",
					"message":      "Commit 2",
					"author_name":  "Author 2",
					"author_email": "author2@example.com",
					"created_at":   time.Now().Format(time.RFC3339),
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
