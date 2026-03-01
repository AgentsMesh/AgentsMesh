package runner

import (
	"testing"
	"time"

	runnerv1 "github.com/AgentsMesh/AgentsMesh/proto/gen/go/runner/v1"
)

func TestSandboxQueryService_CompleteQuery(t *testing.T) {
	svc := NewSandboxQueryService(nil)
	defer svc.Stop()

	requestID := "complete-test"
	ch := svc.RegisterQuery(requestID)

	// Complete the query
	event := &runnerv1.SandboxesStatusEvent{
		RequestId: requestID,
		Sandboxes: []*runnerv1.SandboxStatus{
			{
				PodKey:                "pod-1",
				Exists:                true,
				CanResume:             true,
				SandboxPath:           "/path/to/sandbox",
				RepositoryUrl:         "https://github.com/test/repo",
				BranchName:            "main",
				CurrentCommit:         "abc12345",
				SizeBytes:             1024,
				LastModified:          time.Now().Unix(),
				HasUncommittedChanges: true,
			},
		},
	}

	svc.CompleteQuery(requestID, 42, event)

	// Should receive result on channel
	select {
	case result := <-ch:
		if result == nil {
			t.Fatal("Expected non-nil result")
		}
		if result.RequestID != requestID {
			t.Errorf("RequestID = %s, want %s", result.RequestID, requestID)
		}
		if result.RunnerID != 42 {
			t.Errorf("RunnerID = %d, want 42", result.RunnerID)
		}
		if len(result.Sandboxes) != 1 {
			t.Fatalf("Sandboxes len = %d, want 1", len(result.Sandboxes))
		}
		sb := result.Sandboxes[0]
		if sb.PodKey != "pod-1" {
			t.Errorf("PodKey = %s, want pod-1", sb.PodKey)
		}
		if !sb.Exists {
			t.Error("Exists should be true")
		}
		if !sb.CanResume {
			t.Error("CanResume should be true")
		}
		if sb.SandboxPath != "/path/to/sandbox" {
			t.Errorf("SandboxPath = %s, want /path/to/sandbox", sb.SandboxPath)
		}
		if sb.RepositoryURL != "https://github.com/test/repo" {
			t.Errorf("RepositoryURL = %s, want https://github.com/test/repo", sb.RepositoryURL)
		}
		if sb.BranchName != "main" {
			t.Errorf("BranchName = %s, want main", sb.BranchName)
		}
		if sb.CurrentCommit != "abc12345" {
			t.Errorf("CurrentCommit = %s, want abc12345", sb.CurrentCommit)
		}
		if !sb.HasUncommittedChanges {
			t.Error("HasUncommittedChanges should be true")
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for result")
	}

	// Query should be removed from pending
	_, ok := svc.pendingQueries.Load(requestID)
	if ok {
		t.Error("Query should be removed after completion")
	}
}

func TestSandboxQueryService_CompleteQuery_NotFound(t *testing.T) {
	svc := NewSandboxQueryService(nil)
	defer svc.Stop()

	// Complete a query that was never registered - should not panic
	event := &runnerv1.SandboxesStatusEvent{
		RequestId: "nonexistent",
		Sandboxes: []*runnerv1.SandboxStatus{},
	}

	// Should not panic
	svc.CompleteQuery("nonexistent", 1, event)
}

func TestSandboxQueryService_CompleteQuery_ChannelFull(t *testing.T) {
	svc := NewSandboxQueryService(nil)
	defer svc.Stop()

	requestID := "full-channel"
	ch := svc.RegisterQuery(requestID)

	// Fill the channel first
	ch <- &SandboxQueryResult{RequestID: "dummy"}

	// Now complete query - channel is full, should not panic
	event := &runnerv1.SandboxesStatusEvent{
		RequestId: requestID,
		Sandboxes: []*runnerv1.SandboxStatus{},
	}

	// Should not panic even with full channel
	svc.CompleteQuery(requestID, 1, event)
}

func TestSandboxQueryService_MultipleSandboxes(t *testing.T) {
	svc := NewSandboxQueryService(nil)
	defer svc.Stop()

	requestID := "multi-sandbox"
	ch := svc.RegisterQuery(requestID)

	// Complete with multiple sandboxes
	event := &runnerv1.SandboxesStatusEvent{
		RequestId: requestID,
		Sandboxes: []*runnerv1.SandboxStatus{
			{PodKey: "pod-1", Exists: true, CanResume: true},
			{PodKey: "pod-2", Exists: true, CanResume: false, Error: "session file missing"},
			{PodKey: "pod-3", Exists: false},
		},
	}

	svc.CompleteQuery(requestID, 99, event)

	select {
	case result := <-ch:
		if len(result.Sandboxes) != 3 {
			t.Errorf("Sandboxes len = %d, want 3", len(result.Sandboxes))
		}
		// Check each sandbox
		for i, sb := range result.Sandboxes {
			expectedKey := "pod-" + string(rune('1'+i))
			if sb.PodKey != expectedKey {
				t.Logf("Sandbox %d: PodKey = %s", i, sb.PodKey)
			}
		}
		// Check specific fields
		if result.Sandboxes[1].Error != "session file missing" {
			t.Errorf("Sandbox 2 error = %s, want 'session file missing'", result.Sandboxes[1].Error)
		}
		if result.Sandboxes[2].Exists {
			t.Error("Sandbox 3 should not exist")
		}
	case <-time.After(time.Second):
		t.Fatal("Timeout waiting for result")
	}
}
