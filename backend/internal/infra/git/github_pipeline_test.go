package git

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestGitHubTriggerPipeline(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		callCount := 0
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				// dispatch endpoint returns 204 No Content
				w.WriteHeader(http.StatusNoContent)
			} else {
				// ListPipelines returns workflow runs
				json.NewEncoder(w).Encode(map[string]interface{}{
					"workflow_runs": []map[string]interface{}{
						{
							"id":          1001,
							"run_number":  5,
							"head_branch": "main",
							"head_sha":    "abc123",
							"status":      "queued",
							"event":       "workflow_dispatch",
							"html_url":    "https://github.com/owner/repo/actions/runs/1001",
							"created_at":  time.Now().Format(time.RFC3339),
							"updated_at":  time.Now().Format(time.RFC3339),
						},
					},
				})
			}
		})
		defer server.Close()

		pipeline, err := provider.TriggerPipeline(ctx, "owner/repo", &TriggerPipelineRequest{
			Ref: "main",
		})
		if err != nil {
			t.Fatalf("TriggerPipeline failed: %v", err)
		}
		if pipeline.ID != 1001 {
			t.Errorf("pipeline.ID = %d, want 1001", pipeline.ID)
		}
	})

	t.Run("no runs found", func(t *testing.T) {
		callCount := 0
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				w.WriteHeader(http.StatusNoContent)
			} else {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"workflow_runs": []map[string]interface{}{},
				})
			}
		})
		defer server.Close()

		pipeline, err := provider.TriggerPipeline(ctx, "owner/repo", &TriggerPipelineRequest{
			Ref: "main",
		})
		if err != nil {
			t.Fatalf("TriggerPipeline failed: %v", err)
		}
		// When no runs found, returns a pending pipeline without ID
		if pipeline.Status != PipelineStatusPending {
			t.Errorf("pipeline.Status = %s, want pending", pipeline.Status)
		}
	})
}

func TestGitHubGetPipeline(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":          1001,
				"run_number":  5,
				"head_branch": "main",
				"head_sha":    "abc123",
				"status":      "completed",
				"conclusion":  "success",
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
		if pipeline.ID != 1001 {
			t.Errorf("pipeline.ID = %d, want 1001", pipeline.ID)
		}
		if pipeline.Status != PipelineStatusSuccess {
			t.Errorf("pipeline.Status = %s, want success", pipeline.Status)
		}
	})

	t.Run("not found", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		defer server.Close()

		_, err := provider.GetPipeline(ctx, "owner/repo", 9999)
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestGitHubListPipelines(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"workflow_runs": []map[string]interface{}{
					{
						"id":          1001,
						"run_number":  5,
						"head_branch": "main",
						"head_sha":    "abc123",
						"status":      "completed",
						"conclusion":  "success",
						"event":       "push",
						"html_url":    "https://github.com/owner/repo/actions/runs/1001",
						"created_at":  time.Now().Format(time.RFC3339),
						"updated_at":  time.Now().Format(time.RFC3339),
					},
				},
			})
		})
		defer server.Close()

		pipelines, err := provider.ListPipelines(ctx, "owner/repo", "main", "success", 1, 20)
		if err != nil {
			t.Fatalf("ListPipelines failed: %v", err)
		}
		if len(pipelines) != 1 {
			t.Errorf("len(pipelines) = %d, want 1", len(pipelines))
		}
	})
}

func TestGitHubCancelPipeline(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		callCount := 0
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				w.WriteHeader(http.StatusAccepted)
			} else {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":          1001,
					"run_number":  5,
					"head_branch": "main",
					"head_sha":    "abc123",
					"status":      "completed",
					"conclusion":  "cancelled",
					"event":       "push",
					"html_url":    "https://github.com/owner/repo/actions/runs/1001",
					"created_at":  time.Now().Format(time.RFC3339),
					"updated_at":  time.Now().Format(time.RFC3339),
				})
			}
		})
		defer server.Close()

		pipeline, err := provider.CancelPipeline(ctx, "owner/repo", 1001)
		if err != nil {
			t.Fatalf("CancelPipeline failed: %v", err)
		}
		if pipeline.ID != 1001 {
			t.Errorf("pipeline.ID = %d, want 1001", pipeline.ID)
		}
	})
}

func TestGitHubRetryPipeline(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		callCount := 0
		server, provider := setupGitHubMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if callCount == 1 {
				w.WriteHeader(http.StatusCreated)
			} else {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"id":          1001,
					"run_number":  6,
					"head_branch": "main",
					"head_sha":    "abc123",
					"status":      "queued",
					"event":       "push",
					"html_url":    "https://github.com/owner/repo/actions/runs/1001",
					"created_at":  time.Now().Format(time.RFC3339),
					"updated_at":  time.Now().Format(time.RFC3339),
				})
			}
		})
		defer server.Close()

		pipeline, err := provider.RetryPipeline(ctx, "owner/repo", 1001)
		if err != nil {
			t.Fatalf("RetryPipeline failed: %v", err)
		}
		if pipeline == nil {
			t.Error("expected pipeline, got nil")
		}
	})
}
