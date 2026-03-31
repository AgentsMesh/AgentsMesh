//go:build integration

package workspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createBareRepo initialises a bare repo with one commit on "main" and returns
// the path to the bare repo directory.
func createBareRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	bare := filepath.Join(dir, "test-repo.git")
	clone := filepath.Join(dir, "clone")

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = clone
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "cmd %v failed: %s", args, out)
	}

	require.NoError(t, exec.Command("git", "init", "--bare", bare).Run())
	require.NoError(t, exec.Command("git", "clone", bare, clone).Run())

	run("git", "config", "user.email", "test@test.com")
	run("git", "config", "user.name", "Test")
	require.NoError(t, os.WriteFile(filepath.Join(clone, "README.md"), []byte("hello"), 0644))
	run("git", "add", ".")
	run("git", "commit", "-m", "init")
	run("git", "branch", "-M", "main")
	run("git", "push", "-u", "origin", "main")

	return bare
}

// TestWorkspace_CreateAndRemoveWorktree_Integration creates a worktree from a
// local bare repo and then removes it, verifying both operations.
func TestWorkspace_CreateAndRemoveWorktree_Integration(t *testing.T) {
	bare := createBareRepo(t)
	root := t.TempDir()
	mgr, err := NewManager(root, "")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := mgr.CreateWorktree(ctx, bare, "main", "pod-1")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.DirExists(t, result.Path)
	assert.FileExists(t, filepath.Join(result.Path, "README.md"))
	assert.NotEmpty(t, result.Branch)

	// Remove worktree.
	require.NoError(t, mgr.RemoveWorktree(ctx, result.Path))
	assert.NoDirExists(t, result.Path)
}

// TestWorkspace_MultipleWorktrees_Integration creates three worktrees from the
// same bare repo and verifies they are independent.
func TestWorkspace_MultipleWorktrees_Integration(t *testing.T) {
	bare := createBareRepo(t)
	root := t.TempDir()
	mgr, err := NewManager(root, "")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var results []*WorktreeResult
	for _, pod := range []string{"pod-a", "pod-b", "pod-c"} {
		r, err := mgr.CreateWorktree(ctx, bare, "main", pod)
		require.NoError(t, err, "failed to create worktree for %s", pod)
		results = append(results, r)
	}

	// All paths should be unique.
	seen := map[string]bool{}
	for _, r := range results {
		assert.False(t, seen[r.Path], "duplicate worktree path: %s", r.Path)
		seen[r.Path] = true
		assert.DirExists(t, r.Path)
	}

	// A commit in worktree-0 should NOT appear in worktree-1.
	newFile := filepath.Join(results[0].Path, "extra.txt")
	require.NoError(t, os.WriteFile(newFile, []byte("x"), 0644))
	assert.NoFileExists(t, filepath.Join(results[1].Path, "extra.txt"))

	// Cleanup.
	for _, r := range results {
		require.NoError(t, mgr.RemoveWorktree(ctx, r.Path))
	}
}

// TestWorkspace_WorktreeWithOptions_Integration verifies that WithGitToken does
// not cause an error for local repos (the token is simply unused).
func TestWorkspace_WorktreeWithOptions_Integration(t *testing.T) {
	bare := createBareRepo(t)
	root := t.TempDir()
	mgr, err := NewManager(root, "")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	wtPath := filepath.Join(root, "sandboxes", "pod-opt", "workspace")
	result, err := mgr.CreateWorktreeWithOptions(ctx, bare, "main", wtPath, WithGitToken("unused-token"))
	require.NoError(t, err)
	assert.DirExists(t, result.Path)
	assert.FileExists(t, filepath.Join(result.Path, "README.md"))

	require.NoError(t, mgr.RemoveWorktree(ctx, result.Path))
}

// TestWorkspace_CloneError_Integration tries to create a worktree from a
// non-existent URL and verifies an error is returned with no partial state.
func TestWorkspace_CloneError_Integration(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	root := t.TempDir()
	mgr, err := NewManager(root, "")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = mgr.CreateWorktree(ctx, "/nonexistent/repo.git", "main", "pod-fail")
	require.Error(t, err)

	// Verify no partial worktree directory was left behind.
	wtPath := filepath.Join(root, "sandboxes", "pod-fail", "workspace")
	assert.NoDirExists(t, wtPath)
}

// TestWorkspace_DefaultBranchFallback_Integration creates a bare repo whose
// default branch is "master" and verifies CreateWorktree falls back correctly
// when "main" is not found.
func TestWorkspace_DefaultBranchFallback_Integration(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	bare := filepath.Join(dir, "master-repo.git")
	clone := filepath.Join(dir, "clone")

	require.NoError(t, exec.Command("git", "init", "--bare", bare).Run())
	require.NoError(t, exec.Command("git", "clone", bare, clone).Run())

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = clone
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "%v: %s", args, out)
	}

	run("git", "config", "user.email", "t@t.com")
	run("git", "config", "user.name", "T")
	require.NoError(t, os.WriteFile(filepath.Join(clone, "f.txt"), []byte("x"), 0644))
	run("git", "add", ".")
	run("git", "commit", "-m", "init")
	run("git", "branch", "-M", "master")
	run("git", "push", "-u", "origin", "master")

	root := t.TempDir()
	mgr, err := NewManager(root, "")
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Request "main" — should fallback to "master".
	result, err := mgr.CreateWorktree(ctx, bare, "main", "pod-master")
	require.NoError(t, err)
	assert.DirExists(t, result.Path)
	assert.FileExists(t, filepath.Join(result.Path, "f.txt"))
	// The actual branch should contain "master" or be a worktree branch.
	assert.True(t, strings.Contains(result.Branch, "master") ||
		strings.Contains(result.Branch, "worktree"),
		"unexpected branch: %s", result.Branch)
}
