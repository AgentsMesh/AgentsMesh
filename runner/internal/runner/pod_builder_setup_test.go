package runner

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetup_LocalPath_UsesSandboxRootOverride verifies that setup() replaces
// the newly created sandbox root with the source pod's sandbox (LocalPath)
// when using LocalPathStrategy.
func TestSetup_LocalPath_UsesSandboxRootOverride(t *testing.T) {
	workspaceRoot := t.TempDir()

	// Simulate the source pod's sandbox (what LocalPath points to)
	sourceSandbox := filepath.Join(workspaceRoot, "sandboxes", "source-pod-key")
	sourceWorkspace := filepath.Join(sourceSandbox, "workspace")
	require.NoError(t, os.MkdirAll(sourceWorkspace, 0755))

	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: workspaceRoot,
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "resume-pod-key",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"test"},
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
		SandboxConfig: &runnerv1.SandboxConfig{
			LocalPath: sourceSandbox,
		},
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)
	sandboxRoot, workingDir, _, err := builder.setup(context.Background())
	require.NoError(t, err)

	// sandboxRoot should be overridden to the source sandbox
	assert.Equal(t, sourceSandbox, sandboxRoot,
		"sandboxRoot should be overridden to source pod's sandbox")
	assert.Equal(t, sourceWorkspace, workingDir,
		"workingDir should point to workspace inside source sandbox")

	// The new sandbox directory (for resume-pod-key) should have been cleaned up
	newSandbox := filepath.Join(workspaceRoot, "sandboxes", "resume-pod-key")
	_, err = os.Stat(newSandbox)
	assert.True(t, os.IsNotExist(err),
		"newly created sandbox should be cleaned up after override")
}

// TestCreateFiles_ResumeMode_McpJsonInWorkDir verifies that .mcp.json is
// correctly created inside the working directory when resuming a pod.
// This was the production bug: path template {{.sandbox.work_dir}}/.mcp.json
// resolved to the old sandbox which escaped the new sandbox's validation.
func TestCreateFiles_ResumeMode_McpJsonInWorkDir(t *testing.T) {
	workspaceRoot := t.TempDir()

	// Simulate the source pod's sandbox
	sourceSandbox := filepath.Join(workspaceRoot, "sandboxes", "source-pod")
	sourceWorkspace := filepath.Join(sourceSandbox, "workspace")
	require.NoError(t, os.MkdirAll(sourceWorkspace, 0755))

	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: workspaceRoot,
		},
	}

	mcpContent := `{"mcpServers": {"test": {}}}`
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "resume-mcp-pod",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"test"},
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
		SandboxConfig: &runnerv1.SandboxConfig{
			LocalPath: sourceSandbox,
		},
		FilesToCreate: []*runnerv1.FileToCreate{
			{
				Path:    "{{.sandbox.work_dir}}/.mcp.json",
				Content: mcpContent,
				Mode:    0644,
			},
		},
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)
	sandboxRoot, workingDir, _, err := builder.setup(context.Background())
	require.NoError(t, err)

	// Verify sandbox root was overridden
	assert.Equal(t, sourceSandbox, sandboxRoot)

	// Verify .mcp.json was created in the correct location
	mcpPath := filepath.Join(workingDir, ".mcp.json")
	data, err := os.ReadFile(mcpPath)
	require.NoError(t, err, ".mcp.json should exist in working directory")
	assert.Equal(t, mcpContent, string(data))
}

// TestCreateFiles_ResumeMode_RootPathTemplate verifies that {{.sandbox.root_path}}
// also resolves correctly in resume mode (points to source sandbox, not new empty one).
func TestCreateFiles_ResumeMode_RootPathTemplate(t *testing.T) {
	workspaceRoot := t.TempDir()

	sourceSandbox := filepath.Join(workspaceRoot, "sandboxes", "source-pod-root")
	sourceWorkspace := filepath.Join(sourceSandbox, "workspace")
	require.NoError(t, os.MkdirAll(sourceWorkspace, 0755))

	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: workspaceRoot,
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "resume-root-tpl-pod",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"test"},
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
		SandboxConfig: &runnerv1.SandboxConfig{
			LocalPath: sourceSandbox,
		},
		FilesToCreate: []*runnerv1.FileToCreate{
			{
				Path:    "{{.sandbox.root_path}}/config.yaml",
				Content: "key: value",
				Mode:    0644,
			},
		},
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)
	sandboxRoot, _, _, err := builder.setup(context.Background())
	require.NoError(t, err)

	// {{.sandbox.root_path}} should resolve to source sandbox
	configPath := filepath.Join(sandboxRoot, "config.yaml")
	data, err := os.ReadFile(configPath)
	require.NoError(t, err, "config.yaml should exist in source sandbox root")
	assert.Equal(t, "key: value", string(data))
}

// TestSetup_LocalPath_ErrorDoesNotDeleteSourceSandbox verifies that if file
// creation fails in resume mode, the source pod's sandbox is NOT deleted.
// This is critical: the error cleanup path must not destroy the source data.
func TestSetup_LocalPath_ErrorDoesNotDeleteSourceSandbox(t *testing.T) {
	workspaceRoot := t.TempDir()

	sourceSandbox := filepath.Join(workspaceRoot, "sandboxes", "source-pod-err")
	sourceWorkspace := filepath.Join(sourceSandbox, "workspace")
	require.NoError(t, os.MkdirAll(sourceWorkspace, 0755))

	// Place a sentinel file in source sandbox to verify it's preserved
	sentinelPath := filepath.Join(sourceWorkspace, "important-data.txt")
	require.NoError(t, os.WriteFile(sentinelPath, []byte("do not delete"), 0644))

	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: workspaceRoot,
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "resume-error-pod",
		LaunchCommand: "echo",
		LaunchArgs:    []string{"test"},
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
		SandboxConfig: &runnerv1.SandboxConfig{
			LocalPath: sourceSandbox,
		},
		FilesToCreate: []*runnerv1.FileToCreate{
			{
				// This path escapes even the overridden sandbox → triggers error
				Path:    "/tmp/outside-sandbox/evil.txt",
				Content: "should fail",
				Mode:    0644,
			},
		},
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)
	_, _, _, err := builder.setup(context.Background())
	require.Error(t, err, "setup should fail due to path escape")

	// Critical: source sandbox must NOT be deleted
	_, statErr := os.Stat(sourceSandbox)
	assert.NoError(t, statErr, "source sandbox directory must still exist after error")

	// Sentinel file must be preserved
	data, readErr := os.ReadFile(sentinelPath)
	assert.NoError(t, readErr, "sentinel file must still exist after error")
	assert.Equal(t, "do not delete", string(data))
}
