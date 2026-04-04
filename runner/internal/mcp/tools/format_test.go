package tools

import (
	"strings"
	"testing"
)

// --- PodSnapshot ---

func TestPodSnapshot_FormatText(t *testing.T) {
	tests := []struct {
		name     string
		input    *PodSnapshot
		contains []string
	}{
		{
			name: "basic output",
			input: &PodSnapshot{
				PodKey:     "pod-abc",
				Output:     "$ ls\nfile.go",
				TotalLines: 150,
				HasMore:    true,
			},
			contains: []string{"Pod: pod-abc", "Lines: 150", "Has More: true", "$ ls\nfile.go"},
		},
		{
			name: "with screen",
			input: &PodSnapshot{
				PodKey:     "pod-x",
				Output:     "output",
				Screen:     "screen content",
				TotalLines: 10,
				HasMore:    false,
			},
			contains: []string{"Pod: pod-x", "Has More: false", "--- Screen ---", "screen content"},
		},
		{
			name: "empty output",
			input: &PodSnapshot{
				PodKey:     "pod-empty",
				TotalLines: 0,
				HasMore:    false,
			},
			contains: []string{"Pod: pod-empty", "Lines: 0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.FormatText()
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q, got:\n%s", s, result)
				}
			}
		})
	}
}

// --- Binding ---

func TestBinding_FormatText(t *testing.T) {
	tests := []struct {
		name     string
		input    *Binding
		contains []string
	}{
		{
			name: "full binding",
			input: &Binding{
				ID:            1,
				InitiatorPod:  "pod-a",
				TargetPod:     "pod-b",
				GrantedScopes: []BindingScope{ScopePodRead, ScopePodWrite},
				Status:        BindingStatusActive,
				CreatedAt:     "2026-02-20T10:00:00Z",
				UpdatedAt:     "2026-02-20T11:00:00Z",
			},
			contains: []string{"Binding: #1", "Initiator: pod-a", "Target: pod-b", "Status: active", "pod:read, pod:write", "Created: 2026-02-20T10:00:00Z"},
		},
		{
			name: "with pending scopes",
			input: &Binding{
				ID:            2,
				InitiatorPod:  "pod-c",
				TargetPod:     "pod-d",
				PendingScopes: []BindingScope{ScopePodRead},
				Status:        BindingStatusPending,
			},
			contains: []string{"Binding: #2", "Status: pending", "Pending Scopes: pod:read"},
		},
		{
			name: "no scopes",
			input: &Binding{
				ID:           3,
				InitiatorPod: "pod-e",
				TargetPod:    "pod-f",
				Status:       BindingStatusRejected,
			},
			contains: []string{"Binding: #3", "Status: rejected"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.input.FormatText()
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("expected result to contain %q, got:\n%s", s, result)
				}
			}
		})
	}
}

// --- Channel ---

func TestChannel_FormatText(t *testing.T) {
	t.Run("full channel", func(t *testing.T) {
		ch := &Channel{
			ID:           1,
			Name:         "dev-chat",
			Description:  "Development discussion",
			MemberCount:  5,
			IsArchived:   false,
			CreatedByPod: "pod-leader",
			CreatedAt:    "2026-02-19T08:00:00Z",
			UpdatedAt:    "2026-02-20T10:00:00Z",
		}
		result := ch.FormatText()
		for _, s := range []string{"Channel: dev-chat (ID: 1)", "Description: Development discussion", "Members: 5", "Created By: pod-leader"} {
			if !strings.Contains(result, s) {
				t.Errorf("expected %q in:\n%s", s, result)
			}
		}
	})

	t.Run("minimal channel", func(t *testing.T) {
		ch := &Channel{ID: 2, Name: "empty", MemberCount: 0}
		result := ch.FormatText()
		if !strings.Contains(result, "Channel: empty (ID: 2)") {
			t.Errorf("unexpected result:\n%s", result)
		}
		if strings.Contains(result, "Description:") {
			t.Errorf("should not have Description line for empty description:\n%s", result)
		}
	})
}

