package git

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestGitHubGetJob(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		startedAt := time.Now()
		completedAt := startedAt.Add(2 * time.Minute)
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":           2001,
				"name":         "build",
				"status":       "completed",
				"conclusion":   "success",
				"run_id":       1001,
				"html_url":     "https://github.com/owner/repo/actions/runs/1001/jobs/2001",
				"started_at":   startedAt.Format(time.RFC3339),
				"completed_at": completedAt.Format(time.RFC3339),
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
		if job.Status != JobStatusSuccess {
			t.Errorf("job.Status = %s, want success", job.Status)
		}
	})

	t.Run("not found", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		defer server.Close()

		_, err := provider.GetJob(ctx, "owner/repo", 9999)
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestGitHubListPipelineJobs(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jobs": []map[string]interface{}{
					{
						"id":           2001,
						"name":         "build",
						"status":       "completed",
						"conclusion":   "success",
						"run_id":       1001,
						"html_url":     "https://github.com/owner/repo/actions/runs/1001/jobs/2001",
						"started_at":   time.Now().Format(time.RFC3339),
						"completed_at": time.Now().Format(time.RFC3339),
					},
					{
						"id":           2002,
						"name":         "test",
						"status":       "completed",
						"conclusion":   "success",
						"run_id":       1001,
						"html_url":     "https://github.com/owner/repo/actions/runs/1001/jobs/2002",
						"started_at":   time.Now().Format(time.RFC3339),
						"completed_at": time.Now().Format(time.RFC3339),
					},
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
}

func TestGitHubRetryJob(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		callCount := 0
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				w.WriteHeader(http.StatusCreated)
			} else {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":         2001,
					"name":       "build",
					"status":     "queued",
					"run_id":     1001,
					"html_url":   "https://github.com/owner/repo/actions/runs/1001/jobs/2001",
					"started_at": time.Now().Format(time.RFC3339),
				})
			}
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
}

func TestGitHubCancelJob(t *testing.T) {
	ctx := context.Background()

	t.Run("returns job", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":       2001,
				"name":     "build",
				"status":   "in_progress",
				"run_id":   1001,
				"html_url": "https://github.com/owner/repo/actions/runs/1001/jobs/2001",
			})
		})
		defer server.Close()

		// CancelJob for GitHub just returns the job (doesn't support canceling individual jobs)
		job, err := provider.CancelJob(ctx, "owner/repo", 2001)
		if err != nil {
			t.Fatalf("CancelJob failed: %v", err)
		}
		if job == nil {
			t.Error("expected job, got nil")
		}
	})
}

func TestGitHubGetJobTrace(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
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
}

func TestGitHubGetJobArtifact(t *testing.T) {
	ctx := context.Background()

	t.Run("not supported", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			// Handler shouldn't be called
		})
		defer server.Close()

		// GitHub provider returns ErrNotFound for GetJobArtifact
		_, err := provider.GetJobArtifact(ctx, "owner/repo", 2001, "build/output.zip")
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestGitHubDownloadJobArtifacts(t *testing.T) {
	ctx := context.Background()

	t.Run("not supported", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			// Handler shouldn't be called
		})
		defer server.Close()

		// GitHub provider returns ErrNotFound for DownloadJobArtifacts
		_, err := provider.DownloadJobArtifacts(ctx, "owner/repo", 2001)
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestGitHubStatusMapping(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name           string
		status         string
		conclusion     string
		expectedStatus string
	}{
		{"queued", "queued", "", PipelineStatusPending},
		{"in_progress", "in_progress", "", PipelineStatusRunning},
		{"waiting", "waiting", "", PipelineStatusManual},
		{"completed_success", "completed", "success", PipelineStatusSuccess},
		{"completed_failure", "completed", "failure", PipelineStatusFailed},
		{"completed_cancelled", "completed", "cancelled", PipelineStatusCanceled},
		{"completed_skipped", "completed", "skipped", PipelineStatusSkipped},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":          1001,
					"run_number":  5,
					"head_branch": "main",
					"head_sha":    "abc123",
					"status":      tt.status,
					"conclusion":  tt.conclusion,
					"event":       "push",
					"html_url":    "https://github.com/owner/repo/actions/runs/1001",
					"created_at":  time.Now().Format(time.RFC3339),
					"updated_at":  time.Now().Format(time.RFC3339),
				})
			})
			defer server.Close()

			pipeline, err := provider.GetPipeline(ctx, "owner/repo", 1001)
			if err != nil {
				t.Fatalf("GetPipeline failed: %v", err)
			}
			if pipeline.Status != tt.expectedStatus {
				t.Errorf("pipeline.Status = %s, want %s", pipeline.Status, tt.expectedStatus)
			}
		})
	}
}
