package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/testutil"
)

// --- Test CleanupOldWorktrees ---

func TestCleanupOldWorktreesEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	manager, _ := NewManager(tmpDir, "")

	err := manager.CleanupOldWorktrees(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCleanupOldWorktreesInvalid(t *testing.T) {
	tmpDir := t.TempDir()
	manager, _ := NewManager(tmpDir, "")

	// Create sandbox with an invalid worktree (no .git)
	sandboxesDir := filepath.Join(tmpDir, "sandboxes")
	invalidWT := filepath.Join(sandboxesDir, "invalid-pod", "worktree")
	os.MkdirAll(invalidWT, 0755)

	err := manager.CleanupOldWorktrees(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(invalidWT); !os.IsNotExist(err) {
		t.Error("invalid worktree should be removed")
	}
}

func TestCleanupOldWorktreesWithValidWorktree(t *testing.T) {
	tmpDir := t.TempDir()
	manager, _ := NewManager(tmpDir, "")

	worktreesDir := filepath.Join(tmpDir, "worktrees")
	validWT := filepath.Join(worktreesDir, "valid")
	os.MkdirAll(validWT, 0755)

	gitFile := filepath.Join(validWT, ".git")
	os.WriteFile(gitFile, []byte("gitdir: /some/path"), 0644)

	err := manager.CleanupOldWorktrees(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(validWT); os.IsNotExist(err) {
		t.Error("valid worktree should not be removed")
	}
}

func TestCleanupOldWorktreesWithFile(t *testing.T) {
	tmpDir := t.TempDir()
	manager, _ := NewManager(tmpDir, "")

	worktreesDir := filepath.Join(tmpDir, "worktrees")
	os.MkdirAll(worktreesDir, 0755)

	testFile := filepath.Join(worktreesDir, "testfile")
	os.WriteFile(testFile, []byte("test"), 0644)

	err := manager.CleanupOldWorktrees(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("file should not be removed")
	}
}

func TestCleanupOldWorktreesReadDirError(t *testing.T) {
	testutil.SkipIfRoot(t)
	testutil.SkipIfNoChmodSupport(t)

	tmpDir := t.TempDir()
	manager, _ := NewManager(tmpDir, "")

	sandboxesDir := filepath.Join(tmpDir, "sandboxes")
	os.MkdirAll(sandboxesDir, 0755)

	subDir := filepath.Join(sandboxesDir, "subdir")
	os.MkdirAll(subDir, 0755)
	os.Chmod(sandboxesDir, 0000)
	defer os.Chmod(sandboxesDir, 0755)

	err := manager.CleanupOldWorktrees(context.Background())
	if err == nil {
		t.Error("expected error for unreadable directory")
	}
}
