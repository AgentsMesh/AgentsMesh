package webhooks

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

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
