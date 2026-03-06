package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/testutil"
)

// --- Test ListWorktrees ---

func TestListWorktreesEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	manager, _ := NewManager(tmpDir, "")

	worktrees, err := manager.ListWorktrees()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if worktrees != nil && len(worktrees) > 0 {
		t.Errorf("worktrees should be empty, got %v", worktrees)
	}
}

func TestListWorktreesWithDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	manager, _ := NewManager(tmpDir, "")

	// Create sandboxes with worktree subdirectories
	// New structure: sandboxes/{podKey}/worktree
	sandboxesDir := filepath.Join(tmpDir, "sandboxes")
	os.MkdirAll(filepath.Join(sandboxesDir, "pod1", "worktree"), 0755)
	os.MkdirAll(filepath.Join(sandboxesDir, "pod2", "worktree"), 0755)

	worktrees, err := manager.ListWorktrees()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(worktrees) != 2 {
		t.Errorf("worktrees count: got %v, want 2", len(worktrees))
	}
}

func TestListWorktreesReadError(t *testing.T) {
	testutil.SkipIfRoot(t)
	testutil.SkipIfNoChmodSupport(t)

	tmpDir := t.TempDir()
	manager, _ := NewManager(tmpDir, "")

	sandboxesDir := filepath.Join(tmpDir, "sandboxes")
	os.MkdirAll(sandboxesDir, 0755)
	os.Chmod(sandboxesDir, 0000)
	defer os.Chmod(sandboxesDir, 0755)

	_, err := manager.ListWorktrees()
	if err == nil {
		t.Error("expected error for unreadable directory")
	}
}
