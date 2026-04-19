package channel

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
)

func newErrorService(t *testing.T, failOn ...string) (*Service, *errorInjectingRepo) {
	t.Helper()
	db := setupTestDB(t)
	fail := make(map[string]bool, len(failOn))
	for _, m := range failOn {
		fail[m] = true
	}
	repo := &errorInjectingRepo{
		ChannelRepository: infra.NewChannelRepository(db),
		failOn:            fail,
	}
	return NewService(repo), repo
}

func TestCreateChannel_RepoCreateError(t *testing.T) {
	svc, _ := newErrorService(t, "Create")
	_, err := svc.CreateChannel(context.Background(), &CreateChannelRequest{
		OrganizationID: 1, Name: "fail-create",
	})
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestCreateChannel_GetByOrgAndNameError(t *testing.T) {
	svc, _ := newErrorService(t, "GetByOrgAndName")
	_, err := svc.CreateChannel(context.Background(), &CreateChannelRequest{
		OrganizationID: 1, Name: "fail-check",
	})
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestGetChannel_RepoError(t *testing.T) {
	svc, _ := newErrorService(t, "GetByID")
	_, err := svc.GetChannel(context.Background(), 1)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestGetChannelByName_RepoError(t *testing.T) {
	svc, _ := newErrorService(t, "GetByOrgAndName")
	_, err := svc.GetChannelByName(context.Background(), 1, "test")
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestArchiveChannel_Error(t *testing.T) {
	svc, _ := newErrorService(t, "SetArchived")
	err := svc.ArchiveChannel(context.Background(), 1)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestUnarchiveChannel_Error(t *testing.T) {
	svc, _ := newErrorService(t, "SetArchived")
	err := svc.UnarchiveChannel(context.Background(), 1)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestDeleteChannel_Error(t *testing.T) {
	svc, _ := newErrorService(t, "DeleteWithCleanup")
	err := svc.DeleteChannel(context.Background(), 1)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestDeleteChannelsByOrg_Error(t *testing.T) {
	svc, _ := newErrorService(t, "DeleteChannelsByOrg")
	err := svc.DeleteChannelsByOrg(context.Background(), 1)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestUpdateChannel_UpdateFieldsError(t *testing.T) {
	svc, repo := newErrorService(t)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "upd-fail",
	})

	repo.failOn["UpdateFields"] = true
	name := "new-name"
	_, err := svc.UpdateChannel(ctx, ch.ID, &name, nil, nil)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestRequireMembership_RepoError(t *testing.T) {
	svc, _ := newErrorService(t, "IsMember")
	err := svc.requireMembership(context.Background(), 1, 1)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestRequireCreatorRole_RepoError(t *testing.T) {
	svc, _ := newErrorService(t, "GetMemberRole")
	err := svc.requireCreatorRole(context.Background(), 1, 1)
	if err != ErrNotMember {
		t.Errorf("Expected ErrNotMember, got %v", err)
	}
}

func TestJoinPublicChannel_AddMemberError(t *testing.T) {
	svc, repo := newErrorService(t)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "join-fail",
	})

	repo.failOn["AddMemberWithRole"] = true
	err := svc.JoinPublicChannel(ctx, ch.ID, 20)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestLeaveUserChannel_RemoveMemberError(t *testing.T) {
	svc, repo := newErrorService(t)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "leave-fail", CreatedByUserID: &creator,
		InitialMemberIDs: []int64{20},
	})

	repo.failOn["RemoveMember"] = true
	err := svc.LeaveUserChannel(ctx, ch.ID, 20)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestRemoveMember_RemoveMemberError(t *testing.T) {
	svc, repo := newErrorService(t)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "rm-fail", CreatedByUserID: &creator,
		InitialMemberIDs: []int64{20},
	})

	repo.failOn["RemoveMember"] = true
	// Self-removal to bypass creator check
	err := svc.RemoveMember(ctx, ch.ID, 20, 20)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestSetMemberMuted_RepoError(t *testing.T) {
	svc, repo := newErrorService(t)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "mute-fail", CreatedByUserID: &creator,
	})

	repo.failOn["SetMemberMuted"] = true
	err := svc.SetMemberMuted(ctx, ch.ID, creator, true)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestJoinChannel_AddPodError(t *testing.T) {
	svc, _ := newErrorService(t, "AddPodToChannel")
	err := svc.JoinChannel(context.Background(), 1, "pod-x")
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestSendMessage_CreateMessageError(t *testing.T) {
	svc, repo := newErrorService(t)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "msg-fail",
	})

	repo.failOn["CreateMessage"] = true
	_, err := svc.SendMessage(ctx, ch.ID, nil, nil, textContent("fail"), nil)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestGetMessages_RepoError(t *testing.T) {
	svc, _ := newErrorService(t, "GetMessages")
	_, _, err := svc.GetMessages(context.Background(), 1, nil, nil, 10)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestGetMessagesMentioning_RepoError(t *testing.T) {
	svc, _ := newErrorService(t, "GetMessagesMentioning")
	_, _, err := svc.GetMessagesMentioning(context.Background(), 1, "pod", 10)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestGetMessagesByCursor_RepoError(t *testing.T) {
	svc, _ := newErrorService(t, "GetMessagesBefore")
	_, _, err := svc.GetMessagesByCursor(context.Background(), 1, 1, 10)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestGetRecentMessages_RepoError(t *testing.T) {
	svc, _ := newErrorService(t, "GetRecentMessages")
	_, err := svc.GetRecentMessages(context.Background(), 1, 10)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestEditMessage_GetMessageError(t *testing.T) {
	svc, repo := newErrorService(t)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "edit-get-fail", CreatedByUserID: &creator,
	})
	msg, _ := svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("orig"), nil)

	repo.failOn["GetMessageByID"] = true
	_, err := svc.EditMessage(ctx, ch.ID, msg.ID, creator, textContent("new"))
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestEditMessage_UpdateContentError(t *testing.T) {
	svc, repo := newErrorService(t)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "edit-upd-fail", CreatedByUserID: &creator,
	})
	msg, _ := svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("orig"), nil)

	repo.failOn["UpdateMessage"] = true
	_, err := svc.EditMessage(ctx, ch.ID, msg.ID, creator, textContent("new"))
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestDeleteMessage_GetMessageError(t *testing.T) {
	svc, repo := newErrorService(t)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "del-get-fail", CreatedByUserID: &creator,
	})
	msg, _ := svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("orig"), nil)

	repo.failOn["GetMessageByID"] = true
	err := svc.DeleteMessage(ctx, ch.ID, msg.ID, creator)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestDeleteMessage_SoftDeleteError(t *testing.T) {
	svc, repo := newErrorService(t)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "del-soft-fail", CreatedByUserID: &creator,
	})
	msg, _ := svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("orig"), nil)

	repo.failOn["SoftDeleteMessage"] = true
	err := svc.DeleteMessage(ctx, ch.ID, msg.ID, creator)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestTrackAccess_RepoError(t *testing.T) {
	svc, _ := newErrorService(t, "UpsertAccess")
	podKey := "pod-x"
	err := svc.TrackAccess(context.Background(), 1, &podKey, nil)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestCreateBinding_RepoError(t *testing.T) {
	svc, _ := newErrorService(t, "CreateBinding")
	_, err := svc.CreateBinding(context.Background(), 1, "a", "b", nil)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestGetBinding_RepoError(t *testing.T) {
	svc, _ := newErrorService(t, "GetBindingByID")
	_, err := svc.GetBinding(context.Background(), 1)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestGetBindingByPods_RepoError(t *testing.T) {
	svc, _ := newErrorService(t, "GetBindingByPods")
	_, err := svc.GetBindingByPods(context.Background(), "a", "b")
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestRejectBinding_RepoError(t *testing.T) {
	svc, _ := newErrorService(t, "UpdateBindingFields")
	err := svc.RejectBinding(context.Background(), 1)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestRevokeBinding_RepoError(t *testing.T) {
	svc, _ := newErrorService(t, "UpdateBindingFields")
	err := svc.RevokeBinding(context.Background(), 1)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestApproveBinding_RepoError(t *testing.T) {
	svc, _ := newErrorService(t, "UpdateBindingFields")
	err := svc.ApproveBinding(context.Background(), 1, nil)
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}

func TestApproveBinding_Success(t *testing.T) {
	svc, _ := newErrorService(t)
	ctx := context.Background()

	binding, _ := svc.CreateBinding(ctx, 1, "approve-a", "approve-b", nil)
	err := svc.ApproveBinding(ctx, binding.ID, []string{"read"})
	if err != nil {
		t.Fatalf("ApproveBinding failed: %v", err)
	}
}

func TestInviteMembers_GetChannelErrorAfterAuth(t *testing.T) {
	svc, repo := newErrorService(t)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "inv-get-fail", CreatedByUserID: &creator,
	})

	repo.failOn["GetByID"] = true
	err := svc.InviteMembers(ctx, ch.ID, creator, []int64{20})
	if err != errInjected {
		t.Errorf("Expected injected error, got %v", err)
	}
}
