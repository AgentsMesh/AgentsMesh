package channel

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

// mockPodCreatorResolver is a test double for PodCreatorResolver.
type mockPodCreatorResolver struct {
	pods map[string]*agentpod.Pod
}

func (m *mockPodCreatorResolver) GetPodByKey(_ context.Context, podKey string) (*agentpod.Pod, error) {
	if pod, ok := m.pods[podKey]; ok {
		return pod, nil
	}
	return nil, ErrChannelNotFound
}

func TestCreateChannel_Visibility(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	userID := int64(10)

	t.Run("default visibility is public", func(t *testing.T) {
		ch, err := svc.CreateChannel(ctx, &CreateChannelRequest{
			OrganizationID: 1, Name: "pub-default", CreatedByUserID: &userID,
		})
		if err != nil {
			t.Fatalf("CreateChannel failed: %v", err)
		}
		if ch.Visibility != channel.VisibilityPublic {
			t.Errorf("Visibility = %s, want public", ch.Visibility)
		}
	})

	t.Run("explicit private visibility", func(t *testing.T) {
		ch, err := svc.CreateChannel(ctx, &CreateChannelRequest{
			OrganizationID: 1, Name: "priv-ch", CreatedByUserID: &userID,
			Visibility: channel.VisibilityPrivate,
		})
		if err != nil {
			t.Fatalf("CreateChannel failed: %v", err)
		}
		if ch.Visibility != channel.VisibilityPrivate {
			t.Errorf("Visibility = %s, want private", ch.Visibility)
		}
	})
}

func TestCreateChannel_CreatorAutoJoins(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	userID := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "creator-join", CreatedByUserID: &userID,
	})

	ok, err := svc.IsMember(ctx, ch.ID, userID)
	if err != nil {
		t.Fatalf("IsMember failed: %v", err)
	}
	if !ok {
		t.Error("Creator should be auto-joined as member")
	}
}

func TestCreateChannel_InitialMembers(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "with-members", CreatedByUserID: &creator,
		InitialMemberIDs: []int64{20, 30},
	})

	for _, uid := range []int64{10, 20, 30} {
		ok, _ := svc.IsMember(ctx, ch.ID, uid)
		if !ok {
			t.Errorf("User %d should be a member", uid)
		}
	}
	notMember, _ := svc.IsMember(ctx, ch.ID, 99)
	if notMember {
		t.Error("User 99 should not be a member")
	}
}

func TestSendMessage_PrivateChannelRejectsNonMember(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)
	stranger := int64(99)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "private-msg",
		CreatedByUserID: &creator, Visibility: channel.VisibilityPrivate,
	})

	// Creator can send
	_, err := svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("hello"), nil)
	if err != nil {
		t.Errorf("Creator should be able to send: %v", err)
	}

	// Stranger cannot
	_, err = svc.SendMessage(ctx, ch.ID, nil, &stranger, textContent("intruder"), nil)
	if err != ErrNotMember {
		t.Errorf("Expected ErrNotMember, got %v", err)
	}
}

func TestSendMessage_PublicChannelAutoJoins(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)
	newUser := int64(20)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "public-auto", CreatedByUserID: &creator,
	})

	// New user is not a member yet
	ok, _ := svc.IsMember(ctx, ch.ID, newUser)
	if ok {
		t.Fatal("User should not be a member before sending")
	}

	// Sending auto-joins in public channels
	_, err := svc.SendMessage(ctx, ch.ID, nil, &newUser, textContent("hi"), nil)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	ok, _ = svc.IsMember(ctx, ch.ID, newUser)
	if !ok {
		t.Error("User should be auto-joined after sending to public channel")
	}
}

func TestJoinPublicChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	public, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "join-pub", CreatedByUserID: &creator,
	})
	private, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "join-priv", CreatedByUserID: &creator,
		Visibility: channel.VisibilityPrivate,
	})

	t.Run("can join public", func(t *testing.T) {
		err := svc.JoinPublicChannel(ctx, public.ID, 20)
		if err != nil {
			t.Errorf("JoinPublicChannel failed: %v", err)
		}
		ok, _ := svc.IsMember(ctx, public.ID, 20)
		if !ok {
			t.Error("User should be a member after joining")
		}
	})

	t.Run("cannot join private", func(t *testing.T) {
		err := svc.JoinPublicChannel(ctx, private.ID, 20)
		if err != ErrChannelPrivate {
			t.Errorf("Expected ErrChannelPrivate, got %v", err)
		}
	})
}

func TestInviteMembers(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "invite-test", CreatedByUserID: &creator,
		Visibility: channel.VisibilityPrivate,
	})

	t.Run("member can invite", func(t *testing.T) {
		err := svc.InviteMembers(ctx, ch.ID, creator, []int64{20, 30})
		if err != nil {
			t.Errorf("InviteMembers failed: %v", err)
		}
		for _, uid := range []int64{20, 30} {
			ok, _ := svc.IsMember(ctx, ch.ID, uid)
			if !ok {
				t.Errorf("User %d should be a member after invitation", uid)
			}
		}
	})

	t.Run("non-member cannot invite", func(t *testing.T) {
		err := svc.InviteMembers(ctx, ch.ID, 99, []int64{40})
		if err != ErrNotMember {
			t.Errorf("Expected ErrNotMember, got %v", err)
		}
	})
}

func TestLeaveUserChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "leave-test", CreatedByUserID: &creator,
		InitialMemberIDs: []int64{20},
	})

	ok, _ := svc.IsMember(ctx, ch.ID, 20)
	if !ok {
		t.Fatal("User 20 should be a member")
	}

	err := svc.LeaveUserChannel(ctx, ch.ID, 20)
	if err != nil {
		t.Fatalf("LeaveUserChannel failed: %v", err)
	}

	ok, _ = svc.IsMember(ctx, ch.ID, 20)
	if ok {
		t.Error("User 20 should no longer be a member after leaving")
	}
}

func TestLeaveUserChannel_CreatorCannotLeave(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "creator-leave", CreatedByUserID: &creator,
		InitialMemberIDs: []int64{20},
	})

	err := svc.LeaveUserChannel(ctx, ch.ID, creator)
	if err != ErrNotCreator {
		t.Errorf("Expected ErrNotCreator when creator tries to leave, got %v", err)
	}

	// Creator should still be a member
	ok, _ := svc.IsMember(ctx, ch.ID, creator)
	if !ok {
		t.Error("Creator should still be a member after failed leave attempt")
	}
}

func TestRemoveMember_RequiresCreatorRole(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "remove-auth", CreatedByUserID: &creator,
		InitialMemberIDs: []int64{20, 30},
	})

	t.Run("regular member cannot remove others", func(t *testing.T) {
		err := svc.RemoveMember(ctx, ch.ID, 20, 30)
		if err != ErrNotCreator {
			t.Errorf("Expected ErrNotCreator, got %v", err)
		}
		// User 30 should still be a member
		ok, _ := svc.IsMember(ctx, ch.ID, 30)
		if !ok {
			t.Error("User 30 should still be a member")
		}
	})

	t.Run("creator can remove members", func(t *testing.T) {
		err := svc.RemoveMember(ctx, ch.ID, creator, 30)
		if err != nil {
			t.Errorf("Creator should be able to remove members: %v", err)
		}
		ok, _ := svc.IsMember(ctx, ch.ID, 30)
		if ok {
			t.Error("User 30 should be removed")
		}
	})

	t.Run("self-removal is allowed", func(t *testing.T) {
		err := svc.RemoveMember(ctx, ch.ID, 20, 20)
		if err != nil {
			t.Errorf("Self-removal should be allowed: %v", err)
		}
	})
}

func TestSetMemberMuted_RequiresMembership(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "mute-test", CreatedByUserID: &creator,
		Visibility: channel.VisibilityPrivate,
	})

	err := svc.SetMemberMuted(ctx, ch.ID, 99, true)
	if err != ErrNotMember {
		t.Errorf("Expected ErrNotMember for non-member mute, got %v", err)
	}

	err = svc.SetMemberMuted(ctx, ch.ID, creator, true)
	if err != nil {
		t.Errorf("Creator mute should succeed: %v", err)
	}
}

func TestJoinChannel_PodCreatorAutoJoins(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)
	podCreatorUserID := int64(50)

	// Set up mock pod resolver
	svc.SetPodCreatorResolver(&mockPodCreatorResolver{
		pods: map[string]*agentpod.Pod{
			"agent-pod-1": {CreatedByID: podCreatorUserID},
		},
	})

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "pod-join-test", CreatedByUserID: &creator,
	})

	// Pod creator is NOT a member yet
	ok, _ := svc.IsMember(ctx, ch.ID, podCreatorUserID)
	if ok {
		t.Fatal("Pod creator should not be a member before pod joins")
	}

	// Pod joins channel
	err := svc.JoinChannel(ctx, ch.ID, "agent-pod-1")
	if err != nil {
		t.Fatalf("JoinChannel failed: %v", err)
	}

	// Pod creator should now be auto-joined
	ok, _ = svc.IsMember(ctx, ch.ID, podCreatorUserID)
	if !ok {
		t.Error("Pod creator should be auto-joined when their pod joins the channel")
	}
}

func TestJoinChannel_NestedPodCreatorPenetration(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)
	humanUserID := int64(42)

	// Pod B was created by Pod A, which was created by humanUserID.
	// Due to creator penetration at pod creation time, Pod B's CreatedByID = humanUserID.
	svc.SetPodCreatorResolver(&mockPodCreatorResolver{
		pods: map[string]*agentpod.Pod{
			"nested-pod-b": {CreatedByID: humanUserID},
		},
	})

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "nested-test", CreatedByUserID: &creator,
	})

	_ = svc.JoinChannel(ctx, ch.ID, "nested-pod-b")

	ok, _ := svc.IsMember(ctx, ch.ID, humanUserID)
	if !ok {
		t.Error("Human creator should be auto-joined through nested pod creator penetration")
	}
}
