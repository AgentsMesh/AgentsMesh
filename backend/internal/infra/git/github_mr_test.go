package git

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestGitHubGetMergeRequest(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     1001,
				"number": 10,
				"title":  "Test PR",
				"body":   "PR description",
				"head": map[string]interface{}{
					"ref": "feature",
				},
				"base": map[string]interface{}{
					"ref": "main",
				},
				"state":    "open",
				"html_url": "https://github.com/owner/repo/pull/10",
				"user": map[string]interface{}{
					"id":         123,
					"login":      "testuser",
					"avatar_url": "https://github.com/avatar.png",
				},
				"created_at": time.Now().Format(time.RFC3339),
				"updated_at": time.Now().Format(time.RFC3339),
			})
		})
		defer server.Close()

		mr, err := provider.GetMergeRequest(ctx, "owner/repo", 10)
		if err != nil {
			t.Fatalf("GetMergeRequest failed: %v", err)
		}
		if mr.IID != 10 {
			t.Errorf("mr.IID = %d, want 10", mr.IID)
		}
		if mr.Title != "Test PR" {
			t.Errorf("mr.Title = %s, want Test PR", mr.Title)
		}
	})

	t.Run("not found", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		defer server.Close()

		_, err := provider.GetMergeRequest(ctx, "owner/repo", 999)
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestGitHubListMergeRequests(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("state") != "open" {
				t.Errorf("unexpected state: %s", r.URL.Query().Get("state"))
			}

			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":     1,
					"number": 1,
					"title":  "PR 1",
					"head":   map[string]interface{}{"ref": "feature1"},
					"base":   map[string]interface{}{"ref": "main"},
					"state":  "open",
					"user": map[string]interface{}{
						"id":    123,
						"login": "user1",
					},
					"created_at": time.Now().Format(time.RFC3339),
					"updated_at": time.Now().Format(time.RFC3339),
				},
			})
		})
		defer server.Close()

		mrs, err := provider.ListMergeRequests(ctx, "owner/repo", "opened", 1, 20)
		if err != nil {
			t.Fatalf("ListMergeRequests failed: %v", err)
		}
		if len(mrs) != 1 {
			t.Errorf("len(mrs) = %d, want 1", len(mrs))
		}
	})

	t.Run("merged state", func(t *testing.T) {
		mergedAt := time.Now()
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":        1,
					"number":    1,
					"title":     "Merged PR",
					"head":      map[string]interface{}{"ref": "feature"},
					"base":      map[string]interface{}{"ref": "main"},
					"state":     "closed",
					"merged_at": mergedAt.Format(time.RFC3339),
					"user": map[string]interface{}{
						"id":    123,
						"login": "user1",
					},
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
		if len(mrs) != 1 {
			t.Errorf("len(mrs) = %d, want 1", len(mrs))
		}
		if mrs[0].State != "merged" {
			t.Errorf("mrs[0].State = %s, want merged", mrs[0].State)
		}
	})
}

func TestGitHubListMergeRequestsByBranch(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("head") == "" {
				t.Error("expected head parameter")
			}

			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":               1,
					"number":           1,
					"title":            "PR from feature",
					"head":             map[string]interface{}{"ref": "feature"},
					"base":             map[string]interface{}{"ref": "main"},
					"state":            "open",
					"merge_commit_sha": "abc123",
					"user": map[string]interface{}{
						"id":    123,
						"login": "user1",
					},
					"created_at": time.Now().Format(time.RFC3339),
					"updated_at": time.Now().Format(time.RFC3339),
				},
			})
		})
		defer server.Close()

		mrs, err := provider.ListMergeRequestsByBranch(ctx, "owner/repo", "feature", "opened")
		if err != nil {
			t.Fatalf("ListMergeRequestsByBranch failed: %v", err)
		}
		if len(mrs) != 1 {
			t.Errorf("len(mrs) = %d, want 1", len(mrs))
		}
	})
}

