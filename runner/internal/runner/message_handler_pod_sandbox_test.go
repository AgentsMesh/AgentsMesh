package runner

import (
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/config"
)

// Tests for OnCreatePod with sandbox configurations

func TestOnCreatePodWithSandboxConfig(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:     tempDir,
		},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	// Empty sandbox config - just creates a sandbox directory
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "sandbox-pod",
		LaunchCommand: "echo",
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
		SandboxConfig: &runnerv1.SandboxConfig{},
	}

	err := handler.OnCreatePod(cmd)
	if err != nil {
		t.Logf("OnCreatePod with sandbox config: %v", err)
	}

	// Clean up
	pod, ok := store.Get("sandbox-pod")
	if ok {
		if comps := testPTYComponents(pod); comps != nil && comps.Terminal != nil {
			comps.Terminal.Stop()
		}
	}
}

func TestOnCreatePodWithFilesToCreate(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:     tempDir,
		},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "files-pod",
		LaunchCommand: "echo",
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
		FilesToCreate: []*runnerv1.FileToCreate{
			{
				Path:    "{{.sandbox.root_path}}/test.txt",
				Content: "test content",
				Mode:    0644,
			},
		},
	}

	err := handler.OnCreatePod(cmd)
	if err != nil {
		t.Logf("OnCreatePod with files to create: %v", err)
	}

	// Clean up
	pod, ok := store.Get("files-pod")
	if ok {
		if comps := testPTYComponents(pod); comps != nil && comps.Terminal != nil {
			comps.Terminal.Stop()
		}
	}
}

func TestOnCreatePodWithLocalPath(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	mockConn := client.NewMockConnection()

	runner := &Runner{
		cfg: &config.Config{
			MaxConcurrentPods: 10,
			WorkspaceRoot:     tempDir,
		},
	}

	handler := NewRunnerMessageHandler(runner, store, mockConn)

	cmd := &runnerv1.CreatePodCommand{
		PodKey:        "local-path-pod",
		LaunchCommand: "echo",
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
		SandboxConfig: &runnerv1.SandboxConfig{
			LocalPath: tempDir,
		},
	}

	err := handler.OnCreatePod(cmd)
	if err != nil {
		t.Logf("OnCreatePod with local path: %v", err)
	}

	// Clean up
	pod, ok := store.Get("local-path-pod")
	if ok {
		if comps := testPTYComponents(pod); comps != nil && comps.Terminal != nil {
			comps.Terminal.Stop()
		}
	}
}
