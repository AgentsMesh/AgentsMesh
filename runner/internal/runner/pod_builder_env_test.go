package runner

import (
	"os"
	"strings"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/clipboard"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

// Tests for environment variable merging

func TestPodBuilderMergeEnvVars(t *testing.T) {
	runner := &Runner{
		cfg: &config.Config{
			AgentEnvVars: map[string]string{
				"CONFIG_VAR": "config_value",
				"SHARED_VAR": "config_shared",
			},
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-pod",
		LaunchCommand: "echo",
		EnvVars: map[string]string{
			"BUILDER_VAR": "builder_value",
			"SHARED_VAR":  "builder_shared",
		},
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	result := builder.mergeEnvVars("")

	if result["CONFIG_VAR"] != "config_value" {
		t.Errorf("CONFIG_VAR: got %v, want config_value", result["CONFIG_VAR"])
	}

	if result["BUILDER_VAR"] != "builder_value" {
		t.Errorf("BUILDER_VAR: got %v, want builder_value", result["BUILDER_VAR"])
	}

	if result["SHARED_VAR"] != "builder_shared" {
		t.Errorf("SHARED_VAR: got %v, want builder_shared (command should override config)", result["SHARED_VAR"])
	}
}

func TestPodBuilderMergeEnvVarsNilConfig(t *testing.T) {
	runner := &Runner{
		cfg: nil,
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-pod",
		LaunchCommand: "echo",
		EnvVars: map[string]string{
			"BUILDER_VAR": "builder_value",
		},
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	result := builder.mergeEnvVars("")

	if result["BUILDER_VAR"] != "builder_value" {
		t.Errorf("BUILDER_VAR: got %v, want builder_value", result["BUILDER_VAR"])
	}
}

func TestPodBuilderWithAllOptions(t *testing.T) {
	runner := &Runner{
		cfg: &config.Config{
			WorkspaceRoot: "/workspace",
			AgentEnvVars: map[string]string{
				"CONFIG_VAR": "config_value",
			},
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "pod-key",
		LaunchCommand: "claude",
		LaunchArgs:    []string{"--headless"},
		EnvVars: map[string]string{
			"ENV1": "value1",
			"ENV2": "value2",
		},
		SandboxConfig: &runnerv1.SandboxConfig{
			RepositoryUrl:  "https://github.com/test/repo.git",
			SourceBranch:   "main",
			CredentialType: "runner_local",
		},
		FilesToCreate: []*runnerv1.FileToCreate{
			{Path: "{{.sandbox.root_path}}/test.txt", Content: "test"},
		},
	}

	builder := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).
		WithTerminalSize(120, 40) // (cols, rows)

	if builder.cmd.PodKey != "pod-key" {
		t.Errorf("podKey = %v, want pod-key", builder.cmd.PodKey)
	}
	if len(builder.cmd.LaunchArgs) != 1 || builder.cmd.LaunchArgs[0] != "--headless" {
		t.Errorf("launchArgs = %v, want [--headless]", builder.cmd.LaunchArgs)
	}
	if builder.cmd.EnvVars["ENV1"] != "value1" {
		t.Errorf("envVars[ENV1] = %v, want value1", builder.cmd.EnvVars["ENV1"])
	}
	if builder.cmd.EnvVars["ENV2"] != "value2" {
		t.Errorf("envVars[ENV2] = %v, want value2", builder.cmd.EnvVars["ENV2"])
	}
	if builder.rows != 40 || builder.cols != 120 {
		t.Errorf("terminal size = %dx%d, want 40x120", builder.rows, builder.cols)
	}
	if builder.cmd.SandboxConfig == nil {
		t.Error("sandboxConfig should not be nil")
	}
	if len(builder.cmd.FilesToCreate) != 1 {
		t.Error("filesToCreate not set correctly")
	}
}

// Tests for clipboard PATH merging in mergeEnvVars

func TestPodBuilderMergeEnvVars_ClipboardPrependsPATH(t *testing.T) {
	// Setup: shim backend with real shim bin dir
	dir := t.TempDir()
	shimBackend := &clipboard.ShimBackend{}
	if err := shimBackend.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	builder := NewPodBuilder(PodBuilderDeps{
		Config:    &config.Config{},
		Clipboard: shimBackend,
	}).WithCommand(&runnerv1.CreatePodCommand{
		PodKey:        "test",
		LaunchCommand: "echo",
	})

	result := builder.mergeEnvVars(dir)

	// PATH should be shimBinDir:systemPATH
	path := result["PATH"]
	sep := string(os.PathListSeparator)
	expectedPrefix := clipboard.ShimBinDir(dir) + sep
	if !strings.HasPrefix(path, expectedPrefix) {
		t.Errorf("PATH should start with %q, got %q", expectedPrefix, path)
	}
	if !strings.HasSuffix(path, os.Getenv("PATH")) {
		t.Errorf("PATH should end with system PATH")
	}
}

func TestPodBuilderMergeEnvVars_ClipboardPrependsToCustomPATH(t *testing.T) {
	// When command sets a custom PATH, clipboard should prepend to that
	dir := t.TempDir()
	shimBackend := &clipboard.ShimBackend{}
	if err := shimBackend.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	sep := string(os.PathListSeparator)
	customPath := "/custom/bin" + sep + "/other/bin"
	builder := NewPodBuilder(PodBuilderDeps{
		Config:    &config.Config{},
		Clipboard: shimBackend,
	}).WithCommand(&runnerv1.CreatePodCommand{
		PodKey:        "test",
		LaunchCommand: "echo",
		EnvVars: map[string]string{
			"PATH": customPath,
		},
	})

	result := builder.mergeEnvVars(dir)

	expected := clipboard.ShimBinDir(dir) + sep + customPath
	if result["PATH"] != expected {
		t.Errorf("PATH = %q, want %q", result["PATH"], expected)
	}
}

func TestPodBuilderMergeEnvVars_ClipboardNil(t *testing.T) {
	// When clipboard is nil, PATH should not be modified
	builder := NewPodBuilder(PodBuilderDeps{
		Config:    &config.Config{},
		Clipboard: nil,
	}).WithCommand(&runnerv1.CreatePodCommand{
		PodKey:        "test",
		LaunchCommand: "echo",
	})

	result := builder.mergeEnvVars("/some/sandbox")
	if _, ok := result["PATH"]; ok {
		t.Error("PATH should not be set when clipboard is nil")
	}
}

func TestPodBuilderMergeEnvVars_ClipboardEmptySandbox(t *testing.T) {
	// When sandboxRoot is empty, clipboard overrides should not be applied
	shimBackend := &clipboard.ShimBackend{}

	builder := NewPodBuilder(PodBuilderDeps{
		Config:    &config.Config{},
		Clipboard: shimBackend,
	}).WithCommand(&runnerv1.CreatePodCommand{
		PodKey:        "test",
		LaunchCommand: "echo",
	})

	result := builder.mergeEnvVars("")
	if _, ok := result["PATH"]; ok {
		t.Error("PATH should not be set when sandboxRoot is empty")
	}
}

func TestPodBuilderMergeEnvVars_NativeClipboardNoPathOverride(t *testing.T) {
	// NativeBackend returns nil for EnvOverrides — no PATH modification
	builder := NewPodBuilder(PodBuilderDeps{
		Config:    &config.Config{},
		Clipboard: &clipboard.NativeBackend{},
	}).WithCommand(&runnerv1.CreatePodCommand{
		PodKey:        "test",
		LaunchCommand: "echo",
	})

	result := builder.mergeEnvVars("/some/sandbox")
	if _, ok := result["PATH"]; ok {
		t.Error("PATH should not be set when NativeBackend returns nil overrides")
	}
}

// mockClipboardBackend is a test helper implementing clipboard.Backend
// that returns configurable env overrides.
type mockClipboardBackend struct {
	name      string
	overrides map[string]string
}

func (m *mockClipboardBackend) Name() string                                         { return m.name }
func (m *mockClipboardBackend) Setup(string) error                                   { return nil }
func (m *mockClipboardBackend) WriteImage(string, string, []byte) error              { return nil }
func (m *mockClipboardBackend) EnvOverrides(string) map[string]string                { return m.overrides }

func TestPodBuilderMergeEnvVars_ClipboardNonPathOverride(t *testing.T) {
	// A clipboard backend that returns a non-PATH env var
	mock := &mockClipboardBackend{
		name: "test",
		overrides: map[string]string{
			"CLIPBOARD_TOOL": "custom-tool",
		},
	}

	builder := NewPodBuilder(PodBuilderDeps{
		Config:    &config.Config{},
		Clipboard: mock,
	}).WithCommand(&runnerv1.CreatePodCommand{
		PodKey:        "test",
		LaunchCommand: "echo",
	})

	result := builder.mergeEnvVars("/some/sandbox")

	if result["CLIPBOARD_TOOL"] != "custom-tool" {
		t.Errorf("CLIPBOARD_TOOL = %q, want %q", result["CLIPBOARD_TOOL"], "custom-tool")
	}
}

func TestPodBuilderMergeEnvVars_ClipboardMixedOverrides(t *testing.T) {
	// A clipboard backend that returns both PATH and non-PATH vars
	dir := t.TempDir()
	shimBackend := &clipboard.ShimBackend{}
	if err := shimBackend.Setup(dir); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	// Use a mock that returns PATH + another key
	mock := &mockClipboardBackend{
		name: "test",
		overrides: map[string]string{
			"PATH":      clipboard.ShimBinDir(dir),
			"EXTRA_VAR": "extra",
		},
	}

	builder := NewPodBuilder(PodBuilderDeps{
		Config:    &config.Config{},
		Clipboard: mock,
	}).WithCommand(&runnerv1.CreatePodCommand{
		PodKey:        "test",
		LaunchCommand: "echo",
	})

	result := builder.mergeEnvVars(dir)

	// PATH should be prepended
	expectedPathPrefix := clipboard.ShimBinDir(dir) + string(os.PathListSeparator)
	if !strings.HasPrefix(result["PATH"], expectedPathPrefix) {
		t.Errorf("PATH should start with %q, got %q", expectedPathPrefix, result["PATH"])
	}
	// Non-PATH var should be set directly
	if result["EXTRA_VAR"] != "extra" {
		t.Errorf("EXTRA_VAR = %q, want %q", result["EXTRA_VAR"], "extra")
	}
}

// Benchmarks

func BenchmarkNewPodBuilder(b *testing.B) {
	runner := &Runner{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewPodBuilderFromRunner(runner)
	}
}

func BenchmarkPodBuilderFluentAPI(b *testing.B) {
	runner := &Runner{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd := &runnerv1.CreatePodCommand{
			PodKey:        "pod-1",
			LaunchCommand: "claude",
			LaunchArgs:    []string{"--headless"},
			EnvVars:       map[string]string{"KEY": "VALUE"},
		}
		NewPodBuilderFromRunner(runner).
			WithCommand(cmd).
			WithTerminalSize(120, 40) // (cols, rows)
	}
}

func BenchmarkPodBuilderMergeEnvVars(b *testing.B) {
	runner := &Runner{
		cfg: &config.Config{
			AgentEnvVars: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
		},
	}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "test-pod",
		LaunchCommand: "echo",
		EnvVars: map[string]string{
			"POD_VAR1": "pod_value1",
		},
	}

	builder := NewPodBuilderFromRunner(runner).WithCommand(cmd)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		builder.mergeEnvVars("")
	}
}
