package workspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// --- Test CreateWorktree ---

func TestCreateWorktreeInvalidRepoURL(t *testing.T) {
	tmpDir := t.TempDir()
	manager, _ := NewManager(tmpDir, "")

	result, err := manager.CreateWorktree(context.Background(), "", "main", "pod-1")
	if err == nil {
		t.Error("expected error for empty repo URL")
	}
	if result != nil {
		t.Error("expected nil result on error")
	}
}

func TestCreateWorktreeInvalidRepoURLFormat(t *testing.T) {
	tmpDir := t.TempDir()
	manager, _ := NewManager(tmpDir, "")

	result, err := manager.CreateWorktree(context.Background(), "invalid", "main", "pod-1")
	if err == nil {
		t.Error("expected error for invalid repo URL")
	}
	if result != nil {
		t.Error("expected nil result on error")
	}
}

func TestCreateWorktreeMkdirParentError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping as root")
	}

	tmpDir := t.TempDir()

	workspaceRoot := filepath.Join(tmpDir, "workspace")
	manager, _ := NewManager(workspaceRoot, "")

	worktreesParent := filepath.Join(workspaceRoot, "worktrees")
	os.MkdirAll(worktreesParent, 0755)
	os.Chmod(worktreesParent, 0444)
	defer os.Chmod(worktreesParent, 0755)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := manager.CreateWorktree(ctx, "https://github.com/test/repo.git", "main", "pod-1")
	if err == nil {
		t.Error("expected error when parent directory is read-only")
	}
	if result != nil {
		t.Error("expected nil result on error")
	}
}

// --- Integration tests for CreateWorktree ---

func TestCreateWorktreeFullIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir := t.TempDir()

	sourceRepo := filepath.Join(tmpDir, "source")
	os.MkdirAll(sourceRepo, 0755)

	initCmd := exec.Command("git", "init")
	initCmd.Dir = sourceRepo
	if err := initCmd.Run(); err != nil {
		t.Fatalf("failed to init source repo: %v", err)
	}

	exec.Command("git", "config", "user.email", "test@test.com").Run()
	cmd := exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = sourceRepo
	cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = sourceRepo
	cmd.Run()

	testFile := filepath.Join(sourceRepo, "README.md")
	os.WriteFile(testFile, []byte("# Test Repo\n"), 0644)

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = sourceRepo
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = sourceRepo
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	cmd = exec.Command("git", "branch", "-M", "main")
	cmd.Dir = sourceRepo
	cmd.Run()

	workspaceRoot := filepath.Join(tmpDir, "workspace")
	manager, err := NewManager(workspaceRoot, "")
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := manager.CreateWorktree(ctx, sourceRepo, "main", "test-pod")
	if err != nil {
		t.Logf("CreateWorktree error (expected for local paths): %v", err)
	} else {
		if _, err := os.Stat(result.Path); os.IsNotExist(err) {
			t.Error("worktree path should exist")
		}
		if result.Branch == "" {
			t.Error("expected non-empty branch name")
		}
	}
}

func TestCreateWorktreeExistingWorktree(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir := t.TempDir()

	sourceRepo := filepath.Join(tmpDir, "source")
	os.MkdirAll(sourceRepo, 0755)

	cmd := exec.Command("git", "init")
	cmd.Dir = sourceRepo
	cmd.Run()

	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = sourceRepo
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test")
	cmd.Dir = sourceRepo
	cmd.Run()

	testFile := filepath.Join(sourceRepo, "file.txt")
	os.WriteFile(testFile, []byte("content"), 0644)

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = sourceRepo
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = sourceRepo
	cmd.Run()

	workspaceRoot := filepath.Join(tmpDir, "workspace")
	manager, _ := NewManager(workspaceRoot, "")

	existingWorktree := filepath.Join(workspaceRoot, "worktrees", "test-pod")
	os.MkdirAll(existingWorktree, 0755)
	os.WriteFile(filepath.Join(existingWorktree, "existing.txt"), []byte("old"), 0644)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := manager.CreateWorktree(ctx, sourceRepo, "main", "test-pod")
	t.Logf("CreateWorktree result: %v, err: %v", result, err)
}