package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// mockSandboxQuerySender implements SandboxQuerySender for testing.
type mockSandboxQuerySender struct {
	sendFn      func(runnerID int64, requestID string, podKeys []string) error
	isConnected bool
}

func (m *mockSandboxQuerySender) SendQuerySandboxes(runnerID int64, requestID string, podKeys []string) error {
	if m.sendFn != nil {
		return m.sendFn(runnerID, requestID, podKeys)
	}
	return nil
}

func (m *mockSandboxQuerySender) IsConnected(runnerID int64) bool {
	return m.isConnected
}

func TestSandboxQueryService_QuerySandboxes_Success(t *testing.T) {
	svc := NewSandboxQueryService(nil)
	defer svc.Stop()

	ctx := context.Background()
	podKeys := []string{"pod-1", "pod-2"}

	// Mock sender that simulates async response
	sender := &mockSandboxQuerySender{
		isConnected: true,
		sendFn: func(runnerID int64, requestID string, podKeys []string) error {
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
		},
	}
	svc.SetSender(sender)

	result, err := svc.QuerySandboxes(ctx, 123, podKeys)
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

	sender := &mockSandboxQuerySender{
		isConnected: true,
		sendFn: func(runnerID int64, requestID string, podKeys []string) error {
			return expectedErr
		},
	}
	svc.SetSender(sender)

	_, err := svc.QuerySandboxes(ctx, 1, []string{"pod-1"})
	if err != expectedErr {
		t.Errorf("Error = %v, want %v", err, expectedErr)
	}
}

func TestSandboxQueryService_QuerySandboxes_ContextCanceled(t *testing.T) {
	svc := NewSandboxQueryService(nil)
	defer svc.Stop()

	ctx, cancel := context.WithCancel(context.Background())

	sender := &mockSandboxQuerySender{
		isConnected: true,
		sendFn: func(runnerID int64, requestID string, podKeys []string) error {
			// Cancel context before response arrives
			cancel()
			return nil
		},
	}
	svc.SetSender(sender)

	_, err := svc.QuerySandboxes(ctx, 1, []string{"pod-1"})
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

	sender := &mockSandboxQuerySender{
		isConnected: true,
		sendFn: func(runnerID int64, requestID string, podKeys []string) error {
			// Don't respond - let it timeout
			return nil
		},
	}
	svc.SetSender(sender)

	_, err := svc.QuerySandboxes(ctx, 1, []string{"pod-1"})
	if err != context.DeadlineExceeded {
		t.Errorf("Error = %v, want context.DeadlineExceeded", err)
	}
}

func TestSandboxQueryService_QuerySandboxes_NoSender(t *testing.T) {
	svc := NewSandboxQueryService(nil)
	defer svc.Stop()

	// Don't set sender — should return ErrCommandSenderNotSet
	_, err := svc.QuerySandboxes(context.Background(), 1, []string{"pod-1"})
	if err != ErrCommandSenderNotSet {
		t.Errorf("Error = %v, want ErrCommandSenderNotSet", err)
	}
}

func TestSandboxQueryService_IsConnected(t *testing.T) {
	svc := NewSandboxQueryService(nil)
	defer svc.Stop()

	// No sender → always false
	if svc.IsConnected(1) {
		t.Error("Expected false when sender is nil")
	}

	svc.SetSender(&mockSandboxQuerySender{isConnected: true})
	if !svc.IsConnected(1) {
		t.Error("Expected true when sender reports connected")
	}

	svc.SetSender(&mockSandboxQuerySender{isConnected: false})
	if svc.IsConnected(1) {
		t.Error("Expected false when sender reports disconnected")
	}
}
