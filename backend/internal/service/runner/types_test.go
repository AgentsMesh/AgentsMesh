package runner

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

// --- Types and Error Tests ---

func TestNewService(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	if service == nil {
		t.Fatal("expected non-nil service")
	}
	if service.db != db {
		t.Fatal("expected service.db to be the provided db")
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{ErrRunnerNotFound, "runner not found"},
		{ErrRunnerOffline, "runner is offline"},
		{ErrInvalidToken, "invalid registration token"},
		{ErrTokenExpired, "registration token expired"},
		{ErrTokenExhausted, "registration token usage exhausted"},
		{ErrRunnerAlreadyExists, "runner already exists"},
	}

	for _, tt := range tests {
		if tt.err.Error() != tt.expected {
			t.Errorf("Error message = %s, want %s", tt.err.Error(), tt.expected)
		}
	}
}

func TestActiveRunnerStruct(t *testing.T) {
	ar := &ActiveRunner{
		Runner:   &runner.Runner{ID: 1, NodeID: "test"},
		PodCount: 5,
	}

	if ar.Runner.ID != 1 {
		t.Errorf("expected Runner.ID 1, got %d", ar.Runner.ID)
	}
	if ar.PodCount != 5 {
		t.Errorf("expected PodCount 5, got %d", ar.PodCount)
	}
}

func TestRunnerUpdateInput(t *testing.T) {
	desc := "desc"
	max := 10
	enabled := true

	input := RunnerUpdateInput{
		Description:       &desc,
		MaxConcurrentPods: &max,
		IsEnabled:         &enabled,
	}

	if *input.Description != desc {
		t.Errorf("expected Description %s, got %s", desc, *input.Description)
	}
	if *input.MaxConcurrentPods != max {
		t.Errorf("expected MaxConcurrentPods %d, got %d", max, *input.MaxConcurrentPods)
	}
	if *input.IsEnabled != enabled {
		t.Errorf("expected IsEnabled %v, got %v", enabled, *input.IsEnabled)
	}
}

func TestHeartbeatPodInfo(t *testing.T) {
	hs := HeartbeatPodInfo{
		PodKey:      "pod-123",
		Status:      "running",
		AgentStatus: "waiting",
	}

	if hs.PodKey != "pod-123" {
		t.Errorf("expected PodKey pod-123, got %s", hs.PodKey)
	}
	if hs.Status != "running" {
		t.Errorf("expected Status running, got %s", hs.Status)
	}
	if hs.AgentStatus != "waiting" {
		t.Errorf("expected AgentStatus waiting, got %s", hs.AgentStatus)
	}
}

func TestSupportedFeatures(t *testing.T) {
	features := SupportedFeatures()

	// Should contain expected features
	if len(features) == 0 {
		t.Fatal("expected at least one supported feature")
	}

	// Check expected features are present
	expectedFeatures := []string{
		FeatureFilesToCreate,
		FeatureWorkDirConfig,
		FeatureInitialPrompt,
	}

	for _, expected := range expectedFeatures {
		found := false
		for _, f := range features {
			if f == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected feature %s not found in SupportedFeatures()", expected)
		}
	}
}

func TestFeatureConstants(t *testing.T) {
	// Verify feature constants have expected values
	if FeatureFilesToCreate != "files_to_create" {
		t.Errorf("FeatureFilesToCreate = %s, want files_to_create", FeatureFilesToCreate)
	}
	if FeatureWorkDirConfig != "work_dir_config" {
		t.Errorf("FeatureWorkDirConfig = %s, want work_dir_config", FeatureWorkDirConfig)
	}
	if FeatureInitialPrompt != "initial_prompt" {
		t.Errorf("FeatureInitialPrompt = %s, want initial_prompt", FeatureInitialPrompt)
	}
}
