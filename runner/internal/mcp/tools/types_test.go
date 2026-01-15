package tools

import (
	"testing"
)

func TestBindingScope(t *testing.T) {
	tests := []struct {
		name  string
		scope BindingScope
		want  string
	}{
		{"terminal read", ScopeTerminalRead, "terminal:read"},
		{"terminal write", ScopeTerminalWrite, "terminal:write"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.scope) != tt.want {
				t.Errorf("got %v, want %v", tt.scope, tt.want)
			}
		})
	}
}

func TestBindingStatus(t *testing.T) {
	tests := []struct {
		name   string
		status BindingStatus
		want   string
	}{
		{"pending", BindingStatusPending, "pending"},
		{"active", BindingStatusActive, "active"},
		{"rejected", BindingStatusRejected, "rejected"},
		{"inactive", BindingStatusInactive, "inactive"},
		{"expired", BindingStatusExpired, "expired"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("got %v, want %v", tt.status, tt.want)
			}
		})
	}
}

func TestPodStatus(t *testing.T) {
	tests := []struct {
		name   string
		status PodStatus
		want   string
	}{
		{"initializing", PodStatusInitializing, "initializing"},
		{"running", PodStatusRunning, "running"},
		{"disconnected", PodStatusDisconnected, "disconnected"},
		{"completed", PodStatusCompleted, "completed"},
		{"error", PodStatusError, "error"},
		{"orphaned", PodStatusOrphaned, "orphaned"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("got %v, want %v", tt.status, tt.want)
			}
		})
	}
}

func TestTicketStatus(t *testing.T) {
	tests := []struct {
		name   string
		status TicketStatus
		want   string
	}{
		{"backlog", TicketStatusBacklog, "backlog"},
		{"todo", TicketStatusTodo, "todo"},
		{"in_progress", TicketStatusInProgress, "in_progress"},
		{"in_review", TicketStatusInReview, "in_review"},
		{"done", TicketStatusDone, "done"},
		{"canceled", TicketStatusCanceled, "canceled"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("got %v, want %v", tt.status, tt.want)
			}
		})
	}
}

func TestTicketType(t *testing.T) {
	tests := []struct {
		name       string
		ticketType TicketType
		want       string
	}{
		{"task", TicketTypeTask, "task"},
		{"bug", TicketTypeBug, "bug"},
		{"feature", TicketTypeFeature, "feature"},
		{"improvement", TicketTypeImprovement, "improvement"},
		{"epic", TicketTypeEpic, "epic"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.ticketType) != tt.want {
				t.Errorf("got %v, want %v", tt.ticketType, tt.want)
			}
		})
	}
}

func TestTicketPriority(t *testing.T) {
	tests := []struct {
		name     string
		priority TicketPriority
		want     string
	}{
		{"urgent", TicketPriorityUrgent, "urgent"},
		{"high", TicketPriorityHigh, "high"},
		{"medium", TicketPriorityMedium, "medium"},
		{"low", TicketPriorityLow, "low"},
		{"none", TicketPriorityNone, "none"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.priority) != tt.want {
				t.Errorf("got %v, want %v", tt.priority, tt.want)
			}
		})
	}
}

func TestChannelMessageType(t *testing.T) {
	tests := []struct {
		name    string
		msgType ChannelMessageType
		want    string
	}{
		{"text", ChannelMessageTypeText, "text"},
		{"system", ChannelMessageTypeSystem, "system"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.msgType) != tt.want {
				t.Errorf("got %v, want %v", tt.msgType, tt.want)
			}
		})
	}
}

func TestBindingStruct(t *testing.T) {
	b := Binding{
		ID:               1,
		InitiatorPod: "pod-1",
		TargetPod:    "pod-2",
		GrantedScopes:    []BindingScope{ScopeTerminalRead},
		PendingScopes:    []BindingScope{ScopeTerminalWrite},
		Status:           BindingStatusActive,
		CreatedAt:        "2024-01-01T00:00:00Z",
		UpdatedAt:        "2024-01-01T00:00:00Z",
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
		PodKey: "test-pod",
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
	projectID := 1
	ticketID := 2

	ch := Channel{
		ID:          1,
		Name:        "test-channel",
		Description: "Test description",
		ProjectID:   &projectID,
		TicketID:    &ticketID,
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
		ID:            1,
		ChannelID:     100,
		SenderPod: "test-pod",
		SenderUserID:  &userID,
		Content:       "Hello world",
		MessageType:   ChannelMessageTypeText,
		Mentions:      []string{"pod-1", "pod-2"},
		ReplyTo:       &replyTo,
		CreatedAt:     "2024-01-01T00:00:00Z",
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
		Description:    "Test description",
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

func TestPodCreateRequest(t *testing.T) {
	ticketID := 123

	req := PodCreateRequest{
		RunnerID:      1,
		TicketID:      &ticketID,
		InitialPrompt: "Hello",
		Model:         "claude-sonnet",
	}

	if req.RunnerID != 1 {
		t.Errorf("RunnerID: got %v, want %v", req.RunnerID, 1)
	}
	if req.TicketID == nil || *req.TicketID != 123 {
		t.Errorf("TicketID: got %v, want 123", req.TicketID)
	}
}

func TestPodCreateResponse(t *testing.T) {
	resp := PodCreateResponse{
		PodKey:  "new-pod",
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
