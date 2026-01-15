package git

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestGitLabMergeRequestOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("get merge request", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":            1001,
				"iid":           10,
				"title":         "Test MR",
				"description":   "MR description",
				"source_branch": "feature",
				"target_branch": "main",
				"state":         "opened",
				"web_url":       "https://gitlab.com/owner/repo/-/merge_requests/10",
				"author": map[string]interface{}{
					"id":         123,
					"username":   "testuser",
					"name":       "Test User",
					"avatar_url": "https://gitlab.com/avatar.png",
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

	t.Run("get merge request not found", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		defer server.Close()

		_, err := provider.GetMergeRequest(ctx, "owner/repo", 999)
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("list merge requests", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":            1,
					"iid":           1,
					"title":         "MR 1",
					"source_branch": "feature1",
					"target_branch": "main",
					"state":         "opened",
					"author": map[string]interface{}{
						"id":       123,
						"username": "user1",
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

	t.Run("list merge requests by branch", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":               1,
					"iid":              1,
					"title":            "MR from feature",
					"source_branch":    "feature",
					"target_branch":    "main",
					"state":            "opened",
					"merge_commit_sha": "abc123",
					"author": map[string]interface{}{
						"id":       123,
						"username": "user1",
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
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("unexpected method: %s", r.Method)
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":            1001,
				"iid":           11,
				"title":         "New Feature",
				"description":   "Description",
				"source_branch": "feature",
				"target_branch": "main",
				"state":         "opened",
				"author": map[string]interface{}{
					"id":       123,
					"username": "testuser",
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
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "PUT" {
				t.Errorf("unexpected method: %s", r.Method)
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":            1001,
				"iid":           10,
				"title":         "Updated Title",
				"description":   "Updated Description",
				"source_branch": "feature",
				"target_branch": "main",
				"state":         "opened",
				"author": map[string]interface{}{
					"id":       123,
					"username": "testuser",
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
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":            1001,
				"iid":           10,
				"title":         "Merged MR",
				"source_branch": "feature",
				"target_branch": "main",
				"state":         "merged",
				"author": map[string]interface{}{
					"id":       123,
					"username": "testuser",
				},
				"merged_at":  time.Now().Format(time.RFC3339),
				"created_at": time.Now().Format(time.RFC3339),
				"updated_at": time.Now().Format(time.RFC3339),
			})
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
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":            1001,
				"iid":           10,
				"title":         "Closed MR",
				"source_branch": "feature",
				"target_branch": "main",
				"state":         "closed",
				"author": map[string]interface{}{
					"id":       123,
					"username": "testuser",
				},
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
