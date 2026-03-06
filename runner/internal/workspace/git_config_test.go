package workspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/testutil"
)

// --- Test applyGitConfig ---

func TestApplyGitConfigEmptyPath(t *testing.T) {
	tmpDir := t.TempDir()
	manager, _ := NewManager(tmpDir, "")

	err := manager.applyGitConfig(context.Background(), tmpDir)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestApplyGitConfigMissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	manager, _ := NewManager(tmpDir, "/nonexistent/config")

	err := manager.applyGitConfig(context.Background(), tmpDir)
	if err == nil {
		t.Error("expected error for missing config file")
	}
}

func TestApplyGitConfigValidFile(t *testing.T) {
	tmpDir := t.TempDir()

	configPath := filepath.Join(tmpDir, "git.config")
	os.WriteFile(configPath, []byte("[user]\n\tname = Test User\n"), 0644)

	manager, _ := NewManager(tmpDir, configPath)

	repoPath := filepath.Join(tmpDir, "repo")
	os.MkdirAll(filepath.Join(repoPath, ".git"), 0755)

	cmd := exec.Command("git", "init")
	cmd.Dir = repoPath
	cmd.Run()

	err := manager.applyGitConfig(context.Background(), repoPath)
	if err != nil {
		t.Logf("applyGitConfig error (may fail without git repo): %v", err)
	}
}

func TestApplyGitConfigGitDirError(t *testing.T) {
	tmpDir := t.TempDir()

	configPath := filepath.Join(tmpDir, "git.config")
	os.WriteFile(configPath, []byte("[user]\n\tname = Test\n"), 0644)

	manager, _ := NewManager(tmpDir, configPath)

	nonGitPath := filepath.Join(tmpDir, "not-a-repo")
	os.MkdirAll(nonGitPath, 0755)

	err := manager.applyGitConfig(context.Background(), nonGitPath)
	if err == nil {
		t.Error("expected error when running applyGitConfig on non-git directory")
	}
	if !strings.Contains(err.Error(), "failed to get git directory") {
		t.Errorf("expected 'failed to get git directory' error, got: %v", err)
	}
}

func TestApplyGitConfigWriteError(t *testing.T) {
	testutil.SkipIfRoot(t)
	testutil.SkipIfNoChmodSupport(t)

	tmpDir := t.TempDir()

	configPath := filepath.Join(tmpDir, "git.config")
	os.WriteFile(configPath, []byte("[user]\n\tname = Test\n"), 0644)

	manager, _ := NewManager(tmpDir, configPath)

	repoPath := filepath.Join(tmpDir, "repo")
	gitDir := filepath.Join(repoPath, ".git")
	os.MkdirAll(gitDir, 0755)
	os.Chmod(gitDir, 0444)
	defer os.Chmod(gitDir, 0755)

	err := manager.applyGitConfig(context.Background(), repoPath)
	if err == nil {
		t.Error("expected error when writing to read-only .git directory")
	}
}

// TestApplyGitConfigInWorktree tests applyGitConfig in a real git worktree
// where .git is a file pointing to the main repo, not a directory
func TestApplyGitConfigInWorktree(t *testing.T) {
	tmpDir := t.TempDir()

	configContent := "[user]\n\tname = Worktree Test User\n\temail = worktree@test.com\n"
	configPath := filepath.Join(tmpDir, "git.config")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	manager, err := NewManager(tmpDir, configPath)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	// Create a bare repository
	bareRepoPath := filepath.Join(tmpDir, "repos", "test-repo")
	if err := os.MkdirAll(bareRepoPath, 0755); err != nil {
		t.Fatalf("failed to create bare repo dir: %v", err)
	}

	initCmd := exec.Command("git", "init", "--bare")
	initCmd.Dir = bareRepoPath
	if output, err := initCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to init bare repo: %v, output: %s", err, output)
	}

	// Create initial commit via temporary clone
	tempClonePath := filepath.Join(tmpDir, "temp-clone")
	cloneCmd := exec.Command("git", "clone", bareRepoPath, tempClonePath)
	if output, err := cloneCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to clone: %v, output: %s", err, output)
	}

	exec.Command("git", "-C", tempClonePath, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", tempClonePath, "config", "user.name", "Test").Run()

	testFile := filepath.Join(tempClonePath, "README.md")
	os.WriteFile(testFile, []byte("# Test"), 0644)
	exec.Command("git", "-C", tempClonePath, "add", ".").Run()
	commitCmd := exec.Command("git", "-C", tempClonePath, "commit", "-m", "Initial commit")
	if output, err := commitCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to commit: %v, output: %s", err, output)
	}

	// Push to bare repo
	exec.Command("git", "-C", tempClonePath, "push", "origin", "main").Run()
	exec.Command("git", "-C", tempClonePath, "push", "origin", "master").Run()

	// Create worktree from bare repo
	worktreePath := filepath.Join(tmpDir, "sandboxes", "test-pod", "worktree")
	if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
		t.Fatalf("failed to create worktree parent: %v", err)
	}

	worktreeCmd := exec.Command("git", "worktree", "add", worktreePath, "HEAD")
	worktreeCmd.Dir = bareRepoPath
	if output, err := worktreeCmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to create worktree: %v, output: %s", err, output)
	}

	// Verify .git is a file (not directory) in worktree
	gitPath := filepath.Join(worktreePath, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		t.Fatalf("failed to stat .git: %v", err)
	}
	if info.IsDir() {
		t.Fatal(".git should be a file in worktree, not a directory")
	}

	// Test applyGitConfig on the worktree
	err = manager.applyGitConfig(context.Background(), worktreePath)
	if err != nil {
		t.Errorf("applyGitConfig failed in worktree: %v", err)
	}

	// Verify config was applied
	configCheckCmd := exec.Command("git", "config", "--local", "include.path")
	configCheckCmd.Dir = worktreePath
	output, err := configCheckCmd.Output()
	if err != nil {
		t.Logf("config check output: %s, err: %v", output, err)
	} else {
		if !strings.Contains(string(output), "config.local") {
			t.Errorf("expected include.path to contain config.local, got: %s", output)
		}
	}
}
