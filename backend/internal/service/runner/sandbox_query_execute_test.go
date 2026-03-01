package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	runnerv1 "github.com/AgentsMesh/AgentsMesh/proto/gen/go/runner/v1"
)

func TestSandboxQueryService_QuerySandboxes_Success(t *testing.T) {
	svc := NewSandboxQueryService(nil)
	defer svc.Stop()

	ctx := context.Background()
	podKeys := []string{"pod-1", "pod-2"}

	// Mock send function that simulates async response
	sendFn := func(runnerID int64, requestID string, podKeys []string) error {
		// Simulate async response from runner
		go func() {
			time.Sleep(10 * time.Millisecond)
			event := &runnerv1.SandboxesStatusEvent{
				RequestId: requestID,
				Sandboxes: []*runnerv1.SandboxStatus{
					{PodKey: "pod-1", Exists: true},
					{PodKey: "pod-2", Exists: false},
				},
			}
			svc.CompleteQuery(requestID, runnerID, event)
		}()
		return nil
	}

	result, err := svc.QuerySandboxes(ctx, 123, podKeys, sendFn)
	if err != nil {
		t.Fatalf("QuerySandboxes error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}
	if result.RunnerID != 123 {
		t.Errorf("RunnerID = %d, want 123", result.RunnerID)
	}
	if len(result.Sandboxes) != 2 {
		t.Errorf("Sandboxes len = %d, want 2", len(result.Sandboxes))
	}
}

func TestSandboxQueryService_QuerySandboxes_SendError(t *testing.T) {
	svc := NewSandboxQueryService(nil)
	defer svc.Stop()

	ctx := context.Background()
	expectedErr := errors.New("send failed")

	sendFn := func(runnerID int64, requestID string, podKeys []string) error {
		return expectedErr
	}

	_, err := svc.QuerySandboxes(ctx, 1, []string{"pod-1"}, sendFn)
	if err != expectedErr {
		t.Errorf("Error = %v, want %v", err, expectedErr)
	}
}

func TestSandboxQueryService_QuerySandboxes_ContextCanceled(t *testing.T) {
	svc := NewSandboxQueryService(nil)
	defer svc.Stop()

	ctx, cancel := context.WithCancel(context.Background())

	sendFn := func(runnerID int64, requestID string, podKeys []string) error {
		// Cancel context before response arrives
		cancel()
		return nil
	}

	_, err := svc.QuerySandboxes(ctx, 1, []string{"pod-1"}, sendFn)
	if err != context.Canceled {
		t.Errorf("Error = %v, want context.Canceled", err)
	}
}

func TestSandboxQueryService_QuerySandboxes_Timeout(t *testing.T) {
	// Create service with short timeout for testing
	svc := NewSandboxQueryService(nil)
	defer svc.Stop()

	// Override timeout for this test by using a context with shorter deadline
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	sendFn := func(runnerID int64, requestID string, podKeys []string) error {
		// Don't respond - let it timeout
		return nil
	}

	_, err := svc.QuerySandboxes(ctx, 1, []string{"pod-1"}, sendFn)
	if err != context.DeadlineExceeded {
		t.Errorf("Error = %v, want context.DeadlineExceeded", err)
	}
}
