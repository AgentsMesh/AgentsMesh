package git

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitLabDoRequestErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("http client error", func(t *testing.T) {
		provider, _ := NewGitLabProvider("http://invalid-host-that-does-not-exist:99999", "test-token")
		_, err := provider.GetCurrentUser(ctx)
		if err == nil {
			t.Error("expected error for invalid host")
		}
	})
}

func TestGitLabErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("get current user HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		provider, _ := NewGitLabProvider(server.URL, "test-token")
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
		_, err := provider.CreateBranch(ctx, "owner/repo", "feature", "abc123")
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("delete branch HTTP not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		provider, _ := NewGitLabProvider(server.URL, "test-token")
		err := provider.DeleteBranch(ctx, "owner/repo", "nonexistent-branch")
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})

	t.Run("get commit invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGitLabProvider(server.URL, "test-token")
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
		_, err := provider.ListCommits(ctx, "owner/repo", "main", 1, 20)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("get file content HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		provider, _ := NewGitLabProvider(server.URL, "test-token")
		_, err := provider.GetFileContent(ctx, "owner/repo", "README.md", "main")
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})

	t.Run("register webhook invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGitLabProvider(server.URL, "test-token")
		_, err := provider.RegisterWebhook(ctx, "owner/repo", &WebhookConfig{
			URL:    "https://example.com/webhook",
			Secret: "secret",
			Events: []string{"push_events"},
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
		err := provider.DeleteWebhook(ctx, "owner/repo", "12345")
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})
}

func TestGitLabMergeRequestHttpErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("create merge request HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer server.Close()

		provider, _ := NewGitLabProvider(server.URL, "test-token")
		_, err := provider.CreateMergeRequest(ctx, &CreateMRRequest{
			ProjectID:    "owner/repo",
			Title:        "Test MR",
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
		_, err := provider.CloseMergeRequest(ctx, "owner/repo", 1)
		if err == nil {
			t.Error("expected error for HTTP 403")
		}
	})
}

func TestGitLabMergeRequestErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("get merge request invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGitLabProvider(server.URL, "test-token")
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
		_, err := provider.CreateMergeRequest(ctx, &CreateMRRequest{
			ProjectID:    "owner/repo",
			Title:        "Test MR",
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
		_, err := provider.CloseMergeRequest(ctx, "owner/repo", 1)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestGitLabPipelineErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("trigger pipeline invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGitLabProvider(server.URL, "test-token")
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
		_, err := provider.GetPipeline(ctx, "owner/repo", 1001)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("list pipelines invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not an array"))
		}))
		defer server.Close()

		provider, _ := NewGitLabProvider(server.URL, "test-token")
		_, err := provider.ListPipelines(ctx, "owner/repo", "main", "success", 1, 20)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("cancel pipeline invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGitLabProvider(server.URL, "test-token")
		_, err := provider.CancelPipeline(ctx, "owner/repo", 1001)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("retry pipeline invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGitLabProvider(server.URL, "test-token")
		_, err := provider.RetryPipeline(ctx, "owner/repo", 1001)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})
}

func TestGitLabJobErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("get job invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGitLabProvider(server.URL, "test-token")
		_, err := provider.GetJob(ctx, "owner/repo", 2001)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("list pipeline jobs invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("not an array"))
		}))
		defer server.Close()

		provider, _ := NewGitLabProvider(server.URL, "test-token")
		_, err := provider.ListPipelineJobs(ctx, "owner/repo", 1001)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("retry job invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGitLabProvider(server.URL, "test-token")
		_, err := provider.RetryJob(ctx, "owner/repo", 2001)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("cancel job invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("{invalid"))
		}))
		defer server.Close()

		provider, _ := NewGitLabProvider(server.URL, "test-token")
		_, err := provider.CancelJob(ctx, "owner/repo", 2001)
		if err == nil {
			t.Error("expected error for invalid JSON")
		}
	})

	t.Run("get job trace HTTP error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		provider, _ := NewGitLabProvider(server.URL, "test-token")
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
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

		provider, _ := NewGitLabProvider(server.URL, "test-token")
		_, err := provider.DownloadJobArtifacts(ctx, "owner/repo", 2001)
		if err == nil {
			t.Error("expected error for HTTP 404")
		}
	})
}
