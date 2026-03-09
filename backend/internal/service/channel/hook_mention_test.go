package channel

import (
	"context"
	"testing"

	channelDomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

// --- Mocks ---

type mockUserLookup struct {
	validIDs []int64
	err      error
}

func (m *mockUserLookup) GetUsersByUsernames(_ context.Context, _ int64, _ []string) (map[string]int64, error) {
	return nil, nil
}

func (m *mockUserLookup) ValidateUserIDs(_ context.Context, _ int64, _ []int64) ([]int64, error) {
	return m.validIDs, m.err
}

type mockPodLookup struct {
	validKeys []string
	err       error
}

func (m *mockPodLookup) GetPodsByKeys(_ context.Context, _ int64, _ []string) ([]string, error) {
	return m.validKeys, m.err
}

type mockMetadataRepo struct {
	channelDomain.ChannelRepository // embed to satisfy interface; only UpdateMessageMetadata is used
	updatedMeta                     map[string]interface{}
	updateCalled                    bool
}

func (m *mockMetadataRepo) UpdateMessageMetadata(_ context.Context, _ int64, metadata map[string]interface{}) error {
	m.updateCalled = true
	m.updatedMeta = metadata
	return nil
}

// --- Tests ---

func TestMentionValidatorHook_NilMentions(t *testing.T) {
	hook := NewMentionValidatorHook(nil, nil, nil)

	mc := &MessageContext{
		Channel: &channelDomain.Channel{ID: 1, OrganizationID: 10},
		Message: &channelDomain.Message{ID: 1, Content: "hello"},
	}

	if err := hook(context.Background(), mc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMentionValidatorHook_AllValid(t *testing.T) {
	repo := &mockMetadataRepo{}
	hook := NewMentionValidatorHook(
		&mockUserLookup{validIDs: []int64{1, 2}},
		&mockPodLookup{validKeys: []string{"pod-a", "pod-b"}},
		repo,
	)

	mc := &MessageContext{
		Channel:  &channelDomain.Channel{ID: 1, OrganizationID: 10},
		Message:  &channelDomain.Message{ID: 100, Content: "hi @alice @pod-a"},
		Mentions: &MentionResult{UserIDs: []int64{1, 2}, PodKeys: []string{"pod-a", "pod-b"}},
	}

	if err := hook(context.Background(), mc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All mentions are valid — no DB update needed
	if repo.updateCalled {
		t.Error("UpdateMessageMetadata should not be called when all mentions are valid")
	}
	if len(mc.Mentions.UserIDs) != 2 {
		t.Errorf("UserIDs = %v, want [1,2]", mc.Mentions.UserIDs)
	}
	if len(mc.Mentions.PodKeys) != 2 {
		t.Errorf("PodKeys = %v, want [pod-a, pod-b]", mc.Mentions.PodKeys)
	}
}

func TestMentionValidatorHook_PrunesInvalidUsers(t *testing.T) {
	repo := &mockMetadataRepo{}
	hook := NewMentionValidatorHook(
		&mockUserLookup{validIDs: []int64{1}}, // user 2 is invalid
		&mockPodLookup{validKeys: []string{"pod-a"}},
		repo,
	)

	mc := &MessageContext{
		Channel:  &channelDomain.Channel{ID: 1, OrganizationID: 10},
		Message:  &channelDomain.Message{ID: 100, Content: "hi"},
		Mentions: &MentionResult{UserIDs: []int64{1, 2}, PodKeys: []string{"pod-a"}},
	}

	if err := hook(context.Background(), mc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Invalid user removed — metadata synced to DB
	if !repo.updateCalled {
		t.Error("UpdateMessageMetadata should be called when mentions are pruned")
	}
	if len(mc.Mentions.UserIDs) != 1 || mc.Mentions.UserIDs[0] != 1 {
		t.Errorf("UserIDs = %v, want [1]", mc.Mentions.UserIDs)
	}
	if repo.updatedMeta[MetaMentionedUsers] == nil {
		t.Error("metadata should contain mentioned_users")
	}
}

func TestMentionValidatorHook_PrunesInvalidPods(t *testing.T) {
	repo := &mockMetadataRepo{}
	hook := NewMentionValidatorHook(
		&mockUserLookup{validIDs: []int64{}},
		&mockPodLookup{validKeys: []string{}}, // all pods invalid
		repo,
	)

	mc := &MessageContext{
		Channel:  &channelDomain.Channel{ID: 1, OrganizationID: 10},
		Message:  &channelDomain.Message{ID: 100, Content: "hi"},
		Mentions: &MentionResult{UserIDs: []int64{}, PodKeys: []string{"bad-pod"}},
	}

	if err := hook(context.Background(), mc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !repo.updateCalled {
		t.Error("UpdateMessageMetadata should be called when pods are pruned")
	}
	if len(mc.Mentions.PodKeys) != 0 {
		t.Errorf("PodKeys = %v, want []", mc.Mentions.PodKeys)
	}
}

func TestMentionValidatorHook_NilLookups(t *testing.T) {
	// Nil lookups should skip validation without error
	hook := NewMentionValidatorHook(nil, nil, nil)

	mc := &MessageContext{
		Channel:  &channelDomain.Channel{ID: 1, OrganizationID: 10},
		Message:  &channelDomain.Message{ID: 100, Content: "hi"},
		Mentions: &MentionResult{UserIDs: []int64{1}, PodKeys: []string{"pod-a"}},
	}

	if err := hook(context.Background(), mc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Mentions remain unchanged because lookups are nil
	if len(mc.Mentions.UserIDs) != 1 {
		t.Errorf("UserIDs should remain unchanged, got %v", mc.Mentions.UserIDs)
	}
	if len(mc.Mentions.PodKeys) != 1 {
		t.Errorf("PodKeys should remain unchanged, got %v", mc.Mentions.PodKeys)
	}
}
