package tools

import (
	"strings"
	"testing"
)

// --- AvailablePodList ---

func TestAvailablePodList_FormatText(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		result := AvailablePodList(nil).FormatText()
		if result != "No available pods." {
			t.Errorf("expected empty message, got: %s", result)
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		result := AvailablePodList([]AvailablePod{}).FormatText()
		if result != "No available pods." {
			t.Errorf("expected empty message, got: %s", result)
		}
	})

	t.Run("single pod", func(t *testing.T) {
		pods := AvailablePodList{
			{
				PodKey:    "pod-abc",
				Status:    PodStatusRunning,
				Agent:     AgentField("claude-code"),
				CreatedBy: &PodCreator{Username: "alice"},
				Ticket:    &PodTicket{Title: "Fix bug"},
			},
		}
		result := pods.FormatText()
		if !strings.Contains(result, "| pod-abc |") {
			t.Errorf("expected pod key in table:\n%s", result)
		}
		if !strings.Contains(result, "| running |") {
			t.Errorf("expected status in table:\n%s", result)
		}
		if !strings.Contains(result, "| alice |") {
			t.Errorf("expected username in table:\n%s", result)
		}
	})

	t.Run("nil creator and ticket", func(t *testing.T) {
		pods := AvailablePodList{
			{PodKey: "pod-x", Status: PodStatusRunning, Agent: AgentField("aider")},
		}
		result := pods.FormatText()
		if !strings.Contains(result, "| pod-x |") {
			t.Errorf("expected pod key in table:\n%s", result)
		}
	})
}

// --- RunnerSummaryList ---

func TestRunnerSummaryList_FormatText(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		result := RunnerSummaryList(nil).FormatText()
		if result != "No runners available." {
			t.Errorf("expected empty message, got: %s", result)
		}
	})

	t.Run("runner with agents", func(t *testing.T) {
		runners := RunnerSummaryList{
			{
				ID:                1,
				NodeID:            "node-1",
				Status:            "online",
				CurrentPods:       2,
				MaxConcurrentPods: 5,
				Description:       "Main runner",
				AvailableAgents: []AgentSummary{
					{

						Slug:        "claude-code",
						Name:        "Claude Code",
						Description: "Anthropic's Claude Code agent",
						Config: []ConfigFieldSummary{
							{Name: "model", Type: "string", Options: []string{"opus", "sonnet"}, Required: true},
						},
					},
				},
			},
		}
		result := runners.FormatText()
		for _, s := range []string{"Runner #1", "Node: node-1", "Status: online", "Pods: 2/5", "Main runner", "Claude Code (claude-code)", "model (string)", "opus, sonnet", "*required*"} {
			if !strings.Contains(result, s) {
				t.Errorf("expected %q in:\n%s", s, result)
			}
		}
	})

	t.Run("multiple runners", func(t *testing.T) {
		runners := RunnerSummaryList{
			{ID: 1, NodeID: "a", Status: "online", MaxConcurrentPods: 3},
			{ID: 2, NodeID: "b", Status: "offline", MaxConcurrentPods: 1},
		}
		result := runners.FormatText()
		if !strings.Contains(result, "Runner #1") || !strings.Contains(result, "Runner #2") {
			t.Errorf("expected both runners:\n%s", result)
		}
	})
}

// --- RepositoryList ---

func TestRepositoryList_FormatText(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := RepositoryList(nil).FormatText()
		if result != "No repositories configured." {
			t.Errorf("expected empty message, got: %s", result)
		}
	})

	t.Run("with repos", func(t *testing.T) {
		repos := RepositoryList{
			{ID: 1, Name: "my-repo", ProviderType: "gitlab", DefaultBranch: "main", HttpCloneURL: "https://git.example.com/my-repo.git"},
		}
		result := repos.FormatText()
		for _, s := range []string{"| 1 |", "| my-repo |", "| gitlab |", "| main |"} {
			if !strings.Contains(result, s) {
				t.Errorf("expected %q in:\n%s", s, result)
			}
		}
	})
}

// --- BindingList ---

func TestBindingList_FormatText(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := BindingList(nil).FormatText()
		if result != "No bindings found." {
			t.Errorf("expected empty message, got: %s", result)
		}
	})

	t.Run("with bindings", func(t *testing.T) {
		bindings := BindingList{
			{ID: 1, InitiatorPod: "pod-a", TargetPod: "pod-b", Status: BindingStatusActive, GrantedScopes: []BindingScope{ScopePodRead}},
			{ID: 2, InitiatorPod: "pod-c", TargetPod: "pod-d", Status: BindingStatusPending},
		}
		result := bindings.FormatText()
		if !strings.Contains(result, "| 1 |") || !strings.Contains(result, "| 2 |") {
			t.Errorf("expected both bindings:\n%s", result)
		}
		if !strings.Contains(result, "pod:read") {
			t.Errorf("expected scopes:\n%s", result)
		}
	})
}

