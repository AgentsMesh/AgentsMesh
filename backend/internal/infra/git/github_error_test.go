package git

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGitHubDoRequestErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("http client error", func(t *testing.T) {
		// Use invalid URL to trigger HTTP client error
		provider, _ := NewGitHubProvider("http://invalid-host-that-does-not-exist:99999", "test-token")
		_, err := provider.GetCurrentUser(ctx)
		if err == nil {
			t.Error("expected error for invalid host")
		}
	})
}

func TestGitHubErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("get current user HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
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

		provider, _ := NewGitHubProvider(server.URL, "test-token")
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

		provider, _ := NewGitHubProvider(server.URL, "test-token")
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

		provider, _ := NewGitHubProvider(server.URL, "test-token")
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

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		_, err := provider.ListProjects(ctx, 1, 20)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("search projects invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
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

		provider, _ := NewGitHubProvider(server.URL, "test-token")
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

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		_, err := provider.GetBranch(ctx, "owner/repo", "main")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("create branch HTTP error on ref GET", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		_, err := provider.CreateBranch(ctx, "owner/repo", "feature", "main")
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})

	t.Run("delete branch HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		err := provider.DeleteBranch(ctx, "owner/repo", "protected-branch")
		if err == nil {
			t.Error("expected error for HTTP 403")
		}
	})

	t.Run("get commit invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
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

		provider, _ := NewGitHubProvider(server.URL, "test-token")
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

		provider, _ := NewGitHubProvider(server.URL, "test-token")
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

		provider, _ := NewGitHubProvider(server.URL, "test-token")
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

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		err := provider.DeleteWebhook(ctx, "owner/repo", "12345")
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})
}

func TestGitHubMergeRequestStateMapping(t *testing.T) {
	ctx := context.Background()

	t.Run("list merge requests by branch with merged state", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("state") != "closed" {
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

	t.Run("list merge requests by branch with closed state", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
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

	t.Run("list merge requests by branch with all state", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("state") != "all" {
				t.Errorf("unexpected state: %s, want all", r.URL.Query().Get("state"))
			}
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		})
		defer server.Close()

		_, err := provider.ListMergeRequestsByBranch(ctx, "owner/repo", "feature", "all")
		if err != nil {
			t.Fatalf("ListMergeRequestsByBranch failed: %v", err)
		}
	})

	t.Run("get merge request with merged_at field", func(t *testing.T) {
		mergedAt := time.Now().Add(-1 * time.Hour)
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":         1001,
				"number":     10,
				"title":      "Merged PR",
				"body":       "Description",
				"head":       map[string]interface{}{"ref": "feature"},
				"base":       map[string]interface{}{"ref": "main"},
				"state":      "closed",
				"html_url":   "https://github.com/owner/repo/pull/10",
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
		if mr.State != "merged" {
			t.Errorf("expected state=merged, got %s", mr.State)
		}
	})

	t.Run("list merge requests by branch with merged_at items", func(t *testing.T) {
		mergedAt := time.Now().Add(-1 * time.Hour)
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":         1001,
					"number":     10,
					"title":      "Merged PR",
					"head":       map[string]interface{}{"ref": "feature"},
					"base":       map[string]interface{}{"ref": "main"},
					"state":      "closed",
					"merged_at":  mergedAt.Format(time.RFC3339),
					"user":       map[string]interface{}{"id": 123, "login": "testuser"},
					"created_at": time.Now().Format(time.RFC3339),
					"updated_at": time.Now().Format(time.RFC3339),
				},
			})
		})
		defer server.Close()

		mrs, err := provider.ListMergeRequestsByBranch(ctx, "owner/repo", "feature", "all")
		if err != nil {
			t.Fatalf("ListMergeRequestsByBranch failed: %v", err)
		}
		if len(mrs) != 1 || mrs[0].State != "merged" {
			t.Error("expected merged state for PR with merged_at")
		}
	})
}

func TestGitHubMergeRequestHttpErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("create merge request HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
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

		provider, _ := NewGitHubProvider(server.URL, "test-token")
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

		provider, _ := NewGitHubProvider(server.URL, "test-token")
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

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		_, err := provider.MergeMergeRequest(ctx, "owner/repo", 1)
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})
}

func TestGitHubMergeRequestErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("get merge request invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
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

		provider, _ := NewGitHubProvider(server.URL, "test-token")
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

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		_, err := provider.ListMergeRequestsByBranch(ctx, "owner/repo", "feature", "opened")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("create merge request invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
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

		provider, _ := NewGitHubProvider(server.URL, "test-token")
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

		provider, _ := NewGitHubProvider(server.URL, "test-token")
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

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		_, err := provider.CloseMergeRequest(ctx, "owner/repo", 1)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestGitHubPipelineErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("trigger pipeline invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		_, err := provider.TriggerPipeline(ctx, "owner/repo", &TriggerPipelineRequest{Ref: "main"})
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("get pipeline invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		_, err := provider.GetPipeline(ctx, "owner/repo", 1001)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("list pipelines invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		_, err := provider.ListPipelines(ctx, "owner/repo", "main", "success", 1, 20)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("cancel pipeline HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		_, err := provider.CancelPipeline(ctx, "owner/repo", 1001)
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})

	t.Run("retry pipeline HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		_, err := provider.RetryPipeline(ctx, "owner/repo", 1001)
		if err == nil {
			t.Error("expected error for HTTP 403")
		}
	})
}

func TestGitHubJobErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("get job invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		_, err := provider.GetJob(ctx, "owner/repo", 2001)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("list pipeline jobs invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		_, err := provider.ListPipelineJobs(ctx, "owner/repo", 1001)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("retry job HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		_, err := provider.RetryJob(ctx, "owner/repo", 2001)
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})

	t.Run("cancel job HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		_, err := provider.CancelJob(ctx, "owner/repo", 2001)
		if err == nil {
			t.Error("expected error for HTTP 403")
		}
	})

	t.Run("get job trace HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		_, err := provider.GetJobTrace(ctx, "owner/repo", 2001)
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})

	t.Run("get job artifact HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		_, err := provider.GetJobArtifact(ctx, "owner/repo", 2001, "artifact.zip")
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})

	t.Run("download job artifacts HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		provider, _ := NewGitHubProvider(server.URL, "test-token")
		_, err := provider.DownloadJobArtifacts(ctx, "owner/repo", 2001)
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})
}

func TestGitHubAdditionalStatusMapping(t *testing.T) {
	ctx := context.Background()

	// Test additional status mapping cases not covered in github_job_test.go
	statusTests := []struct {
		status     string
		conclusion string
		expected   string
	}{
		{"completed", "timed_out", PipelineStatusFailed},
		{"unknown", "", PipelineStatusPending},
	}

	for _, tc := range statusTests {
		t.Run(tc.status+"_"+tc.conclusion, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":          1001,
					"run_number":  1,
					"head_branch": "main",
					"head_sha":    "abc123",
					"status":      tc.status,
					"conclusion":  tc.conclusion,
					"event":       "push",
					"html_url":    "https://github.com/owner/repo/actions/runs/1001",
					"created_at":  "2024-01-01T00:00:00Z",
					"updated_at":  "2024-01-01T00:01:00Z",
				})
			}))
			defer server.Close()

			provider, _ := NewGitHubProvider(server.URL, "test-token")
			pipeline, err := provider.GetPipeline(ctx, "owner/repo", 1001)
			if err != nil {
				t.Fatalf("GetPipeline failed: %v", err)
			}
			if pipeline.Status != tc.expected {
				t.Errorf("status = %s, want %s", pipeline.Status, tc.expected)
			}
		})
	}
}
