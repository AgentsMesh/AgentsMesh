package webhooks

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
)

// ===========================================
// processPipelineEvent Tests
// ===========================================

func TestProcessPipelineEvent_Basic(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(t, cfg)

	ctx := &WebhookContext{
		Context:        context.Background(),
		RepoID:         1,
		OrganizationID: 1,
		PipelineID:     789,
		PipelineStatus: "success",
		Payload: map[string]interface{}{
			"object_attributes": map[string]interface{}{
				"id":     float64(789),
				"status": "success",
				"ref":    "main",
				"url":    "https://gitlab.com/org/repo/-/pipelines/789",
			},
		},
	}

	result, err := router.processPipelineEvent(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", result["status"])
	}
	if result["handler"] != "pipeline" {
		t.Errorf("expected handler 'pipeline', got %v", result["handler"])
	}
	if result["pipeline_id"] != int64(789) {
		t.Errorf("expected pipeline_id 789, got %v", result["pipeline_id"])
	}
	if result["pipeline_status"] != "success" {
		t.Errorf("expected pipeline_status 'success', got %v", result["pipeline_status"])
	}
	if result["ref"] != "main" {
		t.Errorf("expected ref 'main', got %v", result["ref"])
	}
}

func TestProcessPipelineEvent_FailedPipeline(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(t, cfg)

	ctx := &WebhookContext{
		Context:        context.Background(),
		RepoID:         1,
		OrganizationID: 1,
		PipelineID:     100,
		PipelineStatus: "failed",
		Payload: map[string]interface{}{
			"object_attributes": map[string]interface{}{
				"id":     float64(100),
				"status": "failed",
				"ref":    "feature-branch",
			},
		},
	}

	result, err := router.processPipelineEvent(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["pipeline_status"] != "failed" {
		t.Errorf("expected pipeline_status 'failed', got %v", result["pipeline_status"])
	}
}

func TestProcessPipelineEvent_NoObjectAttributes(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(t, cfg)

	ctx := &WebhookContext{
		Context:        context.Background(),
		RepoID:         1,
		OrganizationID: 1,
		PipelineID:     100,
		PipelineStatus: "success",
		Payload:        map[string]interface{}{},
	}

	result, err := router.processPipelineEvent(ctx)

	// Should still work, just with empty ref and URL
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["ref"] != "" {
		t.Errorf("expected empty ref, got %v", result["ref"])
	}
}

// ===========================================
// findMRByPipeline Tests (with mock DB)
// ===========================================

func TestFindMRByPipeline_NotFound(t *testing.T) {
	cfg := &config.Config{}
	router, db := createTestRouterForGit(t, cfg)

	// Create merge_requests table
	db.Exec(`
		CREATE TABLE IF NOT EXISTS merge_requests (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			organization_id INTEGER NOT NULL,
			ticket_id INTEGER NOT NULL,
			pod_id INTEGER,
			mr_iid INTEGER NOT NULL,
			mr_url TEXT,
			source_branch TEXT,
			target_branch TEXT,
			title TEXT,
			state TEXT,
			pipeline_id INTEGER,
			pipeline_status TEXT,
			pipeline_url TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_synced_at DATETIME
		)
	`)

	result := router.findMRByPipeline(context.Background(), 1, 999, "")

	if result != nil {
		t.Error("expected nil result when MR not found")
	}
}
