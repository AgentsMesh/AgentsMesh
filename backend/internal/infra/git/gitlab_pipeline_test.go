package git

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestGitLabPipelineOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("trigger pipeline", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("unexpected method: %s", r.Method)
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":         1001,
				"iid":        5,
				"ref":        "main",
				"sha":        "abc123",
				"status":     "pending",
				"source":     "api",
				"web_url":    "https://gitlab.com/owner/repo/-/pipelines/1001",
				"created_at": time.Now().Format(time.RFC3339),
				"updated_at": time.Now().Format(time.RFC3339),
			})
		})
		defer server.Close()

		pipeline, err := provider.TriggerPipeline(ctx, "owner/repo", &TriggerPipelineRequest{
			Ref: "main",
			Variables: map[string]string{
				"VAR1": "value1",
			},
		})
		if err != nil {
			t.Fatalf("TriggerPipeline failed: %v", err)
		}
		if pipeline.ID != 1001 {
			t.Errorf("pipeline.ID = %d, want 1001", pipeline.ID)
		}
	})

	t.Run("get pipeline", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":         1001,
				"iid":        5,
				"ref":        "main",
				"sha":        "abc123",
				"status":     "success",
				"source":     "push",
				"web_url":    "https://gitlab.com/owner/repo/-/pipelines/1001",
				"created_at": time.Now().Format(time.RFC3339),
				"updated_at": time.Now().Format(time.RFC3339),
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
		if pipeline.Status != "success" {
			t.Errorf("pipeline.Status = %s, want success", pipeline.Status)
		}
	})

	t.Run("get pipeline not found", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		defer server.Close()

		_, err := provider.GetPipeline(ctx, "owner/repo", 9999)
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("list pipelines", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode([]map[string]interface{}{
				{
					"id":         1001,
					"iid":        5,
					"ref":        "main",
					"sha":        "abc123",
					"status":     "success",
					"source":     "push",
					"web_url":    "https://gitlab.com/owner/repo/-/pipelines/1001",
					"created_at": time.Now().Format(time.RFC3339),
					"updated_at": time.Now().Format(time.RFC3339),
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

	t.Run("cancel pipeline", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":         1001,
				"iid":        5,
				"ref":        "main",
				"sha":        "abc123",
				"status":     "canceled",
				"source":     "push",
				"web_url":    "https://gitlab.com/owner/repo/-/pipelines/1001",
				"created_at": time.Now().Format(time.RFC3339),
				"updated_at": time.Now().Format(time.RFC3339),
			})
		})
		defer server.Close()

		pipeline, err := provider.CancelPipeline(ctx, "owner/repo", 1001)
		if err != nil {
			t.Fatalf("CancelPipeline failed: %v", err)
		}
		if pipeline.Status != "canceled" {
			t.Errorf("pipeline.Status = %s, want canceled", pipeline.Status)
		}
	})

	t.Run("retry pipeline", func(t *testing.T) {
		server, provider := setupGitLabMockServer(t, func(w http.ResponseWriter, r *http.Request) {
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":         1002,
				"iid":        6,
				"ref":        "main",
				"sha":        "abc123",
				"status":     "pending",
				"source":     "push",
				"web_url":    "https://gitlab.com/owner/repo/-/pipelines/1002",
				"created_at": time.Now().Format(time.RFC3339),
				"updated_at": time.Now().Format(time.RFC3339),
			})
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
