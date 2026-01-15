package git

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestGitLabGetProject(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":                  456,
				"name":                "repo",
				"path_with_namespace": "owner/repo",
				"description":         "Test repo",
				"default_branch":      "main",
				"web_url":             "https://gitlab.com/owner/repo",
				"http_url_to_repo":    "https://gitlab.com/owner/repo.git",
				"ssh_url_to_repo":     "git@gitlab.com:owner/repo.git",
				"visibility":          "public",
				"created_at":          time.Now().Format(time.RFC3339),
				"last_activity_at":    time.Now().Format(time.RFC3339),
			})
		})
		defer server.Close()

		project, err := provider.GetProject(ctx, "owner/repo")
		if err != nil {
			t.Fatalf("GetProject failed: %v", err)
		}
		if project.Name != "repo" {
			t.Errorf("project.Name = %s, want repo", project.Name)
		}
		if project.Visibility != "public" {
			t.Errorf("project.Visibility = %s, want public", project.Visibility)
		}
	})

	t.Run("not found", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		defer server.Close()

		_, err := provider.GetProject(ctx, "owner/nonexistent")
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestGitLabListProjects(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":                  1,
					"name":                "repo1",
					"path_with_namespace": "owner/repo1",
					"default_branch":      "main",
					"visibility":          "public",
					"created_at":          time.Now().Format(time.RFC3339),
					"last_activity_at":    time.Now().Format(time.RFC3339),
				},
				{
					"id":                  2,
					"name":                "repo2",
					"path_with_namespace": "owner/repo2",
					"default_branch":      "master",
					"visibility":          "private",
					"created_at":          time.Now().Format(time.RFC3339),
					"last_activity_at":    time.Now().Format(time.RFC3339),
				},
			})
		})
		defer server.Close()

		projects, err := provider.ListProjects(ctx, 1, 20)
		if err != nil {
			t.Fatalf("ListProjects failed: %v", err)
		}
		if len(projects) != 2 {
			t.Errorf("len(projects) = %d, want 2", len(projects))
		}
	})

	t.Run("empty list", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		})
		defer server.Close()

		projects, err := provider.ListProjects(ctx, 1, 20)
		if err != nil {
			t.Fatalf("ListProjects failed: %v", err)
		}
		if len(projects) != 0 {
			t.Errorf("len(projects) = %d, want 0", len(projects))
		}
	})
}

func TestGitLabSearchProjects(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Query().Get("search") != "test" {
				t.Errorf("unexpected search: %s", r.URL.Query().Get("search"))
			}

			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":                  1,
					"name":                "test-repo",
					"path_with_namespace": "owner/test-repo",
					"default_branch":      "main",
					"visibility":          "public",
					"created_at":          time.Now().Format(time.RFC3339),
					"last_activity_at":    time.Now().Format(time.RFC3339),
				},
			})
		})
		defer server.Close()

		projects, err := provider.SearchProjects(ctx, "test", 1, 20)
		if err != nil {
			t.Fatalf("SearchProjects failed: %v", err)
		}
		if len(projects) != 1 {
			t.Errorf("len(projects) = %d, want 1", len(projects))
		}
	})
}
