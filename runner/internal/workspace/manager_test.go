package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/testutil"
)

// --- Test Manager (workspace.go) ---

func TestNewManager(t *testing.T) {
	tmpDir := t.TempDir()
	root := filepath.Join(tmpDir, "workspace")

	manager, err := NewManager(root, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manager == nil {
		t.Fatal("NewManager returned nil")
		return
	}

	if manager.root != root {
		t.Errorf("root: got %v, want %v", manager.root, root)
	}

	// Root directory should be created
	if _, err := os.Stat(root); os.IsNotExist(err) {
		t.Error("root directory should be created")
	}
}

func TestNewManagerWithGitConfig(t *testing.T) {
	tmpDir := t.TempDir()
	root := filepath.Join(tmpDir, "workspace")
	gitConfigPath := filepath.Join(tmpDir, "git.config")

	manager, err := NewManager(root, gitConfigPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manager.gitConfigPath != gitConfigPath {
		t.Errorf("gitConfigPath: got %v, want %v", manager.gitConfigPath, gitConfigPath)
	}
}

func TestTempWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	manager, _ := NewManager(tmpDir, "")

	path := manager.TempWorkspace("pod-123")

	expectedPath := filepath.Join(tmpDir, "temp", "pod-123")
	if path != expectedPath {
		t.Errorf("path: got %v, want %v", path, expectedPath)
	}

	// Directory should be created
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("temp directory should be created")
	}
}

func TestGetWorkspaceRoot(t *testing.T) {
	tmpDir := t.TempDir()
	manager, _ := NewManager(tmpDir, "")

	if manager.GetWorkspaceRoot() != tmpDir {
		t.Errorf("GetWorkspaceRoot: got %v, want %v", manager.GetWorkspaceRoot(), tmpDir)
	}
}

func TestNewManagerCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	root := filepath.Join(tmpDir, "deep", "nested", "workspace")

	manager, err := NewManager(root, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if manager == nil {
		t.Fatal("NewManager returned nil")
		return
	}

	// Verify directory was created
	if _, err := os.Stat(root); os.IsNotExist(err) {
		t.Error("nested directory should be created")
	}
}

// TestNewManagerError tests NewManager with invalid path
func TestNewManagerError(t *testing.T) {
	// Try to create manager in a read-only location (will likely fail on most systems)
	// Skip if running as root
	testutil.SkipIfRoot(t)
	testutil.SkipIfNoChmodSupport(t)

	// Create a read-only directory
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	os.MkdirAll(readOnlyDir, 0755)
	os.Chmod(readOnlyDir, 0444)
	defer os.Chmod(readOnlyDir, 0755)

	// Try to create workspace inside read-only dir
	root := filepath.Join(readOnlyDir, "workspace")
	_, err := NewManager(root, "")
	if err == nil {
		t.Error("expected error for read-only parent directory")
	}
}
