package runner

import (
	"context"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// TestPodRouter_ObservePod_Success tests the full async query lifecycle:
// register pod → send observe command → async complete → return result
func TestPodRouter_ObservePod_Success(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())
	defer tr.Stop()

	// Set up a mock sender that captures the requestID and completes the query async
	sender := &observeCaptureSender{
		completeFn: func(requestID string) {
			// Simulate async response from runner
			go func() {
				time.Sleep(10 * time.Millisecond)
				tr.completeQuery(requestID, 100, &runnerv1.ObservePodResult{
					RequestId:  requestID,
					Output:     "$ hello world",
					Screen:     "screen snapshot",
					CursorX:    5,
					CursorY:    1,
					TotalLines: 10,
					HasMore:    false,
				})
			}()
		},
	}
	tr.SetCommandSender(sender)

	// Register pod
	tr.RegisterPod("pod-1", 100)

	// Call ObservePod
	result, err := tr.ObservePod(context.Background(), "pod-1", 100, true)
	if err != nil {
		t.Fatalf("ObservePod error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.RunnerID != 100 {
		t.Errorf("RunnerID = %d, want 100", result.RunnerID)
	}
	if result.Output != "$ hello world" {
		t.Errorf("Output = %q, want %q", result.Output, "$ hello world")
	}
	if result.Screen != "screen snapshot" {
		t.Errorf("Screen = %q, want %q", result.Screen, "screen snapshot")
	}
	if result.CursorX != 5 {
		t.Errorf("CursorX = %d, want 5", result.CursorX)
	}
	if result.TotalLines != 10 {
		t.Errorf("TotalLines = %d, want 10", result.TotalLines)
	}
}

// TestPodRouter_ObservePod_PodNotRegistered tests that ObservePod returns
// ErrRunnerNotConnected when the pod is not registered.
func TestPodRouter_ObservePod_PodNotRegistered(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())
	defer tr.Stop()

	tr.SetCommandSender(&MockCommandSender{})

	_, err := tr.ObservePod(context.Background(), "nonexistent-pod", 100, false)
	if err != ErrRunnerNotConnected {
		t.Errorf("err = %v, want ErrRunnerNotConnected", err)
	}
}

// TestPodRouter_ObservePod_SendError tests that ObservePod propagates
// errors from the command sender.
func TestPodRouter_ObservePod_SendError(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())
	defer tr.Stop()

	// Use NoOp sender which returns ErrCommandSenderNotSet
	tr.RegisterPod("pod-1", 100)

	_, err := tr.ObservePod(context.Background(), "pod-1", 100, false)
	if err != ErrCommandSenderNotSet {
		t.Errorf("err = %v, want ErrCommandSenderNotSet", err)
	}
}

// TestPodRouter_ObservePod_ContextCanceled tests that ObservePod
// respects context cancellation.
func TestPodRouter_ObservePod_ContextCanceled(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())
	defer tr.Stop()

	// Sender that succeeds but never completes the query
	sender := &observeCaptureSender{
		completeFn: func(requestID string) {
			// Don't complete - let context cancel
		},
	}
	tr.SetCommandSender(sender)
	tr.RegisterPod("pod-1", 100)

	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately after send
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := tr.ObservePod(ctx, "pod-1", 100, false)
	if err != context.Canceled {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

// TestPodRouter_ObservePod_Timeout tests that ObservePod returns
// a timeout result when the runner doesn't respond.
func TestPodRouter_ObservePod_Timeout(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())
	defer tr.Stop()

	// Sender that succeeds but never completes the query
	sender := &observeCaptureSender{
		completeFn: func(requestID string) {
			// Don't complete - let it timeout
		},
	}
	tr.SetCommandSender(sender)
	tr.RegisterPod("pod-1", 100)

	// Use a short context deadline so we don't wait for the full ObservePodTimeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := tr.ObservePod(ctx, "pod-1", 100, false)
	if err != context.DeadlineExceeded {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
}

// TestPodRouter_Stop tests that Stop() shuts down the cleanup goroutine.
func TestPodRouter_Stop(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewPodRouter(cm, newTestLogger())

	// Stop should not panic and should be callable
	tr.Stop()

	// Verify done channel is closed
	select {
	case <-tr.done:
		// OK - channel is closed
	default:
		t.Error("done channel should be closed after Stop()")
	}
}

// observeCaptureSender is a mock RunnerCommandSender that captures ObservePod calls
// and calls a user-provided function with the requestID.
type observeCaptureSender struct {
	MockCommandSender
	completeFn func(requestID string)
}

func (s *observeCaptureSender) SendObservePod(ctx context.Context, runnerID int64, requestID, podKey string, lines int32, includeScreen bool) error {
	if s.completeFn != nil {
		s.completeFn(requestID)
	}
	return nil
}