// --- BoundPodList ---

func TestBoundPodList_FormatText(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := BoundPodList(nil).FormatText()
		if result != "No bound pods." {
			t.Errorf("expected empty message, got: %s", result)
		}
	})

	t.Run("with pods", func(t *testing.T) {
		result := BoundPodList([]string{"pod-a", "pod-b", "pod-c"}).FormatText()
		if result != "Bound pods: pod-a, pod-b, pod-c" {
			t.Errorf("unexpected result: %s", result)
		}
	})

	t.Run("single pod", func(t *testing.T) {
		result := BoundPodList([]string{"pod-x"}).FormatText()
		if result != "Bound pods: pod-x" {
			t.Errorf("unexpected result: %s", result)
		}
	})
}

// --- ChannelList ---

func TestChannelList_FormatText(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := ChannelList(nil).FormatText()
		if result != "No channels found." {
			t.Errorf("expected empty message, got: %s", result)
		}
	})

	t.Run("with channels", func(t *testing.T) {
		channels := ChannelList{
			{ID: 1, Name: "general", MemberCount: 3, IsArchived: false, Description: "General chat"},
			{ID: 2, Name: "dev", MemberCount: 5, IsArchived: true, Description: "Development"},
		}
		result := channels.FormatText()
		if !strings.Contains(result, "| general |") || !strings.Contains(result, "| dev |") {
			t.Errorf("expected both channels:\n%s", result)
		}
	})
}

// --- ChannelMessageList ---

func TestChannelMessageList_FormatText(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := ChannelMessageList(nil).FormatText()
		if result != "No messages." {
			t.Errorf("expected empty message, got: %s", result)
		}
	})

	t.Run("chat log format", func(t *testing.T) {
		msgs := ChannelMessageList{
			{ID: 1, SenderPod: "pod-alpha", Content: "Please review the PR", CreatedAt: "2026-02-20T10:30:00Z"},
			{ID: 2, SenderPod: "pod-beta", Content: "Looking at it now", CreatedAt: "2026-02-20T10:31:00Z"},
			{ID: 3, SenderPod: "pod-beta", Content: "LGTM, approved", CreatedAt: "2026-02-20T10:35:00Z"},
		}
		result := msgs.FormatText()
		if !strings.HasPrefix(result, "3 messages:") {
			t.Errorf("expected count header:\n%s", result)
		}
		if !strings.Contains(result, "[2026-02-20T10:30:00Z] pod-alpha: Please review the PR") {
			t.Errorf("expected first message:\n%s", result)
		}
		if !strings.Contains(result, "[2026-02-20T10:35:00Z] pod-beta: LGTM, approved") {
			t.Errorf("expected last message:\n%s", result)
		}
	})
}

// --- TicketList ---

func TestTicketList_FormatText(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		result := TicketList(nil).FormatText()
		if result != "No tickets found." {
			t.Errorf("expected empty message, got: %s", result)
		}
	})

	t.Run("with tickets", func(t *testing.T) {
		tickets := TicketList{
			{Slug: "AM-123", Title: "Fix authentication bug", Status: TicketStatusInProgress, Priority: TicketPriorityHigh},
			{Slug: "AM-124", Title: "Add dark mode support", Status: TicketStatusTodo, Priority: TicketPriorityMedium},
		}
		result := tickets.FormatText()
		if !strings.Contains(result, "| AM-123 |") {
			t.Errorf("expected AM-123:\n%s", result)
		}
		if !strings.Contains(result, "| AM-124 |") {
			t.Errorf("expected AM-124:\n%s", result)
		}
	})
}

// --- Helper functions ---

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this is a longer string", 10, "this is..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
		{"", 5, ""},
	}

	for _, tt := range tests {
		result := truncate(tt.input, tt.maxLen)
		if result != tt.expected {
			t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, result, tt.expected)
		}
	}
}

func TestJoinScopes(t *testing.T) {
	tests := []struct {
		input    []BindingScope
		expected string
	}{
		{nil, ""},
		{[]BindingScope{}, ""},
		{[]BindingScope{ScopePodRead}, "pod:read"},
		{[]BindingScope{ScopePodRead, ScopePodWrite}, "pod:read, pod:write"},
	}

	for _, tt := range tests {
		result := joinScopes(tt.input)
		if result != tt.expected {
			t.Errorf("joinScopes(%v) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestEscapeTableCell(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal text", "normal text"},
		{"has|pipe", "has\\|pipe"},
		{"multi\nline", "multi line"},
		{"both|and\nnew", "both\\|and new"},
		{"", ""},
	}

	for _, tt := range tests {
		result := escapeTableCell(tt.input)
		if result != tt.expected {
			t.Errorf("escapeTableCell(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
