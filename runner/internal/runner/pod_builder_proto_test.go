package runner

import (
	"context"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for the proto-driven Build() path after AgentFile eval migration to Backend.
// Covers: prompt injection, placeholder resolution, InteractionMode routing.

// === Prompt Injection ===

func TestBuild_PromptPrepend(t *testing.T) {
	cmd := protoCmd("echo", []string{"--flag", "value"})
	cmd.Prompt = "Fix the bug"
	cmd.PromptPosition = "prepend"

	pod := buildAndCleanup(t, cmd)
	require.True(t, len(pod.LaunchArgs) >= 3)
	assert.Equal(t, "Fix the bug", pod.LaunchArgs[0], "prompt should be first arg")
	assert.Equal(t, "--flag", pod.LaunchArgs[1])
}

func TestBuild_PromptAppend(t *testing.T) {
	cmd := protoCmd("echo", []string{"--flag"})
	cmd.Prompt = "Review code"
	cmd.PromptPosition = "append"

	pod := buildAndCleanup(t, cmd)
	last := pod.LaunchArgs[len(pod.LaunchArgs)-1]
	assert.Equal(t, "Review code", last, "prompt should be last arg")
}

func TestBuild_PromptNone(t *testing.T) {
	cmd := protoCmd("echo", []string{"--flag"})
	cmd.Prompt = "Should not appear"
	cmd.PromptPosition = "none"

	pod := buildAndCleanup(t, cmd)
	for _, arg := range pod.LaunchArgs {
		assert.NotEqual(t, "Should not appear", arg)
	}
}

func TestBuild_PromptEmpty(t *testing.T) {
	cmd := protoCmd("echo", []string{"--only-flag"})
	cmd.Prompt = ""
	cmd.PromptPosition = "prepend"

	pod := buildAndCleanup(t, cmd)
	assert.Equal(t, []string{"--only-flag"}, pod.LaunchArgs)
}

func TestBuild_PromptDefaultPosition(t *testing.T) {
	cmd := protoCmd("echo", []string{"--flag"})
	cmd.Prompt = "Ignored"
	cmd.PromptPosition = "" // empty = no injection

	pod := buildAndCleanup(t, cmd)
	for _, arg := range pod.LaunchArgs {
		assert.NotEqual(t, "Ignored", arg, "empty position means no injection")
	}
}

// === Placeholder Resolution in LaunchArgs ===

func TestBuild_LaunchArgsPlaceholderResolution(t *testing.T) {
	cmd := protoCmd("echo", []string{
		"--plugin-dir", "{{sandbox_root}}/plugin",
		"--work", "{{work_dir}}/src",
	})

	pod := buildAndCleanup(t, cmd)
	assert.NotContains(t, pod.LaunchArgs[1], "{{sandbox_root}}")
	assert.Contains(t, pod.LaunchArgs[1], "/plugin")
	assert.NotContains(t, pod.LaunchArgs[3], "{{work_dir}}")
	assert.Contains(t, pod.LaunchArgs[3], "/src")
}

func TestBuild_LaunchArgsLegacyPlaceholders(t *testing.T) {
	cmd := protoCmd("echo", []string{
		"--dir", "{{.sandbox.root_path}}/data",
		"--out", "{{.sandbox.work_dir}}/output",
	})

	pod := buildAndCleanup(t, cmd)
	assert.NotContains(t, pod.LaunchArgs[1], "{{.sandbox.root_path}}")
	assert.NotContains(t, pod.LaunchArgs[3], "{{.sandbox.work_dir}}")
}

// === InteractionMode ===

func TestBuild_InteractionModePTY(t *testing.T) {
	cmd := protoCmd("echo", nil)
	cmd.InteractionMode = "pty"

	pod := buildAndCleanup(t, cmd)
	assert.Equal(t, InteractionModePTY, pod.InteractionMode)
}

func TestBuild_InteractionModeDefaultToPTY(t *testing.T) {
	cmd := protoCmd("echo", nil)
	cmd.InteractionMode = ""

	pod := buildAndCleanup(t, cmd)
	assert.Equal(t, InteractionModePTY, pod.InteractionMode)
}

// === resolvePathPlaceholders unit tests ===

func TestResolvePathPlaceholders(t *testing.T) {
	tests := []struct {
		name, input, expected string
	}{
		{"sandbox_root", "{{sandbox_root}}/plugin", "/sb/plugin"},
		{"work_dir", "{{work_dir}}/src", "/sb/ws/src"},
		{"legacy root", "{{.sandbox.root_path}}/data", "/sb/data"},
		{"legacy work", "{{.sandbox.work_dir}}/out", "/sb/ws/out"},
		{"no placeholder", "/absolute/path", "/absolute/path"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, resolvePathPlaceholders(tt.input, "/sb", "/sb/ws"))
		})
	}
}

func TestResolveStringSlice(t *testing.T) {
	input := []string{"{{sandbox_root}}/a", "plain", "{{work_dir}}/b"}
	result := resolveStringSlice(input, "/sb", "/sb/ws")
	assert.Equal(t, []string{"/sb/a", "plain", "/sb/ws/b"}, result)
}

// === Helpers ===

func protoCmd(command string, args []string) *runnerv1.CreatePodCommand {
	return &runnerv1.CreatePodCommand{
		PodKey:          "test-proto-pod",
		LaunchCommand:   command,
		LaunchArgs:      args,
		InteractionMode: "pty",
	}
}

func buildAndCleanup(t *testing.T, cmd *runnerv1.CreatePodCommand) *Pod {
	t.Helper()
	tmpDir := t.TempDir()
	r := &Runner{cfg: &config.Config{WorkspaceRoot: tmpDir}}

	builder := NewPodBuilderFromRunner(r).WithCommand(cmd)
	pod, err := builder.Build(context.Background())
	require.NoError(t, err)
	require.NotNil(t, pod)

	t.Cleanup(func() {
		if pod.IO != nil {
			pod.IO.Stop()
		}
	})
	return pod
}
