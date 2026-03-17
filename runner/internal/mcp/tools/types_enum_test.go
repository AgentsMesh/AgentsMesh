package tools

import (
	"testing"
)

// Tests for enum/constant types

func TestBindingScope(t *testing.T) {
	tests := []struct {
		name  string
		scope BindingScope
		want  string
	}{
		{"pod read", ScopePodRead, "pod:read"},
		{"pod write", ScopePodWrite, "pod:write"},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.status) != tt.want {
				t.Errorf("got %v, want %v", tt.status, tt.want)
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