func TestGitHubCreateMergeRequest(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("unexpected method: %s", r.Method)
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     1001,
				"number": 11,
				"title":  "New Feature",
				"body":   "Description",
				"head":   map[string]interface{}{"ref": "feature"},
				"base":   map[string]interface{}{"ref": "main"},
				"state":  "open",
				"user": map[string]interface{}{
					"id":    123,
					"login": "testuser",
				},
				"created_at": time.Now().Format(time.RFC3339),
				"updated_at": time.Now().Format(time.RFC3339),
			})
		})
		defer server.Close()

		mr, err := provider.CreateMergeRequest(ctx, &CreateMRRequest{
			ProjectID:    "owner/repo",
			Title:        "New Feature",
			Description:  "Description",
			SourceBranch: "feature",
			TargetBranch: "main",
		})
		if err != nil {
			t.Fatalf("CreateMergeRequest failed: %v", err)
		}
		if mr.Title != "New Feature" {
			t.Errorf("mr.Title = %s, want New Feature", mr.Title)
		}
	})
}

func TestGitHubUpdateMergeRequest(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "PATCH" {
				t.Errorf("unexpected method: %s", r.Method)
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     1001,
				"number": 10,
				"title":  "Updated Title",
				"body":   "Updated Description",
				"head":   map[string]interface{}{"ref": "feature"},
				"base":   map[string]interface{}{"ref": "main"},
				"state":  "open",
				"user": map[string]interface{}{
					"id":    123,
					"login": "testuser",
				},
				"created_at": time.Now().Format(time.RFC3339),
				"updated_at": time.Now().Format(time.RFC3339),
			})
		})
		defer server.Close()

		mr, err := provider.UpdateMergeRequest(ctx, "owner/repo", 10, "Updated Title", "Updated Description")
		if err != nil {
			t.Fatalf("UpdateMergeRequest failed: %v", err)
		}
		if mr.Title != "Updated Title" {
			t.Errorf("mr.Title = %s, want Updated Title", mr.Title)
		}
	})
}

func TestGitHubMergeMergeRequest(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		mergedAt := time.Now()
		callCount := 0
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				// Merge request
				json.NewEncoder(w).Encode(map[string]interface{}{
					"merged": true,
				})
			} else {
				// Get PR after merge - with merged_at, state becomes "merged"
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":         1001,
					"number":     10,
					"title":      "Merged PR",
					"state":      "closed",
					"head":       map[string]interface{}{"ref": "feature"},
					"base":       map[string]interface{}{"ref": "main"},
					"user":       map[string]interface{}{"id": 123, "login": "testuser"},
					"merged_at":  mergedAt.Format(time.RFC3339),
					"created_at": time.Now().Format(time.RFC3339),
					"updated_at": time.Now().Format(time.RFC3339),
				})
			}
		})
		defer server.Close()

		mr, err := provider.MergeMergeRequest(ctx, "owner/repo", 10)
		if err != nil {
			t.Fatalf("MergeMergeRequest failed: %v", err)
		}
		// When merged_at is present, parsePullRequest sets state to "merged"
		if mr.State != "merged" {
			t.Errorf("mr.State = %s, want merged", mr.State)
		}
	})
}

func TestGitHubCloseMergeRequest(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":         1001,
				"number":     10,
				"title":      "Closed PR",
				"state":      "closed",
				"head":       map[string]interface{}{"ref": "feature"},
				"base":       map[string]interface{}{"ref": "main"},
				"user":       map[string]interface{}{"id": 123, "login": "testuser"},
				"created_at": time.Now().Format(time.RFC3339),
				"updated_at": time.Now().Format(time.RFC3339),
			})
		})
		defer server.Close()

		mr, err := provider.CloseMergeRequest(ctx, "owner/repo", 10)
		if err != nil {
			t.Fatalf("CloseMergeRequest failed: %v", err)
		}
		if mr.State != "closed" {
			t.Errorf("mr.State = %s, want closed", mr.State)
		}
	})
}
