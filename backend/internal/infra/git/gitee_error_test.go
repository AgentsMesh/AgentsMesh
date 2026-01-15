package git

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGiteeDoRequestErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("http client error", func(t *testing.T) {
		provider, _ := NewGiteeProvider("http://invalid-host-that-does-not-exist:99999", "test-token")
		_, err := provider.GetCurrentUser(ctx)
		if err == nil {
			t.Error("expected error for invalid host")
		}
	})

	t.Run("doRequest with query params in path", func(t *testing.T) {
		// Test the "?" branch in doRequest (path already has query params)
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		})
		defer server.Close()

		_, err := provider.ListCommits(ctx, "owner/repo", "main", 1, 20)
		if err != nil {
			t.Fatalf("ListCommits failed: %v", err)
		}
	})
}

func TestGiteeErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("get current user HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.GetCurrentUser(ctx)
		if err == nil {
			t.Error("expected error for HTTP 500")
		}
	})

	t.Run("get current user invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.GetCurrentUser(ctx)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("get project HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.GetProject(ctx, "owner/repo")
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})

	t.Run("get project invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.GetProject(ctx, "owner/repo")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("list projects invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not an array"))
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.ListProjects(ctx, 1, 20)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("search projects invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not an array"))
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.SearchProjects(ctx, "test", 1, 20)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("list branches invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not an array"))
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.ListBranches(ctx, "owner/repo")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("get branch invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.GetBranch(ctx, "owner/repo", "main")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("create branch invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.CreateBranch(ctx, "owner/repo", "feature", "abc123")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("delete branch HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		err := provider.DeleteBranch(ctx, "owner/repo", "protected-branch")
		if err == nil {
			t.Error("expected error for HTTP 403")
		}
	})

	t.Run("get branch HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.GetBranch(ctx, "owner/repo", "nonexistent")
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})

	t.Run("create branch HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.CreateBranch(ctx, "owner/repo", "feature", "main")
		if err == nil {
			t.Error("expected error for HTTP 401")
		}
	})

	t.Run("get commit HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.GetCommit(ctx, "owner/repo", "abc123")
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})

	t.Run("list commits HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.ListCommits(ctx, "owner/repo", "main", 1, 20)
		if err == nil {
			t.Error("expected error for HTTP 403")
		}
	})

	t.Run("get file content HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.GetFileContent(ctx, "owner/repo", "README.md", "main")
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})

	t.Run("get commit invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.GetCommit(ctx, "owner/repo", "abc123")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("list commits invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not an array"))
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.ListCommits(ctx, "owner/repo", "main", 1, 20)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("get file content invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.GetFileContent(ctx, "owner/repo", "README.md", "main")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("register webhook invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.RegisterWebhook(ctx, "owner/repo", &WebhookConfig{
			URL:    "https://example.com/webhook",
			Secret: "secret",
			Events: []string{"push"},
		})
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("delete webhook HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		err := provider.DeleteWebhook(ctx, "owner/repo", "12345")
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})
}

func TestGiteeMergeRequestHttpErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("get merge request HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.GetMergeRequest(ctx, "owner/repo", 1)
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})

	t.Run("create merge request HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.CreateMergeRequest(ctx, &CreateMRRequest{
			ProjectID:    "owner/repo",
			Title:        "Test PR",
			SourceBranch: "feature",
			TargetBranch: "main",
		})
		if err == nil {
			t.Error("expected error for HTTP 401")
		}
	})

	t.Run("update merge request HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.UpdateMergeRequest(ctx, "owner/repo", 1, "Title", "Body")
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})

	t.Run("close merge request HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.CloseMergeRequest(ctx, "owner/repo", 1)
		if err == nil {
			t.Error("expected error for HTTP 403")
		}
	})

	t.Run("merge merge request HTTP error on merge", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.MergeMergeRequest(ctx, "owner/repo", 1)
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})
}

func TestGiteeMergeRequestErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("get merge request invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.GetMergeRequest(ctx, "owner/repo", 1)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("list merge requests invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not an array"))
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.ListMergeRequests(ctx, "owner/repo", "opened", 1, 20)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("list merge requests by branch invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not an array"))
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.ListMergeRequestsByBranch(ctx, "owner/repo", "feature", "opened")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("list merge requests by branch with closed state", func(t *testing.T) {
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("state") != "closed" {
				t.Errorf("unexpected state: %s", r.URL.Query().Get("state"))
			}
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		})
		defer server.Close()

		_, err := provider.ListMergeRequestsByBranch(ctx, "owner/repo", "feature", "closed")
		if err != nil {
			t.Fatalf("ListMergeRequestsByBranch failed: %v", err)
		}
	})

	t.Run("list merge requests by branch with merged state", func(t *testing.T) {
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("state") != "merged" {
				t.Errorf("unexpected state: %s", r.URL.Query().Get("state"))
			}
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		})
		defer server.Close()

		_, err := provider.ListMergeRequestsByBranch(ctx, "owner/repo", "feature", "merged")
		if err != nil {
			t.Fatalf("ListMergeRequestsByBranch failed: %v", err)
		}
	})

	t.Run("list merge requests by branch with all state", func(t *testing.T) {
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("state") != "all" {
				t.Errorf("unexpected state: %s", r.URL.Query().Get("state"))
			}
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		})
		defer server.Close()

		_, err := provider.ListMergeRequestsByBranch(ctx, "owner/repo", "feature", "all")
		if err != nil {
			t.Fatalf("ListMergeRequestsByBranch failed: %v", err)
		}
	})

	t.Run("create merge request invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.CreateMergeRequest(ctx, &CreateMRRequest{
			ProjectID:    "owner/repo",
			Title:        "Test PR",
			SourceBranch: "feature",
			TargetBranch: "main",
		})
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("update merge request invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.UpdateMergeRequest(ctx, "owner/repo", 1, "Updated Title", "Updated Body")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("merge merge request HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.MergeMergeRequest(ctx, "owner/repo", 1)
		if err == nil {
			t.Error("expected error for HTTP 409")
		}
	})

	t.Run("close merge request invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGiteeProvider(server.URL, "test-token")
		_, err := provider.CloseMergeRequest(ctx, "owner/repo", 1)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestGiteeMergeRequestWithMergedAt(t *testing.T) {
	ctx := context.Background()

	t.Run("get merge request with merged_at field", func(t *testing.T) {
		mergedAt := time.Now().Add(-1 * time.Hour)
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":         1001,
				"number":     10,
				"title":      "Merged PR",
				"body":       "Description",
				"head":       map[string]interface{}{"ref": "feature"},
				"base":       map[string]interface{}{"ref": "main"},
				"state":      "merged",
				"html_url":   "https://gitee.com/owner/repo/pulls/10",
				"merged_at":  mergedAt.Format(time.RFC3339),
				"user":       map[string]interface{}{"id": 123, "login": "testuser"},
				"created_at": time.Now().Format(time.RFC3339),
				"updated_at": time.Now().Format(time.RFC3339),
			})
		})
		defer server.Close()

		mr, err := provider.GetMergeRequest(ctx, "owner/repo", 10)
		if err != nil {
			t.Fatalf("GetMergeRequest failed: %v", err)
		}
		if mr.MergedAt == nil {
			t.Error("expected merged_at to be set")
		}
	})

	t.Run("list merge requests with merged_at field", func(t *testing.T) {
		mergedAt := time.Now().Add(-1 * time.Hour)
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":         1001,
					"number":     10,
					"title":      "Merged PR",
					"body":       "Description",
					"head":       map[string]interface{}{"ref": "feature"},
					"base":       map[string]interface{}{"ref": "main"},
					"state":      "merged",
					"merged_at":  mergedAt.Format(time.RFC3339),
					"user":       map[string]interface{}{"id": 123, "login": "testuser"},
					"created_at": time.Now().Format(time.RFC3339),
					"updated_at": time.Now().Format(time.RFC3339),
				},
			})
		})
		defer server.Close()

		mrs, err := provider.ListMergeRequests(ctx, "owner/repo", "merged", 1, 20)
		if err != nil {
			t.Fatalf("ListMergeRequests failed: %v", err)
		}
		if len(mrs) != 1 || mrs[0].MergedAt == nil {
			t.Error("expected merged_at to be set")
		}
	})
}
