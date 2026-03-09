package channel

import (
	"context"
	"testing"

	channelDomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
)

// --- Mock UserNameResolver ---

type mockUserNameResolver struct {
	names map[int64]string
}

func (m *mockUserNameResolver) GetUsername(_ context.Context, userID int64) (string, error) {
	if name, ok := m.names[userID]; ok {
		return name, nil
	}
	return "", nil
}

// --- Mock NotificationDispatcher ---

type mockNotifDispatcher struct {
	calls []notifDomain.NotificationRequest
}

func (m *mockNotifDispatcher) Dispatch(_ context.Context, req *notifDomain.NotificationRequest) error {
	m.calls = append(m.calls, *req)
	return nil
}

// --- Tests ---

func TestResolveSenderName(t *testing.T) {
	resolver := &mockUserNameResolver{names: map[int64]string{42: "alice"}}

	tests := []struct {
		name     string
		pod      *string
		userID   *int64
		resolver UserNameResolver
		want     string
	}{
		{"Pod sender", strPtr("test-pod"), nil, nil, "test-pod"},
		{"User with resolver", nil, intPtr(42), resolver, "alice"},
		{"User without resolver", nil, intPtr(42), nil, "User#42"},
		{"System (no sender)", nil, nil, nil, "System"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &MessageContext{
				Message: &channelDomain.Message{SenderPod: tt.pod, SenderUserID: tt.userID},
			}
			got := resolveSenderName(context.Background(), mc, tt.resolver)
			if got != tt.want {
				t.Errorf("resolveSenderName() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNotificationHook_ExcludesSender(t *testing.T) {
	dispatcher := &mockNotifDispatcher{}
	hook := NewNotificationHook(dispatcher, nil)

	mc := &MessageContext{
		Channel: &channelDomain.Channel{ID: 1, OrganizationID: 10, Name: "general"},
		Message: &channelDomain.Message{
			SenderUserID: intPtr(42),
			Content:      "Hello world",
		},
	}

	err := hook(context.Background(), mc)
	if err != nil {
		t.Fatalf("Hook failed: %v", err)
	}

	if len(dispatcher.calls) != 1 {
		t.Fatalf("Expected 1 dispatch call (channel:message), got %d", len(dispatcher.calls))
	}

	call := dispatcher.calls[0]
	if len(call.ExcludeUserIDs) != 1 || call.ExcludeUserIDs[0] != 42 {
		t.Errorf("ExcludeUserIDs = %v, want [42]", call.ExcludeUserIDs)
	}
}

func TestNotificationHook_MentionHighPriority(t *testing.T) {
	dispatcher := &mockNotifDispatcher{}
	hook := NewNotificationHook(dispatcher, nil)

	mc := &MessageContext{
		Channel: &channelDomain.Channel{ID: 1, OrganizationID: 10, Name: "dev"},
		Message: &channelDomain.Message{
			SenderPod: strPtr("agent-pod"),
			Content:   "Hey @alice check this out",
		},
		Mentions: &MentionResult{UserIDs: []int64{99}},
	}

	err := hook(context.Background(), mc)
	if err != nil {
		t.Fatalf("Hook failed: %v", err)
	}

	if len(dispatcher.calls) != 2 {
		t.Fatalf("Expected 2 dispatch calls (message + mention), got %d", len(dispatcher.calls))
	}

	// First call: channel:message
	if dispatcher.calls[0].Source != notifDomain.SourceChannelMessage {
		t.Errorf("First call source = %s, want %s", dispatcher.calls[0].Source, notifDomain.SourceChannelMessage)
	}
	if dispatcher.calls[0].Priority != notifDomain.PriorityNormal {
		t.Errorf("First call priority = %s, want normal", dispatcher.calls[0].Priority)
	}

	// Second call: channel:mention
	if dispatcher.calls[1].Source != notifDomain.SourceChannelMention {
		t.Errorf("Second call source = %s, want %s", dispatcher.calls[1].Source, notifDomain.SourceChannelMention)
	}
	if dispatcher.calls[1].Priority != notifDomain.PriorityHigh {
		t.Errorf("Second call priority = %s, want high", dispatcher.calls[1].Priority)
	}
	if len(dispatcher.calls[1].RecipientUserIDs) != 1 || dispatcher.calls[1].RecipientUserIDs[0] != 99 {
		t.Errorf("Mention recipients = %v, want [99]", dispatcher.calls[1].RecipientUserIDs)
	}
}

func TestNotificationHook_NilDispatcher(t *testing.T) {
	hook := NewNotificationHook(nil, nil)

	mc := &MessageContext{
		Channel: &channelDomain.Channel{ID: 1, Name: "test"},
		Message: &channelDomain.Message{Content: "hello"},
	}

	// Should not panic
	err := hook(context.Background(), mc)
	if err != nil {
		t.Fatalf("Hook failed: %v", err)
	}
}

func TestNotificationHook_TruncatesBody(t *testing.T) {
	dispatcher := &mockNotifDispatcher{}
	hook := NewNotificationHook(dispatcher, nil)

	longContent := ""
	for i := 0; i < 200; i++ {
		longContent += "x"
	}

	mc := &MessageContext{
		Channel: &channelDomain.Channel{ID: 1, OrganizationID: 10, Name: "ch"},
		Message: &channelDomain.Message{Content: longContent},
	}

	hook(context.Background(), mc)

	body := dispatcher.calls[0].Body
	// "System: " (8 chars) + truncated content (100 chars = 97 + "...") = 108
	if len(body) > 120 {
		t.Errorf("Body should be truncated, length = %d", len(body))
	}
}
