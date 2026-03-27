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
		GrantedScopes: []BindingScope{ScopePodRead},
		PendingScopes: []BindingScope{ScopePodWrite},
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
	if len(b.GrantedScopes) != 1 || b.GrantedScopes[0] != ScopePodRead {
		t.Errorf("GrantedScopes: got %v, want [pod:read]", b.GrantedScopes)
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
		Agent:       "claude",
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

func TestPodSnapshotStruct(t *testing.T) {
	output := PodSnapshot{
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
	ch := Channel{
		ID:          1,
		Name:        "test-channel",
		Description: "Test description",
		TicketSlug:  "AM-456",
		Document:    "test document",
		MemberCount: 5,
		IsArchived:  false,
		CreatedAt:   "2024-01-01T00:00:00Z",
		UpdatedAt:   "2024-01-01T00:00:00Z",
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
	estimate := 5

	ticket := Ticket{
		Slug:             "AM-123",
		Title:            "Test Ticket",
		Content:          "Test content",
		Status:           TicketStatusTodo,
		Priority:         TicketPriorityMedium,
		ProductName:      "Test Product",
		ReporterName:     "Test User",
		ParentTicketSlug: "AM-100",
		Estimate:         &estimate,
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-01T00:00:00Z",
	}

	if ticket.Slug != "AM-123" {
		t.Errorf("Slug: got %v, want %v", ticket.Slug, "AM-123")
	}
	if ticket.ParentTicketSlug != "AM-100" {
		t.Errorf("ParentTicketSlug: got %v, want AM-100", ticket.ParentTicketSlug)
	}
}
