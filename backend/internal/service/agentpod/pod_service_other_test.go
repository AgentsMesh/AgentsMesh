package agentpod

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
)

func TestSubscribe(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestPodService(db)
	ctx := context.Background()

	unsubscribe, err := svc.Subscribe(ctx, "test-pod", func(s *agentpod.Pod) {})
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}
	if unsubscribe == nil {
		t.Error("Unsubscribe function is nil")
	}

	// Should not panic when called
	unsubscribe()
}

func TestErrors(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{ErrPodNotFound, "pod not found"},
		{ErrNoAvailableRunner, "no available runner"},
		{ErrPodTerminated, "pod already terminated"},
		{ErrRunnerNotFound, "runner not found"},
		{ErrRunnerOffline, "runner is offline"},
	}

	for _, tt := range tests {
		if tt.err.Error() != tt.expected {
			t.Errorf("Error message = %s, want %s", tt.err.Error(), tt.expected)
		}
	}
}

func TestCreatePodRequest(t *testing.T) {
	req := &CreatePodRequest{
		OrganizationID:    1,
		RunnerID:          3,
		AgentSlug:         "agent-4",
		RepositoryID:      intPtr(6),
		TicketID:          intPtr(7),
		CreatedByID:       8,
		Prompt:            "Test prompt",
		BranchName:        strPtr("feature/test"),
		Model:             "opus",
		PermissionMode:    "plan",
		SkipPermissions:   true,
		EnvVars:           map[string]string{"KEY": "VALUE"},
	}

	if req.OrganizationID != 1 {
		t.Error("OrganizationID not set")
	}
	if req.EnvVars["KEY"] != "VALUE" {
		t.Error("EnvVars not set correctly")
	}
}
