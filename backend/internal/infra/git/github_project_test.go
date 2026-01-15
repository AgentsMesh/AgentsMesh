package git

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestGitHubGetProject(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/repos/owner/repo" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":             456,
				"name":           "repo",
				"full_name":      "owner/repo",
				"description":    "Test repo",
				"default_branch": "main",
				"html_url":       "https://github.com/owner/repo",
				"clone_url":      "https://github.com/owner/repo.git",
				"ssh_url":        "git@github.com:owner/repo.git",
				"visibility":     "public",
				"created_at":     time.Now().Format(time.RFC3339),
				"updated_at":     time.Now().Format(time.RFC3339),
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

	t.Run("private repo", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":             456,
				"name":           "private-repo",
				"full_name":      "owner/private-repo",
				"default_branch": "main",
				"visibility":     "private",
				"created_at":     time.Now().Format(time.RFC3339),
				"updated_at":     time.Now().Format(time.RFC3339),
			})
		})
		defer server.Close()

		project, err := provider.GetProject(ctx, "owner/private-repo")
		if err != nil {
			t.Fatalf("GetProject failed: %v", err)
		}
		if project.Visibility != "private" {
			t.Errorf("project.Visibility = %s, want private", project.Visibility)
		}
	})

	t.Run("not found", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		defer server.Close()

		_, err := provider.GetProject(ctx, "owner/nonexistent")
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestGitHubListProjects(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/user/repos" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.URL.Query().Get("page") != "1" {
				t.Errorf("unexpected page: %s", r.URL.Query().Get("page"))
			}

			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":             1,
					"name":           "repo1",
					"full_name":      "owner/repo1",
					"default_branch": "main",
					"visibility":     "public",
					"created_at":     time.Now().Format(time.RFC3339),
					"updated_at":     time.Now().Format(time.RFC3339),
				},
				{
					"id":             2,
					"name":           "repo2",
					"full_name":      "owner/repo2",
					"default_branch": "master",
					"visibility":     "private",
					"created_at":     time.Now().Format(time.RFC3339),
					"updated_at":     time.Now().Format(time.RFC3339),
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
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
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

func TestGitHubSearchProjects(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/search/repositories" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.URL.Query().Get("q") != "test" {
				t.Errorf("unexpected query: %s", r.URL.Query().Get("q"))
			}

			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]interface{}{
					{
						"id":             1,
						"name":           "test-repo",
						"full_name":      "owner/test-repo",
						"default_branch": "main",
						"visibility":     "public",
						"created_at":     time.Now().Format(time.RFC3339),
						"updated_at":     time.Now().Format(time.RFC3339),
					},
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

	t.Run("no results", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"items": []map[string]interface{}{},
			})
		})
		defer server.Close()

		projects, err := provider.SearchProjects(ctx, "nonexistent", 1, 20)
		if err != nil {
			t.Fatalf("SearchProjects failed: %v", err)
		}
		if len(projects) != 0 {
			t.Errorf("len(projects) = %d, want 0", len(projects))
		}
	})
}
