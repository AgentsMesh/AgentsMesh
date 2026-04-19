package channel

import (
	"context"
	"testing"

	channelDomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

func TestNewPodPromptHook_NilRouter(t *testing.T) {
	hook := NewPodPromptHook(nil, nil)

	mc := &MessageContext{
		Channel:  &channelDomain.Channel{ID: 1, Name: "test"},
		Message:  &channelDomain.Message{Body: "hello"},
		Mentions: &MentionResult{PodKeys: []string{"pod-a"}},
	}

	err := hook(context.Background(), mc)
	if err != nil {
		t.Fatalf("Expected nil error for nil router, got %v", err)
	}
}

func TestNewPodPromptHook_NilMentions(t *testing.T) {
	hook := NewPodPromptHook(&mockPodRouter{}, nil)

	mc := &MessageContext{
		Channel: &channelDomain.Channel{ID: 1, Name: "test"},
		Message: &channelDomain.Message{Body: "hello"},
	}

	err := hook(context.Background(), mc)
	if err != nil {
		t.Fatalf("Expected nil error for nil mentions, got %v", err)
	}
}

func TestNewPodPromptHook_EmptyPodKeys(t *testing.T) {
	hook := NewPodPromptHook(&mockPodRouter{}, nil)

	mc := &MessageContext{
		Channel:  &channelDomain.Channel{ID: 1, Name: "test"},
		Message:  &channelDomain.Message{Body: "hello"},
		Mentions: &MentionResult{PodKeys: []string{}},
	}

	err := hook(context.Background(), mc)
	if err != nil {
		t.Fatalf("Expected nil error for empty pod keys, got %v", err)
	}
}

func TestNewPodPromptHook_SendsPrompt(t *testing.T) {
	router := &mockPodRouter{}
	hook := NewPodPromptHook(router, nil)

	mc := &MessageContext{
		Channel:  &channelDomain.Channel{ID: 42, Name: "dev", OrganizationID: 1},
		Message:  &channelDomain.Message{Body: "@abcd1234 fix bug", ChannelID: 42},
		Mentions: &MentionResult{PodKeys: []string{"abcd1234efgh5678"}},
	}

	err := hook(context.Background(), mc)
	if err != nil {
		t.Fatalf("Hook failed: %v", err)
	}

	if len(router.prompts) != 1 {
		t.Fatalf("Expected 1 prompt, got %d", len(router.prompts))
	}
	if router.prompts[0].podKey != "abcd1234efgh5678" {
		t.Errorf("PodKey = %s, want abcd1234efgh5678", router.prompts[0].podKey)
	}
}

func TestNewPodPromptHook_SkipsSenderPod(t *testing.T) {
	router := &mockPodRouter{}
	hook := NewPodPromptHook(router, nil)

	senderPod := "abcd1234efgh5678"
	mc := &MessageContext{
		Channel:  &channelDomain.Channel{ID: 1, Name: "test"},
		Message:  &channelDomain.Message{Body: "echo", SenderPod: &senderPod, ChannelID: 1},
		Mentions: &MentionResult{PodKeys: []string{"abcd1234efgh5678"}},
	}

	err := hook(context.Background(), mc)
	if err != nil {
		t.Fatalf("Hook failed: %v", err)
	}

	if len(router.prompts) != 0 {
		t.Errorf("Expected 0 prompts (sender skipped), got %d", len(router.prompts))
	}
}

func TestNewPodPromptHook_OfflineNotice(t *testing.T) {
	router := &mockPodRouter{failKeys: map[string]bool{"offline-pod": true}}
	writer := &mockSystemWriter{}
	hook := NewPodPromptHook(router, writer)

	mc := &MessageContext{
		Channel:  &channelDomain.Channel{ID: 1, Name: "test"},
		Message:  &channelDomain.Message{Body: "hi", ChannelID: 1},
		Mentions: &MentionResult{PodKeys: []string{"offline-pod"}},
	}

	err := hook(context.Background(), mc)
	if err != nil {
		t.Fatalf("Hook failed: %v", err)
	}

	if len(writer.messages) != 1 {
		t.Fatalf("Expected 1 offline notice, got %d", len(writer.messages))
	}
	if writer.messages[0].MessageType != channelDomain.MessageTypeSystem {
		t.Error("Offline notice should be a system message")
	}
}

func TestWriteOfflineNotice_NilWriter(t *testing.T) {
	writeOfflineNotice(context.Background(), nil, 1, "pod-x")
}

func TestNewEventPublishHook_WithMembers(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	hook := NewEventPublishHook(eb, nil, svc)

	creator := int64(10)
	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "hook-member", CreatedByUserID: &creator,
		InitialMemberIDs: []int64{20},
	})

	msg := &channelDomain.Message{
		ID: 1, ChannelID: ch.ID, Body: "hello",
		SenderUserID: &creator, MessageType: "text",
	}

	mc := &MessageContext{
		Channel: ch,
		Message: msg,
	}

	err := hook(ctx, mc)
	if err != nil {
		t.Fatalf("Hook failed: %v", err)
	}
}

// --- Mocks for hooks_coverage ---

type promptRecord struct {
	podKey string
	prompt string
}

type mockPodRouter struct {
	prompts  []promptRecord
	failKeys map[string]bool
}

func (m *mockPodRouter) RoutePrompt(podKey string, prompt string) error {
	if m.failKeys != nil && m.failKeys[podKey] {
		return ErrChannelNotFound
	}
	m.prompts = append(m.prompts, promptRecord{podKey: podKey, prompt: prompt})
	return nil
}

type mockSystemWriter struct {
	messages []*channelDomain.Message
}

func (m *mockSystemWriter) CreateMessage(_ context.Context, msg *channelDomain.Message) error {
	m.messages = append(m.messages, msg)
	return nil
}
