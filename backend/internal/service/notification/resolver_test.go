package notification

import (
	"context"
	"errors"
	"testing"
)

// --- Mock PodInfoProvider ---

type mockPodInfo struct {
	orgID     int64
	creatorID int64
	err       error
}

func (m *mockPodInfo) GetPodOrganizationAndCreator(_ context.Context, _ string) (int64, int64, error) {
	return m.orgID, m.creatorID, m.err
}

func TestPodCreatorResolver_Resolve(t *testing.T) {
	resolver := NewPodCreatorResolver(&mockPodInfo{orgID: 1, creatorID: 42})

	ids, err := resolver.Resolve(context.Background(), "pod-abc")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if len(ids) != 1 || ids[0] != 42 {
		t.Errorf("Expected [42], got %v", ids)
	}
}

func TestPodCreatorResolver_Error(t *testing.T) {
	resolver := NewPodCreatorResolver(&mockPodInfo{err: errors.New("not found")})

	_, err := resolver.Resolve(context.Background(), "pod-xyz")
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// --- Mock ChannelMemberProvider ---

type mockMemberProvider struct {
	userIDs []int64
}

func (m *mockMemberProvider) GetMemberUserIDs(_ context.Context, _ int64) ([]int64, error) {
	return m.userIDs, nil
}

func (m *mockMemberProvider) GetNonMutedMemberUserIDs(_ context.Context, _ int64) ([]int64, error) {
	return m.userIDs, nil
}

func TestChannelMemberResolver_Resolve(t *testing.T) {
	resolver := NewChannelMemberResolver(&mockMemberProvider{userIDs: []int64{10, 20, 30}})

	ids, err := resolver.Resolve(context.Background(), "42")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if len(ids) != 3 {
		t.Errorf("Expected 3 members, got %d", len(ids))
	}
}

func TestChannelMemberResolver_InvalidParam(t *testing.T) {
	resolver := NewChannelMemberResolver(&mockMemberProvider{userIDs: []int64{1}})

	ids, err := resolver.Resolve(context.Background(), "not-a-number")
	if err != nil {
		t.Fatalf("Should not error on invalid param: %v", err)
	}
	if ids != nil {
		t.Errorf("Expected nil for invalid channel ID, got %v", ids)
	}
}
