package autopilot

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDefaultGitExecutor_Status(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Configure git user for commits
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run()

	executor := NewDefaultGitExecutor()

	// Test status on clean repo
	output, err := executor.Status(tmpDir)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if len(output) != 0 {
		t.Errorf("Expected empty status, got: %s", output)
	}

	// Create an untracked file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test status with untracked file
	output, err = executor.Status(tmpDir)
	if err != nil {
		t.Fatalf("Status failed: %v", err)
	}
	if len(output) == 0 {
		t.Error("Expected non-empty status with untracked file")
	}
}

func TestDefaultGitExecutor_DiffStat(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Configure git user for commits
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run()

	executor := NewDefaultGitExecutor()

	// Create and commit a file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	exec.Command("git", "-C", tmpDir, "add", "test.txt").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()

	// Test diff on clean working tree
	output, err := executor.DiffStat(tmpDir)
	if err != nil {
		t.Fatalf("DiffStat failed: %v", err)
	}
	if len(output) != 0 {
		t.Errorf("Expected empty diff, got: %s", output)
	}

	// Modify the file
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Test diff with changes
	output, err = executor.DiffStat(tmpDir)
	if err != nil {
		t.Fatalf("DiffStat failed: %v", err)
	}
	if len(output) == 0 {
		t.Error("Expected non-empty diff with modified file")
	}
}

func TestDefaultGitExecutor_DiffCachedStat(t *testing.T) {
	// Create a temporary git repository
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Skipf("git not available: %v", err)
	}

	// Configure git user for commits
	exec.Command("git", "-C", tmpDir, "config", "user.email", "test@test.com").Run()
	exec.Command("git", "-C", tmpDir, "config", "user.name", "Test").Run()

	executor := NewDefaultGitExecutor()

	// Create and commit a file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	exec.Command("git", "-C", tmpDir, "add", "test.txt").Run()
	exec.Command("git", "-C", tmpDir, "commit", "-m", "initial").Run()

	// Test cached diff on clean staging area
	output, err := executor.DiffCachedStat(tmpDir)
	if err != nil {
		t.Fatalf("DiffCachedStat failed: %v", err)
	}
	if len(output) != 0 {
		t.Errorf("Expected empty cached diff, got: %s", output)
	}

	// Modify and stage the file
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}
	exec.Command("git", "-C", tmpDir, "add", "test.txt").Run()

	// Test cached diff with staged changes
	output, err = executor.DiffCachedStat(tmpDir)
	if err != nil {
		t.Fatalf("DiffCachedStat failed: %v", err)
	}
	if len(output) == 0 {
		t.Error("Expected non-empty cached diff with staged file")
	}
}

func TestDefaultGitExecutor_NonGitDir(t *testing.T) {
	// Create a temporary non-git directory
	tmpDir := t.TempDir()

	executor := NewDefaultGitExecutor()

	// All operations should fail on non-git directory
	_, err := executor.Status(tmpDir)
	if err == nil {
		t.Error("Expected error for Status on non-git directory")
	}

	_, err = executor.DiffStat(tmpDir)
	if err == nil {
		t.Error("Expected error for DiffStat on non-git directory")
	}

	_, err = executor.DiffCachedStat(tmpDir)
	if err == nil {
		t.Error("Expected error for DiffCachedStat on non-git directory")
	}
}

// MockGitExecutor is a mock implementation for testing.
type MockGitExecutor struct {
	StatusOutput         []byte
	StatusErr            error
	DiffStatOutput       []byte
	DiffStatErr          error
	DiffCachedStatOutput []byte
	DiffCachedStatErr    error
}

func (m *MockGitExecutor) Status(workDir string) ([]byte, error) {
	return m.StatusOutput, m.StatusErr
}

func (m *MockGitExecutor) DiffStat(workDir string) ([]byte, error) {
	return m.DiffStatOutput, m.DiffStatErr
}

func (m *MockGitExecutor) DiffCachedStat(workDir string) ([]byte, error) {
	return m.DiffCachedStatOutput, m.DiffCachedStatErr
}

// Ensure MockGitExecutor implements GitExecutor.
var _ GitExecutor = (*MockGitExecutor)(nil)

func TestProgressTracker_WithMockGitExecutor(t *testing.T) {
	tmpDir := t.TempDir()

	// Create .git directory to pass the git repository check
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	mockExecutor := &MockGitExecutor{
		StatusOutput: []byte(" M file1.go\n?? file2.go\n"),
		DiffStatOutput: []byte(` file1.go | 10 +++++++---
 1 file changed, 7 insertions(+), 3 deletions(-)`),
	}

	pt := NewProgressTracker(ProgressTrackerConfig{
		WorkDir:     tmpDir,
		GitExecutor: mockExecutor,
	})

	snapshot := pt.CaptureSnapshot()
	if snapshot == nil {
		t.Fatal("CaptureSnapshot returned nil")
		return // unreachable, satisfies staticcheck SA5011
	}

	if !snapshot.GitDiff.HasChanges {
		t.Error("Expected HasChanges to be true")
	}

	if len(snapshot.GitDiff.FilesChanged) != 2 {
		t.Errorf("Expected 2 files changed, got %d", len(snapshot.GitDiff.FilesChanged))
	}

	if len(snapshot.GitDiff.UntrackedFiles) != 1 {
		t.Errorf("Expected 1 untracked file, got %d", len(snapshot.GitDiff.UntrackedFiles))
	}

	if snapshot.GitDiff.Insertions != 7 {
		t.Errorf("Expected 7 insertions, got %d", snapshot.GitDiff.Insertions)
	}

	if snapshot.GitDiff.Deletions != 3 {
		t.Errorf("Expected 3 deletions, got %d", snapshot.GitDiff.Deletions)
	}
}
