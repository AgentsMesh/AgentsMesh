//go:build integration

package runner

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/acp"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunner_PodCreationFailure_InvalidAgentfile_Integration verifies that
// an AgentFile with syntax errors causes Build to fail cleanly.
func TestRunner_PodCreationFailure_InvalidAgentfile_Integration(t *testing.T) {
	store := NewInMemoryPodStore()
	runner := &Runner{cfg: &config.Config{WorkspaceRoot: t.TempDir()}}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:          "bad-agentfile-pod",
		AgentfileSource: "AGENT echo\nINVALID_KEYWORD ???\n",
	}

	pod, err := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).WithPtySize(80, 24).
		Build(context.Background())

	require.Error(t, err, "build with invalid AgentFile should fail")
	assert.Nil(t, pod, "pod should be nil on error")
	assert.Equal(t, 0, store.Count(), "store should remain empty")
}

// TestRunner_PodCreationFailure_MissingAgent_Integration verifies that
// an AgentFile referencing a non-existent executable fails cleanly.
func TestRunner_PodCreationFailure_MissingAgent_Integration(t *testing.T) {
	store := NewInMemoryPodStore()
	runner := &Runner{cfg: &config.Config{WorkspaceRoot: t.TempDir()}}

	cmd := &runnerv1.CreatePodCommand{
		PodKey:          "missing-agent-pod",
		AgentfileSource: "AGENT /nonexistent/binary/xyzzy\nMODE pty\nPROMPT_POSITION prepend\n",
	}

	pod, err := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).WithPtySize(80, 24).
		Build(context.Background())

	require.Error(t, err, "build with missing executable should fail")
	assert.Nil(t, pod, "pod should be nil on error")
	assert.Equal(t, 0, store.Count(), "store should remain empty")
}

// TestRunner_ACPRelayEventForwarding_Integration wires an ACP pod with the
// mock agent and verifies content chunk and state change callbacks fire.
func TestRunner_ACPRelayEventForwarding_Integration(t *testing.T) {
	// Build ACP pod shell.
	agentfile := "AGENT echo\nMODE acp\nPROMPT_POSITION prepend\n"
	runner := &Runner{cfg: &config.Config{WorkspaceRoot: t.TempDir()}}
	cmd := &runnerv1.CreatePodCommand{
		PodKey:          "acp-relay-test",
		AgentfileSource: agentfile,
	}
	pod, err := NewPodBuilderFromRunner(runner).
		WithCommand(cmd).WithPtySize(80, 24).
		Build(context.Background())
	require.NoError(t, err)

	// Wire ACPClient with tracking callbacks.
	var mu sync.Mutex
	var chunks []acp.ContentChunk
	var states []string

	acpClient := acp.NewClient(acp.ClientConfig{
		Command: mockAgentCmd(),
		Args:    mockAgentArgs(),
		Env:     mockAgentEnv(),
		WorkDir: t.TempDir(),
		Logger:  slog.Default(),
		Callbacks: acp.EventCallbacks{
			OnContentChunk: func(_ string, chunk acp.ContentChunk) {
				mu.Lock()
				chunks = append(chunks, chunk)
				mu.Unlock()
			},
			OnStateChange: func(newState string) {
				mu.Lock()
				states = append(states, newState)
				mu.Unlock()
			},
		},
	})
	pod.IO = NewACPPodIO(acpClient, pod.PodKey)

	require.NoError(t, acpClient.Start())
	defer acpClient.Stop()

	require.NoError(t, acpClient.NewSession(nil))
	require.NoError(t, acpClient.SendPrompt("hello"))

	// Wait for content chunk and idle state.
	deadline := time.After(5 * time.Second)
	for {
		mu.Lock()
		gotChunks := len(chunks)
		mu.Unlock()
		if gotChunks > 0 && acpClient.State() == acp.StateIdle {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for content chunks and idle state")
		case <-time.After(50 * time.Millisecond):
		}
	}

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, "Hello from runner mock", chunks[0].Text)
	assert.Contains(t, states, acp.StateProcessing, "should have seen processing state")
}

// TestRunner_ConcurrentTermination_Integration creates 3 pods, terminates
// all concurrently, and verifies the store is empty with no panics.
func TestRunner_ConcurrentTermination_Integration(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()

	for i := 0; i < 3; i++ {
		podKey := "term-" + string(rune('a'+i))
		r := &Runner{cfg: &config.Config{WorkspaceRoot: tempDir}}
		cmd := &runnerv1.CreatePodCommand{
			PodKey:          podKey,
			AgentfileSource: "AGENT cat\nMODE pty\nPROMPT_POSITION prepend\n",
		}
		pod, err := NewPodBuilderFromRunner(r).
			WithCommand(cmd).WithPtySize(80, 24).
			Build(context.Background())
		require.NoError(t, err)
		store.Put(podKey, pod)
	}
	require.Equal(t, 3, store.Count())

	// Terminate all concurrently.
	var wg sync.WaitGroup
	var panicCount atomic.Int32
	for _, pod := range store.All() {
		wg.Add(1)
		go func(p *Pod) {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					panicCount.Add(1)
				}
			}()
			if comps := testPTYComponents(p); comps != nil && comps.Terminal != nil {
				comps.Terminal.Stop()
			}
			store.Delete(p.PodKey)
		}(pod)
	}
	wg.Wait()

	assert.Equal(t, int32(0), panicCount.Load(), "no panics during concurrent termination")
	assert.Equal(t, 0, store.Count(), "store should be empty after termination")
}
