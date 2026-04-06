package tools

import (
	"testing"
)

// Tests for Pod-related types and methods

func TestPodCreateRequest(t *testing.T) {
	ticketSlug := "AM-123"

	req := PodCreateRequest{
		RunnerID:   1,
		TicketSlug: &ticketSlug,
	}

	if req.RunnerID != 1 {
		t.Errorf("RunnerID: got %v, want %v", req.RunnerID, 1)
	}
	if req.TicketSlug == nil || *req.TicketSlug != "AM-123" {
		t.Errorf("TicketSlug: got %v, want AM-123", req.TicketSlug)
	}
}

func TestPodCreateRequestWithAllFields(t *testing.T) {
	ticketSlug := "AM-123"
	repositoryID := int64(789)
	repositoryURL := "https://github.com/example/repo.git"
	branchName := "feature/new-feature"
	credentialProfileID := int64(111)

	req := PodCreateRequest{
		RunnerID:            1,
		AgentSlug:           "claude-code",
		TicketSlug:          &ticketSlug,
		RepositoryID:        &repositoryID,
		RepositoryURL:       &repositoryURL,
		BranchName:          &branchName,
		CredentialProfileID: &credentialProfileID,
	}

	if req.RunnerID != 1 {
		t.Errorf("RunnerID: got %v, want %v", req.RunnerID, 1)
	}
	if req.AgentSlug != "claude-code" {
		t.Errorf("AgentSlug: got %v, want claude-code", req.AgentSlug)
	}
	if req.RepositoryID == nil || *req.RepositoryID != 789 {
		t.Errorf("RepositoryID: got %v, want 789", req.RepositoryID)
	}
	if req.RepositoryURL == nil || *req.RepositoryURL != "https://github.com/example/repo.git" {
		t.Errorf("RepositoryURL: got %v, want https://github.com/example/repo.git", req.RepositoryURL)
	}
	if req.BranchName == nil || *req.BranchName != "feature/new-feature" {
		t.Errorf("BranchName: got %v, want feature/new-feature", req.BranchName)
	}
	if req.CredentialProfileID == nil || *req.CredentialProfileID != 111 {
		t.Errorf("CredentialProfileID: got %v, want 111", req.CredentialProfileID)
	}
}

func TestPodCreateResponse(t *testing.T) {
	resp := PodCreateResponse{
		PodKey:      "new-pod",
		Status:      "created",
		TerminalURL: "ws://localhost:8080/terminal",
	}

	if resp.PodKey != "new-pod" {
		t.Errorf("PodKey: got %v, want %v", resp.PodKey, "new-pod")
	}
	if resp.Status != "created" {
		t.Errorf("Status: got %v, want %v", resp.Status, "created")
	}
}

func TestAgentFieldUnmarshalJSONString(t *testing.T) {
	var field AgentField
	err := field.UnmarshalJSON([]byte(`"claude-code"`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(field) != "claude-code" {
		t.Errorf("AgentField: got %v, want claude-code", field)
	}
}

func TestAgentFieldUnmarshalJSONObject(t *testing.T) {
	var field AgentField
	err := field.UnmarshalJSON([]byte(`{"id": 1, "slug": "aider", "name": "Aider"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(field) != "aider" {
		t.Errorf("AgentField: got %v, want aider", field)
	}
}

func TestAgentFieldUnmarshalJSONInvalid(t *testing.T) {
	var field AgentField
	// Invalid JSON should not cause error, just ignore
	err := field.UnmarshalJSON([]byte(`invalid json`))
	if err != nil {
		t.Errorf("expected no error for invalid JSON, got: %v", err)
	}
}

func TestAvailablePodGetUsername(t *testing.T) {
	// Test with CreatedBy set
	pod := AvailablePod{
		PodKey: "test-pod",
		CreatedBy: &PodCreator{
			ID:       1,
			Username: "testuser",
			Name:     "Test User",
		},
	}
	if pod.GetUsername() != "testuser" {
		t.Errorf("GetUsername: got %v, want testuser", pod.GetUsername())
	}

	// Test with CreatedBy nil
	pod2 := AvailablePod{
		PodKey: "test-pod-2",
	}
	if pod2.GetUsername() != "" {
		t.Errorf("GetUsername: got %v, want empty string", pod2.GetUsername())
	}
}

func TestAvailablePodGetTicketTitle(t *testing.T) {
	// Test with Ticket set
	ticketID := 123
	pod := AvailablePod{
		PodKey:   "test-pod",
		TicketID: &ticketID,
		Ticket: &PodTicket{
			ID:    123,
			Slug:  "AM-123",
			Title: "Test Ticket Title",
		},
	}
	if pod.GetTicketTitle() != "Test Ticket Title" {
		t.Errorf("GetTicketTitle: got %v, want Test Ticket Title", pod.GetTicketTitle())
	}

	// Test with Ticket nil
	pod2 := AvailablePod{
		PodKey: "test-pod-2",
	}
	if pod2.GetTicketTitle() != "" {
		t.Errorf("GetTicketTitle: got %v, want empty string", pod2.GetTicketTitle())
	}
}
