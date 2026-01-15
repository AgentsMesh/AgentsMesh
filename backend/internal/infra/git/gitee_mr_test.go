package git

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestGiteeMergeRequestOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("get merge request", func(t *testing.T) {
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
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
				"html_url": "https://gitee.com/owner/repo/pulls/10",
				"user": map[string]interface{}{
					"id":         123,
					"login":      "testuser",
					"name":       "Test User",
					"avatar_url": "https://gitee.com/avatar.png",
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
	})

	t.Run("list merge requests opened state", func(t *testing.T) {
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
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

	t.Run("list merge requests merged state", func(t *testing.T) {
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		})
		defer server.Close()

		mrs, err := provider.ListMergeRequests(ctx, "owner/repo", "merged", 1, 20)
		if err != nil {
			t.Fatalf("ListMergeRequests failed: %v", err)
		}
		if len(mrs) != 0 {
			t.Errorf("len(mrs) = %d, want 0", len(mrs))
		}
	})

	t.Run("list merge requests closed state", func(t *testing.T) {
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		})
		defer server.Close()

		mrs, err := provider.ListMergeRequests(ctx, "owner/repo", "closed", 1, 20)
		if err != nil {
			t.Fatalf("ListMergeRequests failed: %v", err)
		}
		if len(mrs) != 0 {
			t.Errorf("len(mrs) = %d, want 0", len(mrs))
		}
	})

	t.Run("list merge requests by branch", func(t *testing.T) {
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
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

	t.Run("create merge request", func(t *testing.T) {
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
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

	t.Run("update merge request", func(t *testing.T) {
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
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

	t.Run("merge merge request", func(t *testing.T) {
		callCount := 0
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				// Merge request
				w.WriteHeader(http.StatusOK)
			} else {
				// Get PR after merge
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":         1001,
					"number":     10,
					"title":      "Merged PR",
					"state":      "merged",
					"head":       map[string]interface{}{"ref": "feature"},
					"base":       map[string]interface{}{"ref": "main"},
					"user":       map[string]interface{}{"id": 123, "login": "testuser"},
					"merged_at":  time.Now().Format(time.RFC3339),
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
		if mr.State != "merged" {
			t.Errorf("mr.State = %s, want merged", mr.State)
		}
	})

	t.Run("close merge request", func(t *testing.T) {
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
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
