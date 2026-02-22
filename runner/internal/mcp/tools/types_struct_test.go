package tools

import (
	"testing"
)

// Tests for struct types

func TestBindingStruct(t *testing.T) {
	b := Binding{
		ID:            1,
		InitiatorPod:  "pod-1",
		TargetPod:     "pod-2",
		GrantedScopes: []BindingScope{ScopeTerminalRead},
		PendingScopes: []BindingScope{ScopeTerminalWrite},
		Status:        BindingStatusActive,
		CreatedAt:     "2024-01-01T00:00:00Z",
		UpdatedAt:     "2024-01-01T00:00:00Z",
	}

	if b.ID != 1 {
		t.Errorf("ID: got %v, want %v", b.ID, 1)
	}
	if b.InitiatorPod != "pod-1" {
		t.Errorf("InitiatorPod: got %v, want %v", b.InitiatorPod, "pod-1")
	}
	if b.TargetPod != "pod-2" {
		t.Errorf("TargetPod: got %v, want %v", b.TargetPod, "pod-2")
	}
	if len(b.GrantedScopes) != 1 || b.GrantedScopes[0] != ScopeTerminalRead {
		t.Errorf("GrantedScopes: got %v, want [terminal:read]", b.GrantedScopes)
	}
	if b.Status != BindingStatusActive {
		t.Errorf("Status: got %v, want %v", b.Status, BindingStatusActive)
	}
}

func TestAvailablePodStruct(t *testing.T) {
	ticketID := 123
	s := AvailablePod{
		ID:          1,
		PodKey:      "test-pod",
		CreatedByID: 1,
		Status:      PodStatusRunning,
		TicketID:    &ticketID,
		AgentType:   "claude",
		CreatedAt:   "2024-01-01T00:00:00Z",
	}

	if s.PodKey != "test-pod" {
		t.Errorf("PodKey: got %v, want %v", s.PodKey, "test-pod")
	}
	if s.Status != PodStatusRunning {
		t.Errorf("Status: got %v, want %v", s.Status, PodStatusRunning)
	}
	if s.TicketID == nil || *s.TicketID != 123 {
		t.Errorf("TicketID: got %v, want 123", s.TicketID)
	}
}

func TestTerminalOutputStruct(t *testing.T) {
	output := TerminalOutput{
		PodKey:     "test-pod",
		Output:     "test output",
		Screen:     "test screen",
		CursorX:    10,
		CursorY:    5,
		TotalLines: 100,
		HasMore:    true,
	}

	if output.PodKey != "test-pod" {
		t.Errorf("PodKey: got %v, want %v", output.PodKey, "test-pod")
	}
	if output.CursorX != 10 {
		t.Errorf("CursorX: got %v, want %v", output.CursorX, 10)
	}
	if !output.HasMore {
		t.Error("HasMore should be true")
	}
}

func TestChannelStruct(t *testing.T) {
	repositoryID := 1
	ticketID := 2

	ch := Channel{
		ID:           1,
		Name:         "test-channel",
		Description:  "Test description",
		RepositoryID: &repositoryID,
		TicketID:     &ticketID,
		Document:     "test document",
		MemberCount:  5,
		IsArchived:   false,
		CreatedAt:    "2024-01-01T00:00:00Z",
		UpdatedAt:    "2024-01-01T00:00:00Z",
	}

	if ch.Name != "test-channel" {
		t.Errorf("Name: got %v, want %v", ch.Name, "test-channel")
	}
	if ch.MemberCount != 5 {
		t.Errorf("MemberCount: got %v, want %v", ch.MemberCount, 5)
	}
}

func TestChannelMessageStruct(t *testing.T) {
	userID := 1
	replyTo := 10

	msg := ChannelMessage{
		ID:           1,
		ChannelID:    100,
		SenderPod:    "test-pod",
		SenderUserID: &userID,
		Content:      "Hello world",
		MessageType:  ChannelMessageTypeText,
		Mentions:     []string{"pod-1", "pod-2"},
		ReplyTo:      &replyTo,
		CreatedAt:    "2024-01-01T00:00:00Z",
	}

	if msg.Content != "Hello world" {
		t.Errorf("Content: got %v, want %v", msg.Content, "Hello world")
	}
	if len(msg.Mentions) != 2 {
		t.Errorf("Mentions: got %v mentions, want 2", len(msg.Mentions))
	}
}

func TestTicketStruct(t *testing.T) {
	parentID := 100
	estimate := 5

	ticket := Ticket{
		ID:             1,
		Identifier:     "AM-123",
		Title:          "Test Ticket",
		Content:        "Test content",
		Type:           TicketTypeTask,
		Status:         TicketStatusTodo,
		Priority:       TicketPriorityMedium,
		ProductID:      1,
		ProductName:    "Test Product",
		ReporterID:     1,
		ReporterName:   "Test User",
		ParentTicketID: &parentID,
		Estimate:       &estimate,
		CreatedAt:      "2024-01-01T00:00:00Z",
		UpdatedAt:      "2024-01-01T00:00:00Z",
	}

	if ticket.Identifier != "AM-123" {
		t.Errorf("Identifier: got %v, want %v", ticket.Identifier, "AM-123")
	}
	if ticket.Type != TicketTypeTask {
		t.Errorf("Type: got %v, want %v", ticket.Type, TicketTypeTask)
	}
	if ticket.ParentTicketID == nil || *ticket.ParentTicketID != 100 {
		t.Errorf("ParentTicketID: got %v, want 100", ticket.ParentTicketID)
	}
}