// --- ChannelMessage ---

func TestChannelMessage_FormatText(t *testing.T) {
	replyTo := 5
	msg := &ChannelMessage{
		ID:          10,
		ChannelID:   1,
		SenderPod:   "pod-alpha",
		Content:     "Hello world",
		MessageType: "text",
		Mentions:    []string{"pod-beta", "pod-gamma"},
		ReplyTo:     &replyTo,
		CreatedAt:   "2026-02-20T10:30:00Z",
	}
	result := msg.FormatText()
	for _, s := range []string{"Message #10", "Channel: 1", "pod-alpha", "text", "Reply To: #5", "pod-beta, pod-gamma", "Hello world"} {
		if !strings.Contains(result, s) {
			t.Errorf("expected %q in:\n%s", s, result)
		}
	}
}

// --- Ticket ---

func TestTicket_FormatText(t *testing.T) {
	t.Run("full ticket", func(t *testing.T) {
		tk := &Ticket{
			Slug:         "AM-123",
			Title:        "Fix authentication bug",
			Status:       TicketStatusInProgress,
			Priority:     TicketPriorityHigh,
			ReporterName: "john",
			CreatedAt:    "2026-02-19T08:00:00Z",
			UpdatedAt:    "2026-02-20T15:00:00Z",
		}
		result := tk.FormatText()
		for _, s := range []string{"AM-123 - Fix authentication bug", "Status: in_progress", "Priority: high", "Reporter: john"} {
			if !strings.Contains(result, s) {
				t.Errorf("expected %q in:\n%s", s, result)
			}
		}
	})

	t.Run("with parent ticket", func(t *testing.T) {
		tk := &Ticket{
			Slug:             "AM-125",
			Title:            "Sub-task",
			Status:           TicketStatusTodo,
			Priority:         TicketPriorityMedium,
			ParentTicketSlug: "AM-100",
		}
		result := tk.FormatText()
		if !strings.Contains(result, "Parent: AM-100") {
			t.Errorf("expected parent ticket slug in:\n%s", result)
		}
	})

	t.Run("content with line range metadata", func(t *testing.T) {
		tk := &Ticket{
			Slug:              "AM-124",
			Title:             "Test",
			Content:           "Line one\nLine two\nLine three",
			Status:            TicketStatusTodo,
			Priority:          TicketPriorityMedium,
			ContentTotalLines: 50,
			ContentOffset:     0,
			ContentLimit:      3,
		}
		result := tk.FormatText()
		if !strings.Contains(result, "Content (lines 1-3 of 50):") {
			t.Errorf("expected line range header in content:\n%s", result)
		}
		if !strings.Contains(result, "Line one") {
			t.Errorf("expected content body:\n%s", result)
		}
	})

	t.Run("content with offset", func(t *testing.T) {
		tk := &Ticket{
			Slug:              "AM-126",
			Title:             "Paginated",
			Content:           "Line 201\nLine 202",
			Status:            TicketStatusTodo,
			Priority:          TicketPriorityMedium,
			ContentTotalLines: 500,
			ContentOffset:     200,
			ContentLimit:      2,
		}
		result := tk.FormatText()
		if !strings.Contains(result, "Content (lines 201-202 of 500):") {
			t.Errorf("expected offset line range in content:\n%s", result)
		}
	})

	t.Run("content without metadata falls back to simple format", func(t *testing.T) {
		tk := &Ticket{
			Slug:     "AM-127",
			Title:    "Simple",
			Content:  "Just plain text",
			Status:   TicketStatusTodo,
			Priority: TicketPriorityMedium,
		}
		result := tk.FormatText()
		if !strings.Contains(result, "Content:\nJust plain text") {
			t.Errorf("expected simple content format:\n%s", result)
		}
		if strings.Contains(result, "lines") {
			t.Errorf("should not have line range when no metadata:\n%s", result)
		}
	})
}
