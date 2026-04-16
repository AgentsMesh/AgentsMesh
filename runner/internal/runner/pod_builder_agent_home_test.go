package runner

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/agentkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareAgentHome_NilCmd(t *testing.T) {
	builder := &PodBuilder{cmd: nil}
	assert.NoError(t, builder.prepareAgentHome("/sandbox", "/workspace"))
}

func TestPrepareAgentHome_NilEnvVars(t *testing.T) {
	builder := &PodBuilder{
		cmd: &runnerv1.CreatePodCommand{PodKey: "test-pod", AgentfileSource: "AGENT test\nPROMPT_POSITION prepend\n"},
	}
	assert.NoError(t, builder.prepareAgentHome("/sandbox", "/workspace"))
}

func TestPrepareAgentHome_NoMatch(t *testing.T) {
	builder := &PodBuilder{
		cmd: &runnerv1.CreatePodCommand{
			PodKey:          "test-pod",
			AgentfileSource: "AGENT test\nPROMPT_POSITION prepend\n",
			EnvVars:         map[string]string{"FOO": "bar"},
		},
	}
	assert.NoError(t, builder.prepareAgentHome("/sandbox", "/workspace"))
}

func TestPrepareAgentHome_CreatesEmptyDir(t *testing.T) {
	sandboxRoot := t.TempDir()
	agentHome := filepath.Join(sandboxRoot, "codex-home")

	builder := &PodBuilder{
		cmd: &runnerv1.CreatePodCommand{
			PodKey:          "test-pod",
			AgentfileSource: "AGENT test\nPROMPT_POSITION prepend\n",
			EnvVars:         map[string]string{"CODEX_HOME": agentHome},
		},
	}

	err := builder.prepareAgentHome(sandboxRoot, "")
	require.NoError(t, err)
	assert.True(t, dirExists(agentHome))
}

func TestPrepareAgentHome_ResolvesTemplateVars(t *testing.T) {
	sandboxRoot := t.TempDir()

	builder := &PodBuilder{
		cmd: &runnerv1.CreatePodCommand{
			PodKey:          "test-pod",
			AgentfileSource: "AGENT test\nPROMPT_POSITION prepend\n",
			EnvVars:         map[string]string{"CODEX_HOME": "{{.sandbox.root_path}}/codex-home"},
		},
	}

	err := builder.prepareAgentHome(sandboxRoot, "")
	require.NoError(t, err)
	assert.True(t, dirExists(filepath.Join(sandboxRoot, "codex-home")))
}

func TestPrepareAgentHome_NilMergeConfigReturnsEarly(t *testing.T) {
	sandboxRoot := t.TempDir()
	agentHome := filepath.Join(sandboxRoot, "test-home")

	agentkit.RegisterAgentHome(agentkit.AgentHomeSpec{
		EnvVar:      "TEST_NIL_MERGE_HOME",
		UserDirName: ".test-nil-merge",
		MergeConfig: nil,
	})

	builder := &PodBuilder{
		cmd: &runnerv1.CreatePodCommand{
			PodKey:          "test-pod",
			AgentfileSource: "AGENT test\nPROMPT_POSITION prepend\n",
			EnvVars:         map[string]string{"TEST_NIL_MERGE_HOME": agentHome},
			FilesToCreate: []*runnerv1.FileToCreate{
				{Path: filepath.Join(agentHome, "config.toml"), Content: "data", IsDirectory: false},
			},
		},
	}

	err := builder.prepareAgentHome(sandboxRoot, "")
	require.NoError(t, err)
	assert.True(t, dirExists(agentHome))
}

func TestPrepareAgentHome_MergesConfigAndRemovesFromFilesToCreate(t *testing.T) {
	sandboxRoot := t.TempDir()
	agentHome := filepath.Join(sandboxRoot, "test-merge-home")
	require.NoError(t, os.MkdirAll(agentHome, 0755))

	merged := false
	agentkit.RegisterAgentHome(agentkit.AgentHomeSpec{
		EnvVar:      "TEST_MERGE_HOME",
		UserDirName: ".test-merge",
		MergeConfig: func(configPath, content string) error {
			merged = true
			return os.WriteFile(configPath, []byte(content), 0644)
		},
	})

	configPath := filepath.Join(agentHome, "config.toml")
	builder := &PodBuilder{
		cmd: &runnerv1.CreatePodCommand{
			PodKey:          "test-pod",
			AgentfileSource: "AGENT test\nPROMPT_POSITION prepend\n",
			EnvVars:         map[string]string{"TEST_MERGE_HOME": agentHome},
			FilesToCreate: []*runnerv1.FileToCreate{
				{Path: configPath, Content: "merged-content", IsDirectory: false},
			},
		},
	}

	err := builder.prepareAgentHome(sandboxRoot, "")
	require.NoError(t, err)
	assert.True(t, merged)
	assert.Empty(t, builder.cmd.FilesToCreate)

	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, "merged-content", string(data))
}

