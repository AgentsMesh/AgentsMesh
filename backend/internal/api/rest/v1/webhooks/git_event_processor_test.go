package webhooks

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
)

// ===========================================
// processMROrPipelineEvent Tests
// ===========================================

func TestProcessMROrPipelineEvent_MergeRequest(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(t, cfg)

	ctx := &WebhookContext{
		Context:        context.Background(),
		RepoID:         1,
		OrganizationID: 1,
		Payload: map[string]interface{}{
			"object_kind": "merge_request",
			"object_attributes": map[string]interface{}{
				"iid":           float64(123),
				"url":           "https://gitlab.com/org/repo/-/merge_requests/123",
				"title":         "Test MR",
				"source_branch": "feature-branch",
				"target_branch": "main",
				"state":         "opened",
				"action":        "open",
			},
		},
	}

	result, err := router.processMROrPipelineEvent(ctx, "merge_request")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["handler"] != "merge_request" {
		t.Errorf("expected handler 'merge_request', got %v", result["handler"])
	}
	if result["mr_iid"] != 123 {
		t.Errorf("expected mr_iid 123, got %v", result["mr_iid"])
	}
}

func TestProcessMROrPipelineEvent_Pipeline(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(t, cfg)

	ctx := &WebhookContext{
		Context:        context.Background(),
		RepoID:         1,
		OrganizationID: 1,
		PipelineID:     456,
		PipelineStatus: "success",
		Payload: map[string]interface{}{
			"object_kind": "pipeline",
			"object_attributes": map[string]interface{}{
				"id":     float64(456),
				"status": "success",
				"ref":    "main",
				"url":    "https://gitlab.com/org/repo/-/pipelines/456",
			},
		},
	}

	result, err := router.processMROrPipelineEvent(ctx, "pipeline")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["handler"] != "pipeline" {
		t.Errorf("expected handler 'pipeline', got %v", result["handler"])
	}
	if result["pipeline_id"] != int64(456) {
		t.Errorf("expected pipeline_id 456, got %v", result["pipeline_id"])
	}
}

func TestProcessMROrPipelineEvent_UnsupportedKind(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(t, cfg)

	ctx := &WebhookContext{
		Context: context.Background(),
		Payload: map[string]interface{}{},
	}

	_, err := router.processMROrPipelineEvent(ctx, "unsupported")

	if err == nil {
		t.Error("expected error for unsupported object kind")
	}
}

// ===========================================
// processMergeRequestEvent Tests
// ===========================================

func TestProcessMergeRequestEvent_Basic(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(t, cfg)

	ctx := &WebhookContext{
		Context:        context.Background(),
		RepoID:         1,
		OrganizationID: 1,
		Payload: map[string]interface{}{
			"object_attributes": map[string]interface{}{
				"iid":           float64(42),
				"url":           "https://gitlab.com/org/repo/-/merge_requests/42",
				"title":         "Fix bug",
				"source_branch": "fix/AM-123-bug",
				"target_branch": "main",
				"state":         "opened",
				"action":        "open",
			},
		},
	}

	result, err := router.processMergeRequestEvent(ctx)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("expected status 'ok', got %v", result["status"])
	}
	if result["handler"] != "merge_request" {
		t.Errorf("expected handler 'merge_request', got %v", result["handler"])
	}
	if result["mr_iid"] != 42 {
		t.Errorf("expected mr_iid 42, got %v", result["mr_iid"])
	}
	if result["source_branch"] != "fix/AM-123-bug" {
		t.Errorf("expected source_branch 'fix/AM-123-bug', got %v", result["source_branch"])
	}
	if result["action"] != "open" {
		t.Errorf("expected action 'open', got %v", result["action"])
	}
}

func TestProcessMergeRequestEvent_InvalidPayload(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(t, cfg)

	ctx := &WebhookContext{
		Context:        context.Background(),
		RepoID:         1,
		OrganizationID: 1,
		Payload:        map[string]interface{}{},
	}

	_, err := router.processMergeRequestEvent(ctx)

	if err == nil {
		t.Error("expected error for invalid payload")
	}
}
