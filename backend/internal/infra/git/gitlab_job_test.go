package git

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestGitLabJobOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("get job", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":            2001,
				"name":          "build",
				"stage":         "build",
				"status":        "success",
				"ref":           "main",
				"web_url":       "https://gitlab.com/owner/repo/-/jobs/2001",
				"allow_failure": false,
				"duration":      120.5,
				"pipeline": map[string]interface{}{
					"id": 1001,
				},
				"created_at":  time.Now().Format(time.RFC3339),
				"started_at":  time.Now().Format(time.RFC3339),
				"finished_at": time.Now().Format(time.RFC3339),
			})
		})
		defer server.Close()

		job, err := provider.GetJob(ctx, "owner/repo", 2001)
		if err != nil {
			t.Fatalf("GetJob failed: %v", err)
		}
		if job.ID != 2001 {
			t.Errorf("job.ID = %d, want 2001", job.ID)
		}
		if job.Status != "success" {
			t.Errorf("job.Status = %s, want success", job.Status)
		}
	})

	t.Run("get job not found", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		defer server.Close()

		_, err := provider.GetJob(ctx, "owner/repo", 9999)
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("list pipeline jobs", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":            2001,
					"name":          "build",
					"stage":         "build",
					"status":        "success",
					"ref":           "main",
					"web_url":       "https://gitlab.com/owner/repo/-/jobs/2001",
					"allow_failure": false,
					"duration":      120.5,
					"pipeline": map[string]interface{}{
						"id": 1001,
					},
					"created_at": time.Now().Format(time.RFC3339),
				},
				{
					"id":            2002,
					"name":          "test",
					"stage":         "test",
					"status":        "success",
					"ref":           "main",
					"web_url":       "https://gitlab.com/owner/repo/-/jobs/2002",
					"allow_failure": false,
					"duration":      60.0,
					"pipeline": map[string]interface{}{
						"id": 1001,
					},
					"created_at": time.Now().Format(time.RFC3339),
				},
			})
		})
		defer server.Close()

		jobs, err := provider.ListPipelineJobs(ctx, "owner/repo", 1001)
		if err != nil {
			t.Fatalf("ListPipelineJobs failed: %v", err)
		}
		if len(jobs) != 2 {
			t.Errorf("len(jobs) = %d, want 2", len(jobs))
		}
	})

	t.Run("retry job", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     2001,
				"name":   "build",
				"stage":  "build",
				"status": "pending",
				"ref":    "main",
				"pipeline": map[string]interface{}{
					"id": 1001,
				},
				"created_at": time.Now().Format(time.RFC3339),
			})
		})
		defer server.Close()

		job, err := provider.RetryJob(ctx, "owner/repo", 2001)
		if err != nil {
			t.Fatalf("RetryJob failed: %v", err)
		}
		if job == nil {
			t.Error("expected job, got nil")
		}
	})

	t.Run("cancel job", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     2001,
				"name":   "build",
				"stage":  "build",
				"status": "canceled",
				"ref":    "main",
				"pipeline": map[string]interface{}{
					"id": 1001,
				},
				"created_at": time.Now().Format(time.RFC3339),
			})
		})
		defer server.Close()

		job, err := provider.CancelJob(ctx, "owner/repo", 2001)
		if err != nil {
			t.Fatalf("CancelJob failed: %v", err)
		}
		if job.Status != "canceled" {
			t.Errorf("job.Status = %s, want canceled", job.Status)
		}
	})

	t.Run("get job trace", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Job log output here\nLine 2\nLine 3"))
		})
		defer server.Close()

		trace, err := provider.GetJobTrace(ctx, "owner/repo", 2001)
		if err != nil {
			t.Fatalf("GetJobTrace failed: %v", err)
		}
		if trace == "" {
			t.Error("expected non-empty trace")
		}
	})

	t.Run("get job artifact", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("artifact content"))
		})
		defer server.Close()

		artifact, err := provider.GetJobArtifact(ctx, "owner/repo", 2001, "build/output.zip")
		if err != nil {
			t.Fatalf("GetJobArtifact failed: %v", err)
		}
		if string(artifact) != "artifact content" {
			t.Errorf("artifact = %s, want 'artifact content'", string(artifact))
		}
	})

	t.Run("download job artifacts", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("zip archive content"))
		})
		defer server.Close()

		artifacts, err := provider.DownloadJobArtifacts(ctx, "owner/repo", 2001)
		if err != nil {
			t.Fatalf("DownloadJobArtifacts failed: %v", err)
		}
		if len(artifacts) == 0 {
			t.Error("expected non-empty artifacts")
		}
	})
}