func TestPrepareAgentHome_MergeConfigErrorContinues(t *testing.T) {
	sandboxRoot := t.TempDir()
	agentHome := filepath.Join(sandboxRoot, "test-err-home")
	require.NoError(t, os.MkdirAll(agentHome, 0755))

	agentkit.RegisterAgentHome(agentkit.AgentHomeSpec{
		EnvVar:      "TEST_MERGE_ERR_HOME",
		UserDirName: ".test-merge-err",
		MergeConfig: func(_, _ string) error { return errors.New("merge failed") },
	})

	configPath := filepath.Join(agentHome, "config.toml")
	builder := &PodBuilder{
		cmd: &runnerv1.CreatePodCommand{
			PodKey:          "test-pod",
			AgentfileSource: "AGENT test\nPROMPT_POSITION prepend\n",
			EnvVars:         map[string]string{"TEST_MERGE_ERR_HOME": agentHome},
			FilesToCreate: []*runnerv1.FileToCreate{
				{Path: configPath, Content: "data", IsDirectory: false},
			},
		},
	}

	err := builder.prepareAgentHome(sandboxRoot, "")
	require.NoError(t, err)
	assert.Len(t, builder.cmd.FilesToCreate, 1)
}

func TestPrepareAgentHome_NoHomeDir(t *testing.T) {
	t.Setenv("HOME", "")
	t.Setenv("USERPROFILE", "")

	sandboxRoot := t.TempDir()
	agentHome := filepath.Join(sandboxRoot, "nohome-agent")

	agentkit.RegisterAgentHome(agentkit.AgentHomeSpec{
		EnvVar: "TEST_NOHOME", UserDirName: ".nohome",
	})

	builder := &PodBuilder{
		cmd: &runnerv1.CreatePodCommand{
			PodKey:          "test-pod",
			AgentfileSource: "AGENT test\nPROMPT_POSITION prepend\n",
			EnvVars:         map[string]string{"TEST_NOHOME": agentHome},
		},
	}

	err := builder.prepareAgentHome(sandboxRoot, "")
	require.NoError(t, err)
	assert.True(t, dirExists(agentHome))
}

func TestPrepareAgentHome_CopiesUserConfig(t *testing.T) {
	sandboxRoot := t.TempDir()
	codexHome := filepath.Join(sandboxRoot, "codex-home")

	userHome := t.TempDir()
	userCodexDir := filepath.Join(userHome, ".codex")
	require.NoError(t, os.MkdirAll(userCodexDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(userCodexDir, "config.toml"),
		[]byte("[mcp_servers.user_server]\ncommand = \"my-server\"\n"),
		0644,
	))

	err := copyDirSelective(userCodexDir, codexHome)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "user_server")
}

func TestPrepareAgentHome_NoFilesToCreateMatchDoesNotMerge(t *testing.T) {
	sandboxRoot := t.TempDir()
	agentHome := filepath.Join(sandboxRoot, "test-nomatch-home")
	require.NoError(t, os.MkdirAll(agentHome, 0755))

	merged := false
	agentkit.RegisterAgentHome(agentkit.AgentHomeSpec{
		EnvVar:      "TEST_NOMATCH_HOME",
		UserDirName: ".test-nomatch",
		MergeConfig: func(_, _ string) error { merged = true; return nil },
	})

	builder := &PodBuilder{
		cmd: &runnerv1.CreatePodCommand{
			PodKey:          "test-pod",
			AgentfileSource: "AGENT test\nPROMPT_POSITION prepend\n",
			EnvVars:         map[string]string{"TEST_NOMATCH_HOME": agentHome},
			FilesToCreate: []*runnerv1.FileToCreate{
				{Path: "/some/other/path/config.toml", Content: "data", IsDirectory: false},
			},
		},
	}

	err := builder.prepareAgentHome(sandboxRoot, "")
	require.NoError(t, err)
	assert.False(t, merged)
	assert.Len(t, builder.cmd.FilesToCreate, 1)
}
