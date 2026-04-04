package workspace

import (
	"os"
	"strings"
	"testing"
)

// --- Test PreparationContext ---

func TestPreparationContextGetEnvVars(t *testing.T) {
	ctx := &PreparationContext{
		PodID:        "pod-1",
		TicketSlug:   "TICKET-123",
		BranchName:   "feature/test",
		WorkspaceDir: "/workspace/sandboxes/pod-1/workspace",
		MainRepoDir:  "/workspace/repos/main",
		BaseEnvVars:  map[string]string{"API_KEY": "secret"},
	}

	envVars := ctx.GetEnvVars()

	if envVars["WORKSPACE_DIR"] != "/workspace/sandboxes/pod-1/workspace" {
		t.Errorf("WORKSPACE_DIR: got %v, want /workspace/sandboxes/pod-1/workspace", envVars["WORKSPACE_DIR"])
	}

	if envVars["MAIN_REPO_DIR"] != "/workspace/repos/main" {
		t.Errorf("MAIN_REPO_DIR: got %v, want /workspace/repos/main", envVars["MAIN_REPO_DIR"])
	}

	if envVars["TICKET_SLUG"] != "TICKET-123" {
		t.Errorf("TICKET_SLUG: got %v, want TICKET-123", envVars["TICKET_SLUG"])
	}

	if envVars["BRANCH_NAME"] != "feature/test" {
		t.Errorf("BRANCH_NAME: got %v, want feature/test", envVars["BRANCH_NAME"])
	}

	if envVars["API_KEY"] != "secret" {
		t.Errorf("API_KEY: got %v, want secret", envVars["API_KEY"])
	}
}

func TestPreparationContextString(t *testing.T) {
	ctx := &PreparationContext{
		PodID:        "pod-1",
		TicketSlug:   "TICKET-123",
		WorkspaceDir: "/workspace/test",
	}

	str := ctx.String()

	if str == "" {
		t.Error("String() should not be empty")
	}
}

func TestPreparationContextGetEnvVarsEmpty(t *testing.T) {
	ctx := &PreparationContext{
		PodID:        "pod-1",
		WorkspaceDir: "/workspace",
	}

	envVars := ctx.GetEnvVars()

	if envVars["WORKSPACE_DIR"] != "/workspace" {
		t.Errorf("WORKSPACE_DIR: got %v, want /workspace", envVars["WORKSPACE_DIR"])
	}

	if _, ok := envVars["TICKET_SLUG"]; ok {
		t.Error("TICKET_SLUG should not be set when empty")
	}
}

func TestPreparationContextStringFormat(t *testing.T) {
	ctx := &PreparationContext{
		PodID:        "pod-1",
		TicketSlug:   "TICKET-123",
		BranchName:   "feature/test",
		WorkspaceDir: "/workspace",
	}

	str := ctx.String()

	if !strings.Contains(str, "pod-1") {
		t.Error("String() should contain pod ID")
	}
	if !strings.Contains(str, "TICKET-123") {
		t.Error("String() should contain ticket slug")
	}
}

func TestPreparationError(t *testing.T) {
	err := &PreparationError{
		Step:   "script",
		Cause:  os.ErrNotExist,
		Output: "command not found",
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() should not be empty")
	}

	if err.Unwrap() != os.ErrNotExist {
		t.Error("Unwrap() should return the cause")
	}
}

func TestPreparationErrorNoOutput(t *testing.T) {
	err := &PreparationError{
		Step:  "script",
		Cause: os.ErrNotExist,
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() should not be empty")
	}
}

func BenchmarkPreparationContextGetEnvVars(b *testing.B) {
	ctx := &PreparationContext{
		PodID:        "pod-1",
		TicketSlug:   "TICKET-123",
		BranchName:   "feature/test",
		WorkspaceDir: "/workspace/sandboxes/pod-1/workspace",
		MainRepoDir:  "/workspace/repos/main",
		BaseEnvVars:  map[string]string{"API_KEY": "secret"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ctx.GetEnvVars()
	}
}
