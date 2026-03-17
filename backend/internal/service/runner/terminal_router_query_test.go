package runner

import (
	"context"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// TestTerminalRouter_ObserveTerminal_Success tests the full async query lifecycle:
// register pod → send observe command → async complete → return result
func TestTerminalRouter_ObserveTerminal_Success(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	defer tr.Stop()

	// Set up a mock sender that captures the requestID and completes the query async
	sender := &observeCaptureSender{
		completeFn: func(requestID string) {
			// Simulate async response from runner
			go func() {
				time.Sleep(10 * time.Millisecond)
				tr.completeQuery(requestID, 100, &runnerv1.ObserveTerminalResult{
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

	// Call ObserveTerminal
	result, err := tr.ObserveTerminal(context.Background(), "pod-1", 100, true)
	if err != nil {
		t.Fatalf("ObserveTerminal error: %v", err)
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

// TestTerminalRouter_ObserveTerminal_PodNotRegistered tests that ObserveTerminal returns
// ErrRunnerNotConnected when the pod is not registered.
func TestTerminalRouter_ObserveTerminal_PodNotRegistered(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	defer tr.Stop()

	tr.SetCommandSender(&MockCommandSender{})

	_, err := tr.ObserveTerminal(context.Background(), "nonexistent-pod", 100, false)
	if err != ErrRunnerNotConnected {
		t.Errorf("err = %v, want ErrRunnerNotConnected", err)
	}
}

// TestTerminalRouter_ObserveTerminal_SendError tests that ObserveTerminal propagates
// errors from the command sender.
func TestTerminalRouter_ObserveTerminal_SendError(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	defer tr.Stop()

	// Use NoOp sender which returns ErrCommandSenderNotSet
	tr.RegisterPod("pod-1", 100)

	_, err := tr.ObserveTerminal(context.Background(), "pod-1", 100, false)
	if err != ErrCommandSenderNotSet {
		t.Errorf("err = %v, want ErrCommandSenderNotSet", err)
	}
}

// TestTerminalRouter_ObserveTerminal_ContextCanceled tests that ObserveTerminal
// respects context cancellation.
func TestTerminalRouter_ObserveTerminal_ContextCanceled(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
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

	_, err := tr.ObserveTerminal(ctx, "pod-1", 100, false)
	if err != context.Canceled {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

// TestTerminalRouter_ObserveTerminal_Timeout tests that ObserveTerminal returns
// a timeout result when the runner doesn't respond.
func TestTerminalRouter_ObserveTerminal_Timeout(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())
	defer tr.Stop()

	// Sender that succeeds but never completes the query
	sender := &observeCaptureSender{
		completeFn: func(requestID string) {
			// Don't complete - let it timeout
		},
	}
	tr.SetCommandSender(sender)
	tr.RegisterPod("pod-1", 100)

	// Use a short context deadline so we don't wait for the full TerminalQueryTimeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := tr.ObserveTerminal(ctx, "pod-1", 100, false)
	if err != context.DeadlineExceeded {
		t.Errorf("err = %v, want context.DeadlineExceeded", err)
	}
}

// TestTerminalRouter_Stop tests that Stop() shuts down the cleanup goroutine.
func TestTerminalRouter_Stop(t *testing.T) {
	cm := NewRunnerConnectionManager(newTestLogger())
	tr := NewTerminalRouter(cm, newTestLogger())

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

// observeCaptureSender is a mock RunnerCommandSender that captures ObserveTerminal calls
// and calls a user-provided function with the requestID.
type observeCaptureSender struct {
	MockCommandSender
	completeFn func(requestID string)
}

func (s *observeCaptureSender) SendObserveTerminal(ctx context.Context, runnerID int64, requestID, podKey string, lines int32, includeScreen bool) error {
	if s.completeFn != nil {
		s.completeFn(requestID)
	}
	return nil
}
