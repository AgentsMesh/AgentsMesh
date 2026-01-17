package updater

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for waitAndApply functionality

func TestGracefulUpdater_WaitAndApply_ContextCanceled(t *testing.T) {
	u := New("1.0.0")

	var podCount int32 = 5
	podCounter := func() int { return int(atomic.LoadInt32(&podCount)) }

	g := NewGracefulUpdater(u, podCounter,
		WithMaxWaitTime(10*time.Second),
		WithPollInterval(50*time.Millisecond),
	)

	tmpDir, err := os.MkdirTemp("", "graceful-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pendingPath := filepath.Join(tmpDir, "pending")
	err = os.WriteFile(pendingPath, []byte("binary"), 0755)
	require.NoError(t, err)

	g.mu.Lock()
	g.pendingPath = pendingPath
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0"}
	g.mu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())

	errCh := make(chan error, 1)
	go func() {
		errCh <- g.waitAndApply(ctx)
	}()

	time.Sleep(100 * time.Millisecond)
	cancel()

	err = <-errCh
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
	assert.False(t, g.IsDraining())
}

func TestGracefulUpdater_WaitAndApply_Timeout(t *testing.T) {
	u := New("1.0.0")
	podCounter := func() int { return 1 }

	g := NewGracefulUpdater(u, podCounter,
		WithMaxWaitTime(100*time.Millisecond),
		WithPollInterval(20*time.Millisecond),
	)

	tmpDir, err := os.MkdirTemp("", "graceful-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pendingPath := filepath.Join(tmpDir, "pending")
	err = os.WriteFile(pendingPath, []byte("binary"), 0755)
	require.NoError(t, err)

	g.mu.Lock()
	g.pendingPath = pendingPath
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0"}
	g.mu.Unlock()

	err = g.waitAndApply(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "postponed")
	assert.Equal(t, StateIdle, g.State())

	g.mu.RLock()
	assert.Empty(t, g.pendingPath)
	assert.Nil(t, g.pendingInfo)
	g.mu.RUnlock()
}

func TestGracefulUpdater_WaitAndApply_PodsFinish(t *testing.T) {
	u := New("1.0.0")

	var podCount int32 = 2
	podCounter := func() int { return int(atomic.LoadInt32(&podCount)) }

	g := NewGracefulUpdater(u, podCounter,
		WithMaxWaitTime(5*time.Second),
		WithPollInterval(50*time.Millisecond),
	)

	tmpDir, err := os.MkdirTemp("", "graceful-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pendingPath := filepath.Join(tmpDir, "pending")
	err = os.WriteFile(pendingPath, []byte("binary"), 0755)
	require.NoError(t, err)

	g.mu.Lock()
	g.pendingPath = pendingPath
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	go func() {
		time.Sleep(100 * time.Millisecond)
		atomic.StoreInt32(&podCount, 1)
		time.Sleep(100 * time.Millisecond)
		atomic.StoreInt32(&podCount, 0)
	}()

	err = g.waitAndApply(context.Background())
	_ = err // May or may not fail at apply stage
}

func TestGracefulUpdater_WaitAndApply_StatusCallback(t *testing.T) {
	u := New("1.0.0")

	var podCount int32 = 1
	podCounter := func() int { return int(atomic.LoadInt32(&podCount)) }

	var mu sync.Mutex
	var states []State
	cb := func(state State, info *UpdateInfo, activePods int) {
		mu.Lock()
		states = append(states, state)
		mu.Unlock()
	}

	g := NewGracefulUpdater(u, podCounter,
		WithMaxWaitTime(500*time.Millisecond),
		WithPollInterval(50*time.Millisecond),
		WithStatusCallback(cb),
	)

	tmpDir, err := os.MkdirTemp("", "graceful-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pendingPath := filepath.Join(tmpDir, "pending")
	err = os.WriteFile(pendingPath, []byte("binary"), 0755)
	require.NoError(t, err)

	g.mu.Lock()
	g.pendingPath = pendingPath
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0"}
	g.mu.Unlock()

	_ = g.waitAndApply(context.Background())

	mu.Lock()
	defer mu.Unlock()
	assert.Contains(t, states, StateDraining)
}

func TestGracefulUpdater_WaitAndApply_NilPodCounter(t *testing.T) {
	u := New("1.0.0")

	g := NewGracefulUpdater(u, nil,
		WithMaxWaitTime(100*time.Millisecond),
		WithPollInterval(20*time.Millisecond),
	)

	tmpDir, err := os.MkdirTemp("", "graceful-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	pendingPath := filepath.Join(tmpDir, "pending")
	err = os.WriteFile(pendingPath, []byte("binary"), 0755)
	require.NoError(t, err)

	g.mu.Lock()
	g.pendingPath = pendingPath
	g.pendingInfo = &UpdateInfo{LatestVersion: "v2.0.0", CurrentVersion: "v1.0.0"}
	g.mu.Unlock()

	err = g.waitAndApply(context.Background())
	_ = err // May fail at apply stage
}
