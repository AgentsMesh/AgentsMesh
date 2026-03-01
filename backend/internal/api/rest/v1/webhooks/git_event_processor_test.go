package webhooks

import (
	"context"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/config"
	"github.com/AgentsMesh/AgentsMesh/backend/internal/infra/eventbus"
)

// ===========================================
// processMROrPipelineEvent Tests
// ===========================================

func TestProcessMROrPipelineEvent_MergeRequest(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

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
	router, _ := createTestRouterForGit(cfg)

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
	router, _ := createTestRouterForGit(cfg)

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
// extractMRData Tests
// ===========================================

func TestExtractMRData_CompletePayload(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	payload := map[string]interface{}{
		"object_attributes": map[string]interface{}{
			"iid":           float64(123),
			"url":           "https://gitlab.com/org/repo/-/merge_requests/123",
			"title":         "Test MR Title",
			"source_branch": "feature/AM-100-test",
			"target_branch": "main",
			"state":         "opened",
			"action":        "open",
			"head_pipeline": map[string]interface{}{
				"id":      float64(456),
				"status":  "running",
				"web_url": "https://gitlab.com/org/repo/-/pipelines/456",
			},
		},
	}

	mrData, action, err := router.extractMRData(payload)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mrData.IID != 123 {
		t.Errorf("expected IID 123, got %d", mrData.IID)
	}
	if mrData.WebURL != "https://gitlab.com/org/repo/-/merge_requests/123" {
		t.Errorf("unexpected WebURL: %s", mrData.WebURL)
	}
	if mrData.Title != "Test MR Title" {
		t.Errorf("expected title 'Test MR Title', got %s", mrData.Title)
	}
	if mrData.SourceBranch != "feature/AM-100-test" {
		t.Errorf("expected source_branch 'feature/AM-100-test', got %s", mrData.SourceBranch)
	}
	if mrData.TargetBranch != "main" {
		t.Errorf("expected target_branch 'main', got %s", mrData.TargetBranch)
	}
	if mrData.State != "opened" {
		t.Errorf("expected state 'opened', got %s", mrData.State)
	}
	if action != "open" {
		t.Errorf("expected action 'open', got %s", action)
	}
	if mrData.PipelineStatus == nil || *mrData.PipelineStatus != "running" {
		t.Error("expected pipeline status 'running'")
	}
	if mrData.PipelineID == nil || *mrData.PipelineID != 456 {
		t.Error("expected pipeline ID 456")
	}
}

func TestExtractMRData_MinimalPayload(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	payload := map[string]interface{}{
		"object_attributes": map[string]interface{}{
			"iid":           float64(1),
			"source_branch": "branch",
			"target_branch": "main",
			"state":         "opened",
		},
	}

	mrData, action, err := router.extractMRData(payload)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mrData.IID != 1 {
		t.Errorf("expected IID 1, got %d", mrData.IID)
	}
	if action != "" {
		t.Errorf("expected empty action, got %s", action)
	}
	if mrData.PipelineStatus != nil {
		t.Error("expected nil pipeline status")
	}
}

func TestExtractMRData_MissingObjectAttributes(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	payload := map[string]interface{}{}

	_, _, err := router.extractMRData(payload)

	if err == nil {
		t.Error("expected error for missing object_attributes")
	}
}

func TestExtractMRData_MergedState(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	payload := map[string]interface{}{
		"object_attributes": map[string]interface{}{
			"iid":              float64(123),
			"source_branch":    "feature",
			"target_branch":    "main",
			"state":            "merged",
			"action":           "merge",
			"merge_commit_sha": "abc123def456",
			"merged_at":        "2026-02-06T10:00:00Z",
		},
	}

	mrData, action, err := router.extractMRData(payload)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mrData.State != "merged" {
		t.Errorf("expected state 'merged', got %s", mrData.State)
	}
	if action != "merge" {
		t.Errorf("expected action 'merge', got %s", action)
	}
	if mrData.MergeCommitSHA == nil || *mrData.MergeCommitSHA != "abc123def456" {
		t.Error("expected merge commit SHA")
	}
	if mrData.MergedAt == nil {
		t.Error("expected merged_at to be parsed")
	}
}

// ===========================================
// determineMREventType Tests
// ===========================================

func TestDetermineMREventType_Open(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	tests := []struct {
		state    string
		action   string
		expected eventbus.EventType
	}{
		{"opened", "open", eventbus.EventMRCreated},
		{"opened", "opened", eventbus.EventMRCreated},
		{"", "reopen", eventbus.EventMRCreated},
	}

	for _, tt := range tests {
		result := router.determineMREventType(tt.state, tt.action)
		if result != tt.expected {
			t.Errorf("state=%s, action=%s: expected %s, got %s", tt.state, tt.action, tt.expected, result)
		}
	}
}

func TestDetermineMREventType_Merged(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	tests := []struct {
		state    string
		action   string
		expected eventbus.EventType
	}{
		{"merged", "", eventbus.EventMRMerged},
		{"", "merge", eventbus.EventMRMerged},
		{"merged", "merge", eventbus.EventMRMerged},
	}

	for _, tt := range tests {
		result := router.determineMREventType(tt.state, tt.action)
		if result != tt.expected {
			t.Errorf("state=%s, action=%s: expected %s, got %s", tt.state, tt.action, tt.expected, result)
		}
	}
}

func TestDetermineMREventType_Closed(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	tests := []struct {
		state    string
		action   string
		expected eventbus.EventType
	}{
		{"closed", "", eventbus.EventMRClosed},
		{"", "close", eventbus.EventMRClosed},
		{"closed", "close", eventbus.EventMRClosed},
	}

	for _, tt := range tests {
		result := router.determineMREventType(tt.state, tt.action)
		if result != tt.expected {
			t.Errorf("state=%s, action=%s: expected %s, got %s", tt.state, tt.action, tt.expected, result)
		}
	}
}

func TestDetermineMREventType_Updated(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

	tests := []struct {
		state  string
		action string
	}{
		{"opened", "update"},
		{"opened", ""},
		{"", "approved"},
		{"", "unapproved"},
	}

	for _, tt := range tests {
		result := router.determineMREventType(tt.state, tt.action)
		if result != eventbus.EventMRUpdated {
			t.Errorf("state=%s, action=%s: expected EventMRUpdated, got %s", tt.state, tt.action, result)
		}
	}
}

// ===========================================
// processMergeRequestEvent Tests
// ===========================================

func TestProcessMergeRequestEvent_Basic(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

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
	router, _ := createTestRouterForGit(cfg)

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

// ===========================================
// processPipelineEvent Tests
// ===========================================

func TestProcessPipelineEvent_Basic(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(cfg)

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
	router, _ := createTestRouterForGit(cfg)

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
	router, _ := createTestRouterForGit(cfg)

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
	router, db := createTestRouterForGit(cfg)

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
