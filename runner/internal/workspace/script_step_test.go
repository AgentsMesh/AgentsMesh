package workspace

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// --- Test ScriptPreparationStep ---

func TestScriptPreparationStepName(t *testing.T) {
	step := NewScriptPreparationStep("echo hello", time.Minute)

	if step.Name() != "script" {
		t.Errorf("Name: got %v, want script", step.Name())
	}
}

func TestScriptPreparationStepExecuteEmpty(t *testing.T) {
	step := NewScriptPreparationStep("", time.Minute)
	ctx := &PreparationContext{
		PodID:      "pod-1",
		WorkspaceDir: t.TempDir(),
	}

	err := step.Execute(context.Background(), ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScriptPreparationStepExecuteWithEnvVars(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test that requires Unix shell variable expansion")
	}
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.txt")

	script := `echo "$TICKET_SLUG" > "` + outputFile + `"`
	step := NewScriptPreparationStep(script, time.Minute)

	ctx := &PreparationContext{
		PodID:            "pod-1",
		TicketSlug: "TICKET-123",
		WorkspaceDir:       tmpDir,
	}

	err := step.Execute(context.Background(), ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	if string(data) != "TICKET-123\n" {
		t.Errorf("output: got %v, want TICKET-123", string(data))
	}
}

func TestScriptPreparationStepTimeout(t *testing.T) {
	var script string
	if runtime.GOOS == "windows" {
		script = "ping -n 11 127.0.0.1 >nul" // ~10 second delay on Windows
	} else {
		script = "sleep 10"
	}
	step := NewScriptPreparationStep(script, 100*time.Millisecond)
	ctx := &PreparationContext{
		PodID:      "pod-1",
		WorkspaceDir: t.TempDir(),
	}

	err := step.Execute(context.Background(), ctx)
	if err == nil {
		t.Error("expected error for timeout")
	}
}

func TestScriptPreparationStepDefaultTimeout(t *testing.T) {
	step := NewScriptPreparationStep("echo hello", 0)

	if step.timeout != 5*time.Minute {
		t.Errorf("default timeout: got %v, want %v", step.timeout, 5*time.Minute)
	}
}

// --- Test addToolPaths ---

func TestAddToolPathsWithPATH(t *testing.T) {
	step := NewScriptPreparationStep("echo test", time.Minute)

	var env []string
	if runtime.GOOS == "windows" {
		env = []string{"HOME=C:\\Users\\test", "PATH=C:\\Windows\\System32;C:\\Windows", "USER=test"}
	} else {
		env = []string{"HOME=/home/test", "PATH=/usr/bin:/bin", "USER=test"}
	}
	result := step.addToolPaths(env)

	pathFound := false
	for _, e := range result {
		if strings.HasPrefix(e, "PATH=") {
			pathFound = true
			path := strings.TrimPrefix(e, "PATH=")
			if runtime.GOOS == "windows" {
				// On Windows, should still contain original paths
				if !strings.Contains(path, "System32") {
					t.Errorf("PATH should contain System32, got: %s", path)
				}
			} else {
				if !strings.Contains(path, "/usr/bin") {
					t.Errorf("PATH should contain /usr/bin, got: %s", path)
				}
				if !strings.Contains(path, "/usr/local/bin") {
					t.Errorf("PATH should contain /usr/local/bin, got: %s", path)
				}
			}
		}
	}

	if !pathFound {
		t.Error("PATH should be present in result")
	}
}

func TestAddToolPathsWithoutPATH(t *testing.T) {
	step := NewScriptPreparationStep("echo test", time.Minute)

	env := []string{"HOME=/home/test", "USER=test"}
	result := step.addToolPaths(env)

	pathFound := false
	for _, e := range result {
		if strings.HasPrefix(e, "PATH=") {
			pathFound = true
			path := strings.TrimPrefix(e, "PATH=")
			if runtime.GOOS == "windows" {
				// On Windows, the fallback path should contain user binary dirs
				if !strings.Contains(strings.ToLower(path), ".local") {
					t.Errorf("PATH should contain .local dir, got: %s", path)
				}
			} else {
				if !strings.Contains(path, "/usr/bin") {
					t.Errorf("PATH should contain /usr/bin, got: %s", path)
				}
				if !strings.Contains(path, "/bin") {
					t.Errorf("PATH should contain /bin, got: %s", path)
				}
			}
		}
	}

	if !pathFound {
		t.Error("PATH should be added to environment")
	}
}

func TestBuildEnv(t *testing.T) {
	step := NewScriptPreparationStep("echo test", time.Minute)

	prepCtx := &PreparationContext{
		PodID:            "test-pod",
		TicketSlug: "TICKET-123",
		WorkspaceDir:     "/workspace",
	}

	env := step.buildEnv(prepCtx)

	hasWorkspaceDir := false
	hasTicketSlug := false
	for _, e := range env {
		if e == "WORKSPACE_DIR=/workspace" {
			hasWorkspaceDir = true
		}
		if e == "TICKET_SLUG=TICKET-123" {
			hasTicketSlug = true
		}
	}

	if !hasWorkspaceDir {
		t.Error("env should contain WORKSPACE_DIR")
	}
	if !hasTicketSlug {
		t.Error("env should contain TICKET_SLUG")
	}
}
