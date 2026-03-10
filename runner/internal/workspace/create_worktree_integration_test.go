//go:build integration

package workspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// --- Additional CreateWorktree Integration Tests ---

func TestCreateWorktreeWithGitConfig(t *testing.T) {
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

	cmd = exec.Command("git", "branch", "-M", "main")
	cmd.Dir = sourceRepo
	cmd.Run()

	configPath := filepath.Join(tmpDir, "git.config")
	os.WriteFile(configPath, []byte("[user]\n\tname = Custom User\n"), 0644)

	workspaceRoot := filepath.Join(tmpDir, "workspace")
	manager, _ := NewManager(workspaceRoot, configPath)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := manager.CreateWorktree(ctx, sourceRepo, "main", "test-pod")
	t.Logf("CreateWorktree with git config result: %v, err: %v", result, err)
}

func TestCreateWorktreeNonMainBranch(t *testing.T) {
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

	cmd = exec.Command("git", "checkout", "-b", "feature/test")
	cmd.Dir = sourceRepo
	cmd.Run()

	workspaceRoot := filepath.Join(tmpDir, "workspace")
	manager, _ := NewManager(workspaceRoot, "")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := manager.CreateWorktree(ctx, sourceRepo, "feature/test", "test-pod")
	t.Logf("CreateWorktree with feature branch result: %v, err: %v", result, err)
}
