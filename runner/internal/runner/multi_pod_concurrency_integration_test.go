//go:build integration

package runner

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func buildMultiPod(t *testing.T, tempDir, podKey string) *Pod {
	t.Helper()
	runner := &Runner{cfg: &config.Config{WorkspaceRoot: tempDir}}
	cmd := &runnerv1.CreatePodCommand{
		PodKey:        podKey,
		AgentfileSource: "AGENT echo\nPROMPT_POSITION prepend\n",
	}
	pod, err := NewPodBuilderFromRunner(runner).WithCommand(cmd).WithPtySize(80, 24).Build(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		if comps := testPTYComponents(pod); comps != nil && comps.Terminal != nil {
			comps.Terminal.Stop()
		}
	})
	return pod
}

// TestMultiPod_ConcurrentCreate_Integration creates N pods concurrently
// and verifies all succeed with unique sandboxes.
func TestMultiPod_ConcurrentCreate_Integration(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	const n = 5

	var wg sync.WaitGroup
	errCh := make(chan error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			podKey := "concurrent-" + string(rune('a'+idx))
			pod := buildMultiPod(t, tempDir, podKey)
			store.Put(podKey, pod)
		}(i)
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("unexpected error: %v", err)
	}
	assert.Equal(t, n, store.Count())

	// Verify unique sandbox paths
	seen := map[string]bool{}
	for _, pod := range store.All() {
		assert.NotEmpty(t, pod.SandboxPath)
		assert.False(t, seen[pod.SandboxPath], "duplicate sandbox: %s", pod.SandboxPath)
		seen[pod.SandboxPath] = true
	}
}

// TestMultiPod_CapacityLimit_Integration verifies CanAcceptPod gate.
func TestMultiPod_CapacityLimit_Integration(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()
	runner := &Runner{
		cfg:      &config.Config{WorkspaceRoot: tempDir, MaxConcurrentPods: 2},
		podStore: store,
	}

	pod1 := buildMultiPod(t, tempDir, "cap-pod-1")
	store.Put("cap-pod-1", pod1)
	assert.True(t, runner.CanAcceptPod())

	pod2 := buildMultiPod(t, tempDir, "cap-pod-2")
	store.Put("cap-pod-2", pod2)
	assert.False(t, runner.CanAcceptPod(), "should reject at capacity")

	store.Delete("cap-pod-1")
	assert.True(t, runner.CanAcceptPod(), "should accept after termination")
}

// TestMultiPod_ConcurrentCreateTerminate_Integration mixes creation and
// termination concurrently and verifies final count is correct.
func TestMultiPod_ConcurrentCreateTerminate_Integration(t *testing.T) {
	tempDir := t.TempDir()
	store := NewInMemoryPodStore()

	// Pre-create 3 pods
	for i := 0; i < 3; i++ {
		podKey := "pre-" + string(rune('a'+i))
		pod := buildMultiPod(t, tempDir, podKey)
		store.Put(podKey, pod)
	}
	require.Equal(t, 3, store.Count())

	var wg sync.WaitGroup
	// Concurrently create 2 more
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			podKey := "new-" + string(rune('a'+idx))
			pod := buildMultiPod(t, tempDir, podKey)
			store.Put(podKey, pod)
		}(i)
	}
	// Concurrently terminate 2 existing
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			podKey := "pre-" + string(rune('a'+idx))
			if p, ok := store.Get(podKey); ok {
				if comps := testPTYComponents(p); comps != nil && comps.Terminal != nil {
					comps.Terminal.Stop()
				}
				store.Delete(podKey)
			}
		}(i)
	}
	wg.Wait()

	// 3 initial + 2 created - 2 terminated = 3
	assert.Equal(t, 3, store.Count())
}

// TestMultiPod_IsolatedSandboxes_Integration verifies each pod gets
// a unique sandbox directory and files are isolated.
func TestMultiPod_IsolatedSandboxes_Integration(t *testing.T) {
	tempDir := t.TempDir()
	pods := make([]*Pod, 3)
	for i := 0; i < 3; i++ {
		pods[i] = buildMultiPod(t, tempDir, "iso-"+string(rune('a'+i)))
	}

	// Write a unique file in each sandbox
	for i, pod := range pods {
		require.DirExists(t, pod.SandboxPath)
		f := filepath.Join(pod.SandboxPath, "marker.txt")
		require.NoError(t, os.WriteFile(f, []byte{byte(i)}, 0644))
	}

	// Verify isolation — each sandbox sees only its own marker
	for i, pod := range pods {
		f := filepath.Join(pod.SandboxPath, "marker.txt")
		data, err := os.ReadFile(f)
		require.NoError(t, err)
		assert.Equal(t, []byte{byte(i)}, data)

		// Other sandboxes' markers are not visible here
		for j, other := range pods {
			if i == j {
				continue
			}
			otherMarker := filepath.Join(pod.SandboxPath, other.PodKey+"-marker.txt")
			_, err := os.Stat(otherMarker)
			assert.True(t, os.IsNotExist(err), "sandbox isolation violated")
		}
	}
}
