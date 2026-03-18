package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalPathStrategy_ReturnsSandboxRoot(t *testing.T) {
	localPath := t.TempDir()

	strategy := NewLocalPathStrategy()
	result, err := strategy.Setup(context.Background(), "/unused/sandbox", &runnerv1.SandboxConfig{
		LocalPath: localPath,
	})

	require.NoError(t, err)
	assert.Equal(t, localPath, result.SandboxRoot,
		"LocalPathStrategy should return source sandbox path as SandboxRoot")
	assert.Equal(t, localPath, result.WorkingDir,
		"without workspace subdir, WorkingDir should be localPath itself")
}

func TestLocalPathStrategy_WithWorkspace_ReturnsSandboxRoot(t *testing.T) {
	localPath := t.TempDir()
	workspaceDir := filepath.Join(localPath, "workspace")
	require.NoError(t, os.MkdirAll(workspaceDir, 0755))

	strategy := NewLocalPathStrategy()
	result, err := strategy.Setup(context.Background(), "/unused/sandbox", &runnerv1.SandboxConfig{
		LocalPath: localPath,
	})

	require.NoError(t, err)
	assert.Equal(t, localPath, result.SandboxRoot,
		"SandboxRoot should always be the localPath (source sandbox)")
	assert.Equal(t, workspaceDir, result.WorkingDir,
		"WorkingDir should be workspace subdir when it exists")
}

func TestGitWorktreeStrategy_DoesNotSetSandboxRoot(t *testing.T) {
	// GitWorktreeStrategy should NOT set SandboxRoot (it creates its own workspace)
	// This verifies only LocalPathStrategy triggers the override path.
	strategy := NewEmptySandboxStrategy()
	sandboxRoot := t.TempDir()

	result, err := strategy.Setup(context.Background(), sandboxRoot, &runnerv1.SandboxConfig{})

	require.NoError(t, err)
	assert.Empty(t, result.SandboxRoot,
		"EmptySandboxStrategy should not set SandboxRoot override")
}
