package runner

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/AgentsMesh/AgentsMesh/runner/internal/config"
)

func TestGetSandboxStatus_NotExists(t *testing.T) {
	tempDir := t.TempDir()
	r := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	status := r.GetSandboxStatus("nonexistent-pod")

	if status.PodKey != "nonexistent-pod" {
		t.Errorf("PodKey = %s, want nonexistent-pod", status.PodKey)
	}
	if status.Exists {
		t.Error("Exists should be false for nonexistent sandbox")
	}
	if status.CanResume {
		t.Error("CanResume should be false for nonexistent sandbox")
	}
}

func TestGetSandboxStatus_ExistsWithoutWorkspace(t *testing.T) {
	tempDir := t.TempDir()
	podKey := "test-pod"

	// Create sandbox directory without workspace
	sandboxPath := filepath.Join(tempDir, "sandboxes", podKey)
	if err := os.MkdirAll(sandboxPath, 0755); err != nil {
		t.Fatalf("Failed to create sandbox dir: %v", err)
	}

	r := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	status := r.GetSandboxStatus(podKey)

	if status.PodKey != podKey {
		t.Errorf("PodKey = %s, want %s", status.PodKey, podKey)
	}
	if !status.Exists {
		t.Error("Exists should be true")
	}
	if !status.CanResume {
		t.Error("CanResume should be true when sandbox exists")
	}
	if status.SandboxPath != sandboxPath {
		t.Errorf("SandboxPath = %s, want %s", status.SandboxPath, sandboxPath)
	}
	if status.LastModified == 0 {
		t.Error("LastModified should be set")
	}
}

func TestGetSandboxStatus_ExistsWithWorkspace(t *testing.T) {
	tempDir := t.TempDir()
	podKey := "workspace-pod"

	// Create sandbox with workspace directory
	workspacePath := filepath.Join(tempDir, "sandboxes", podKey, "workspace")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	r := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	status := r.GetSandboxStatus(podKey)

	if !status.Exists {
		t.Error("Exists should be true")
	}
	if !status.CanResume {
		t.Error("CanResume should be true")
	}
	if status.LastModified == 0 {
		t.Error("LastModified should be set from workspace")
	}
}

func TestGetSandboxStatus_WithGitRepo(t *testing.T) {
	tempDir := t.TempDir()
	podKey := "git-pod"

	// Create sandbox with workspace directory and git repo
	workspacePath := filepath.Join(tempDir, "sandboxes", podKey, "workspace")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	// Initialize git repo
	r := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	// Run git init
	_, err := r.runGitCommand(workspacePath, "init")
	if err != nil {
		t.Skipf("Git not available: %v", err)
	}

	// Configure git user
	r.runGitCommand(workspacePath, "config", "user.email", "test@test.com")
	r.runGitCommand(workspacePath, "config", "user.name", "Test")

	// Create a file and commit
	testFile := filepath.Join(workspacePath, "test.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)
	r.runGitCommand(workspacePath, "add", ".")
	r.runGitCommand(workspacePath, "commit", "-m", "initial commit")

	status := r.GetSandboxStatus(podKey)

	if !status.Exists {
		t.Error("Exists should be true")
	}
	if !status.CanResume {
		t.Error("CanResume should be true")
	}
	if status.BranchName == "" {
		t.Log("Branch name not detected (might be 'master' or 'main' depending on git config)")
	}
	if status.CurrentCommit == "" {
		t.Error("CurrentCommit should be set after commit")
	}
	if len(status.CurrentCommit) > 8 {
		t.Errorf("CurrentCommit should be truncated to 8 chars, got %s", status.CurrentCommit)
	}
}

func TestGetSandboxStatus_WithUncommittedChanges(t *testing.T) {
	tempDir := t.TempDir()
	podKey := "changes-pod"

	// Create sandbox with workspace directory and git repo
	workspacePath := filepath.Join(tempDir, "sandboxes", podKey, "workspace")
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		t.Fatalf("Failed to create workspace dir: %v", err)
	}

	r := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	// Initialize git repo
	_, err := r.runGitCommand(workspacePath, "init")
	if err != nil {
		t.Skipf("Git not available: %v", err)
	}

	// Configure git user and make initial commit
	r.runGitCommand(workspacePath, "config", "user.email", "test@test.com")
	r.runGitCommand(workspacePath, "config", "user.name", "Test")
	testFile := filepath.Join(workspacePath, "test.txt")
	os.WriteFile(testFile, []byte("initial"), 0644)
	r.runGitCommand(workspacePath, "add", ".")
	r.runGitCommand(workspacePath, "commit", "-m", "initial")

	// Create uncommitted change
	os.WriteFile(testFile, []byte("modified"), 0644)

	status := r.GetSandboxStatus(podKey)

	if !status.HasUncommittedChanges {
		t.Error("HasUncommittedChanges should be true")
	}
}

func TestGetGitInfo_NotGitRepo(t *testing.T) {
	tempDir := t.TempDir()
	r := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	info := r.getGitInfo(tempDir)

	if info.RepositoryURL != "" {
		t.Error("RepositoryURL should be empty for non-git directory")
	}
	if info.BranchName != "" {
		t.Error("BranchName should be empty for non-git directory")
	}
	if info.CurrentCommit != "" {
		t.Error("CurrentCommit should be empty for non-git directory")
	}
	if info.HasUncommittedChanges {
		t.Error("HasUncommittedChanges should be false for non-git directory")
	}
}

func TestRunGitCommand_Success(t *testing.T) {
	tempDir := t.TempDir()
	r := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	// Run git version (should work on any system with git)
	out, err := r.runGitCommand(tempDir, "version")
	if err != nil {
		t.Skipf("Git not available: %v", err)
	}

	if out == "" {
		t.Error("Expected non-empty output from git version")
	}
}

func TestRunGitCommand_InvalidCommand(t *testing.T) {
	tempDir := t.TempDir()
	r := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	_, err := r.runGitCommand(tempDir, "invalid-command-that-does-not-exist")
	if err == nil {
		t.Error("Expected error for invalid git command")
	}
}

func TestGetDirSize(t *testing.T) {
	tempDir := t.TempDir()
	r := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	// Create some files
	os.WriteFile(filepath.Join(tempDir, "file1.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(tempDir, "file2.txt"), []byte("world!"), 0644)

	size := r.getDirSize(tempDir)

	// "hello" = 5 bytes, "world!" = 6 bytes = 11 total
	if size != 11 {
		t.Errorf("Size = %d, want 11", size)
	}
}

func TestGetDirSize_EmptyDir(t *testing.T) {
	tempDir := t.TempDir()
	r := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	size := r.getDirSize(tempDir)

	if size != 0 {
		t.Errorf("Size = %d, want 0 for empty directory", size)
	}
}

func TestGetDirSize_WithSubdirectories(t *testing.T) {
	tempDir := t.TempDir()
	r := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: tempDir,
		},
	}

	// Create subdirectory with file
	subdir := filepath.Join(tempDir, "sub")
	os.MkdirAll(subdir, 0755)
	os.WriteFile(filepath.Join(subdir, "nested.txt"), []byte("nested content"), 0644)
	os.WriteFile(filepath.Join(tempDir, "root.txt"), []byte("root"), 0644)

	size := r.getDirSize(tempDir)

	// "nested content" = 14, "root" = 4 = 18 total
	if size != 18 {
		t.Errorf("Size = %d, want 18", size)
	}
}
