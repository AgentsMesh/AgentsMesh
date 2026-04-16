package claude

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestControlRequestTracker_NextID_Unique(t *testing.T) {
	tr := newControlRequestTracker()
	ids := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		id := tr.nextID()
		assert.False(t, ids[id], "duplicate ID: %s", id)
		ids[id] = true
		assert.Contains(t, id, "req_")
	}
}

func TestControlRequestTracker_TrackAndResolve(t *testing.T) {
	tr := newControlRequestTracker()
	ch := tr.track("req_1")

	resolved := tr.resolve("req_1", map[string]any{"ok": true}, nil)
	assert.True(t, resolved)

	result := <-ch
	assert.Equal(t, map[string]any{"ok": true}, result.response)
	assert.NoError(t, result.err)
}

func TestControlRequestTracker_Resolve_UnknownID(t *testing.T) {
	tr := newControlRequestTracker()
	resolved := tr.resolve("nonexistent", nil, nil)
	assert.False(t, resolved)
}

func TestControlRequestTracker_Resolve_WithError(t *testing.T) {
	tr := newControlRequestTracker()
	ch := tr.track("req_err")

	testErr := errors.New("test error")
	tr.resolve("req_err", nil, testErr)

	result := <-ch
	assert.Nil(t, result.response)
	assert.EqualError(t, result.err, "test error")
}

func TestControlRequestTracker_SendAndWait_Success(t *testing.T) {
	tr := newControlRequestTracker()

	var writtenMsg any
	writeFunc := func(v any) error {
		writtenMsg = v
		// Simulate async response: resolve after write
		go func() {
			time.Sleep(10 * time.Millisecond)
			msg := writtenMsg.(map[string]any)
			reqID := msg["request_id"].(string)
			tr.resolve(reqID, map[string]any{"status": "ok"}, nil)
		}()
		return nil
	}

	resp, err := tr.sendAndWait("interrupt", nil, writeFunc)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"status": "ok"}, resp)

	// Verify written message structure
	msg := writtenMsg.(map[string]any)
	assert.Equal(t, "control_request", msg["type"])
	req := msg["request"].(map[string]any)
	assert.Equal(t, "interrupt", req["subtype"])
}

func TestControlRequestTracker_SendAndWait_WithPayload(t *testing.T) {
	tr := newControlRequestTracker()

	var writtenMsg any
	writeFunc := func(v any) error {
		writtenMsg = v
		go func() {
			msg := writtenMsg.(map[string]any)
			tr.resolve(msg["request_id"].(string), nil, nil)
		}()
		return nil
	}

	_, err := tr.sendAndWait("set_permission_mode", map[string]any{"mode": "default"}, writeFunc)
	require.NoError(t, err)

	msg := writtenMsg.(map[string]any)
	req := msg["request"].(map[string]any)
	assert.Equal(t, "set_permission_mode", req["subtype"])
	assert.Equal(t, "default", req["mode"])
}

func TestControlRequestTracker_SendAndWait_Timeout(t *testing.T) {
	// Override timeout for fast test. Use a short-lived tracker.
	tr := newControlRequestTracker()

	writeFunc := func(v any) error {
		// Don't resolve — simulates timeout
		return nil
	}

	start := time.Now()
	_, err := tr.sendAndWait("interrupt", nil, writeFunc)
	elapsed := time.Since(start)

	assert.ErrorIs(t, err, acp.ErrControlTimeout)
	assert.GreaterOrEqual(t, elapsed, 25*time.Second, "should wait close to timeout")
}

func TestControlRequestTracker_SendAndWait_WriteError(t *testing.T) {
	tr := newControlRequestTracker()

	writeFunc := func(v any) error {
		return errors.New("write failed")
	}

	_, err := tr.sendAndWait("interrupt", nil, writeFunc)
	assert.ErrorContains(t, err, "write failed")

	// Verify pending map is cleaned up
	tr.mu.Lock()
	assert.Empty(t, tr.pending)
	tr.mu.Unlock()
}

func TestControlRequestTracker_Concurrent(t *testing.T) {
	tr := newControlRequestTracker()
	const n = 10

	writeFunc := func(v any) error {
		go func() {
			msg := v.(map[string]any)
			reqID := msg["request_id"].(string)
			time.Sleep(5 * time.Millisecond)
			tr.resolve(reqID, map[string]any{"id": reqID}, nil)
		}()
		return nil
	}

	var wg sync.WaitGroup
	results := make([]string, n)
	errs := make([]error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			resp, err := tr.sendAndWait("test", nil, writeFunc)
			errs[idx] = err
			if resp != nil {
				results[idx] = resp["id"].(string)
			}
		}(i)
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		assert.NoError(t, errs[i], "request %d failed", i)
		assert.NotEmpty(t, results[i], "request %d got empty response", i)
	}
	// All results should be unique (different request IDs)
	unique := make(map[string]bool)
	for _, r := range results {
		unique[r] = true
	}
	assert.Len(t, unique, n, "all concurrent requests should get unique responses")
}
