package git

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestGiteeGetProject(t *testing.T) {
	ctx := context.Background()

	t.Run("success public repo", func(t *testing.T) {
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":             456,
				"name":           "repo",
				"full_name":      "owner/repo",
				"description":    "Test repo",
				"default_branch": "main",
				"html_url":       "https://gitee.com/owner/repo",
				"clone_url":      "https://gitee.com/owner/repo.git",
				"ssh_url":        "git@gitee.com:owner/repo.git",
				"public":         true,
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

	t.Run("success private repo", func(t *testing.T) {
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":             456,
				"name":           "private-repo",
				"full_name":      "owner/private-repo",
				"default_branch": "main",
				"public":         false,
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
}

func TestGiteeListProjects(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":             1,
					"name":           "repo1",
					"full_name":      "owner/repo1",
					"default_branch": "main",
					"public":         true,
					"created_at":     time.Now().Format(time.RFC3339),
					"updated_at":     time.Now().Format(time.RFC3339),
				},
				{
					"id":             2,
					"name":           "repo2",
					"full_name":      "owner/repo2",
					"default_branch": "master",
					"public":         false,
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
}

func TestGiteeSearchProjects(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGiteeMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":             1,
					"name":           "test-repo",
					"full_name":      "owner/test-repo",
					"default_branch": "main",
					"public":         true,
					"created_at":     time.Now().Format(time.RFC3339),
					"updated_at":     time.Now().Format(time.RFC3339),
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
