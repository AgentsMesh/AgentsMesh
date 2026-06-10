package webhooks

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/config"
)

func githubPRPayload(action string, pr map[string]interface{}) map[string]interface{} {
	if _, ok := pr["number"]; !ok {
		pr["number"] = float64(42)
	}
	if _, ok := pr["html_url"]; !ok {
		pr["html_url"] = "https://github.example/org/repo/pull/42"
	}
	if _, ok := pr["title"]; !ok {
		pr["title"] = "Test PR Title"
	}
	if _, ok := pr["head"]; !ok {
		pr["head"] = map[string]interface{}{"ref": "feature/AM-100-test"}
	}
	if _, ok := pr["base"]; !ok {
		pr["base"] = map[string]interface{}{"ref": "main"}
	}
	if _, ok := pr["state"]; !ok {
		pr["state"] = "open"
	}
	return map[string]interface{}{
		"action":       action,
		"pull_request": pr,
	}
}

func TestNormalizeGitHubMRPayload_OpenedExtractsViaGitLabPath(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(t, cfg)

	payload := githubPRPayload("opened", map[string]interface{}{})

	normalizeGitHubMRPayload(payload)

	mrData, action, err := router.extractMRData(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mrData.IID != 42 {
		t.Errorf("expected IID 42, got %d", mrData.IID)
	}
	if mrData.WebURL != "https://github.example/org/repo/pull/42" {
		t.Errorf("unexpected WebURL: %s", mrData.WebURL)
	}
	if mrData.Title != "Test PR Title" {
		t.Errorf("expected title 'Test PR Title', got %s", mrData.Title)
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
}

func TestNormalizeGitHubMRPayload_MergedPR(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(t, cfg)

	payload := githubPRPayload("closed", map[string]interface{}{
		"state":            "closed",
		"merged":           true,
		"merge_commit_sha": "abc123def456",
		"merged_at":        "2026-06-10T10:00:00Z",
	})

	normalizeGitHubMRPayload(payload)

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
		t.Error("expected merge commit SHA 'abc123def456'")
	}
	if mrData.MergedAt == nil {
		t.Error("expected merged_at to be parsed")
	}
}

func TestNormalizeGitHubMRPayload_ClosedWithoutMerge(t *testing.T) {
	cfg := &config.Config{}
	router, _ := createTestRouterForGit(t, cfg)

	payload := githubPRPayload("closed", map[string]interface{}{
		"state":            "closed",
		"merged":           false,
		"merge_commit_sha": "deadbeef0000",
	})

	normalizeGitHubMRPayload(payload)

	mrData, action, err := router.extractMRData(payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mrData.State != "closed" {
		t.Errorf("expected state 'closed', got %s", mrData.State)
	}
	if action != "close" {
		t.Errorf("expected action 'close', got %s", action)
	}
	if mrData.MergeCommitSHA != nil {
		t.Error("expected no merge commit SHA for unmerged close")
	}
}

func TestNormalizeGitHubMRPayload_ActionMapping(t *testing.T) {
	tests := []struct {
		githubAction string
		merged       bool
		expected     string
	}{
		{"opened", false, "open"},
		{"reopened", false, "reopen"},
		{"closed", false, "close"},
		{"closed", true, "merge"},
		{"synchronize", false, "update"},
		{"edited", false, "update"},
		{"ready_for_review", false, "ready_for_review"},
	}

	for _, tt := range tests {
		result := githubMRAction(tt.githubAction, tt.merged)
		if result != tt.expected {
			t.Errorf("action=%s merged=%v: expected %s, got %s", tt.githubAction, tt.merged, tt.expected, result)
		}
	}
}

func TestNormalizeGitHubMRPayload_NoPullRequestKeyIsNoop(t *testing.T) {
	payload := map[string]interface{}{
		"action": "opened",
		"issue":  map[string]interface{}{"number": float64(7)},
	}

	normalizeGitHubMRPayload(payload)

	if _, exists := payload["object_attributes"]; exists {
		t.Error("expected no object_attributes for payload without pull_request")
	}
}

func TestNormalizeGitHubMRPayload_ExistingObjectAttributesUntouched(t *testing.T) {
	original := map[string]interface{}{"iid": float64(9)}
	payload := map[string]interface{}{
		"object_attributes": original,
		"pull_request":      map[string]interface{}{"number": float64(42)},
	}

	normalizeGitHubMRPayload(payload)

	objAttrs, ok := payload["object_attributes"].(map[string]interface{})
	if !ok || objAttrs["iid"] != float64(9) {
		t.Error("expected pre-existing object_attributes to be preserved")
	}
}
