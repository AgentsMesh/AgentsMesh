package workspace

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/testutil"
)

// --- Test ensureRepository ---

func TestEnsureRepositoryClone(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir := t.TempDir()
	manager, _ := NewManager(tmpDir, "")

	// Create a source repo
	sourceRepo := filepath.Join(tmpDir, "source")
	os.MkdirAll(sourceRepo, 0755)

	cmd := exec.Command("git", "init")
	cmd.Dir = sourceRepo
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init source repo: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = sourceRepo
	cmd.Run()

	cmd = exec.Command("git", "config", "user.name", "Test")
	cmd.Dir = sourceRepo
	cmd.Run()

	testFile := filepath.Join(sourceRepo, "README.md")
	os.WriteFile(testFile, []byte("# Test\n"), 0644)

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = sourceRepo
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = sourceRepo
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	// Clone with ensureRepository
	destPath := filepath.Join(tmpDir, "clone")
	err := manager.ensureRepository(context.Background(), sourceRepo, destPath)
	if err != nil {
		t.Fatalf("ensureRepository clone failed: %v", err)
	}

	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Error("clone directory should exist")
	}
}

func TestEnsureRepositoryFetch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	tmpDir := t.TempDir()
	manager, _ := NewManager(tmpDir, "")

	// Create a source repo
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

	testFile := filepath.Join(sourceRepo, "README.md")
	os.WriteFile(testFile, []byte("# Test\n"), 0644)

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = sourceRepo
	cmd.Run()

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = sourceRepo
	cmd.Run()

	// First clone
	destPath := filepath.Join(tmpDir, "clone")
	err := manager.ensureRepository(context.Background(), sourceRepo, destPath)
	if err != nil {
		t.Fatalf("initial clone failed: %v", err)
	}

	// Create .git marker to trigger fetch path
	os.MkdirAll(filepath.Join(destPath, ".git"), 0755)

	// Second call should fetch
	err = manager.ensureRepository(context.Background(), sourceRepo, destPath)
	t.Logf("fetch result: %v", err)
}

func TestEnsureRepositoryMkdirError(t *testing.T) {
	testutil.SkipIfRoot(t)
	testutil.SkipIfNoChmodSupport(t)

	tmpDir := t.TempDir()
	manager, _ := NewManager(tmpDir, "")

	readOnlyDir := filepath.Join(tmpDir, "readonly")
	os.MkdirAll(readOnlyDir, 0755)
	os.Chmod(readOnlyDir, 0444)
	defer os.Chmod(readOnlyDir, 0755)

	destPath := filepath.Join(readOnlyDir, "nested", "repo")
	err := manager.ensureRepository(context.Background(), "file:///fake", destPath)
	if err == nil {
		t.Error("expected error for mkdir in read-only directory")
	}
}
