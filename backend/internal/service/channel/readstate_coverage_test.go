package channel

import (
	"context"
	"strconv"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

func containsID(ids []int64, id int64) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func userMentionContent(userID int64) channel.MessageContent {
	return channel.MessageContent{
		Kind: "text",
		Blocks: []channel.Block{{
			Type: "paragraph",
			Elements: []channel.InlineElement{
				{Type: channel.InlineMention, EntityType: channel.EntityUser, EntityKey: strconv.FormatInt(userID, 10)},
			},
		}},
	}
}

// Marking a never-opened channel unread must seed the cursor to the latest
// message so it shows a sticky dot (unread=0 + flag), not unread=COUNT(all).
func TestSetManuallyUnread_NeverOpened_SeedsCursor(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)
	viewer := int64(20)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "mark-unread-fresh", CreatedByUserID: &creator,
	})
	for i := 0; i < 3; i++ {
		svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("m"), nil)
	}

	if err := svc.SetManuallyUnread(ctx, ch.ID, viewer); err != nil {
		t.Fatalf("SetManuallyUnread: %v", err)
	}

	counts, _ := svc.GetChannelSummaries(ctx, viewer)
	s := counts[ch.ID]
	if s.Unread != 0 {
		t.Errorf("mark-unread on never-opened channel should seed cursor (unread=0), got %d", s.Unread)
	}
	if !s.ManuallyUnread {
		t.Error("ManuallyUnread flag should be set in the summary")
	}
}

func TestSetMemberPinned_PublicChannelAutoJoins(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)
	viewer := int64(20)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "pin-pub", CreatedByUserID: &creator,
	})

	if err := svc.SetMemberPinned(ctx, ch.ID, viewer, true); err != nil {
		t.Fatalf("pinning a public channel should auto-join, got %v", err)
	}
	if ok, _ := svc.IsMember(ctx, ch.ID, viewer); !ok {
		t.Error("viewer should be auto-joined after pinning a public channel")
	}
}

func TestSetMemberPinned_PrivateChannelRejectsNonMember(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "pin-priv", CreatedByUserID: &creator,
		Visibility: channel.VisibilityPrivate,
	})
	if err := svc.SetMemberPinned(ctx, ch.ID, 99, true); err != ErrNotMember {
		t.Errorf("expected ErrNotMember pinning a private channel as non-member, got %v", err)
	}
}

// Read receipts are author-only: the author may query who read their own
// message, but another member must not enumerate receipts for it.
func TestGetMessageReadBy_AuthorOnly(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	author := int64(10)
	reader := int64(20)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "readby", CreatedByUserID: &author,
		InitialMemberIDs: []int64{reader},
	})
	msg, _ := svc.SendMessage(ctx, ch.ID, nil, &author, textContent("hi"), nil)
	svc.MarkRead(ctx, ch.ID, reader, msg.ID)

	ids, err := svc.GetMessageReadBy(ctx, ch.ID, msg.ID, author)
	if err != nil {
		t.Fatalf("author GetMessageReadBy: %v", err)
	}
	if !containsID(ids, reader) {
		t.Errorf("expected reader %d in read-by, got %v", reader, ids)
	}
	if _, err := svc.GetMessageReadBy(ctx, ch.ID, msg.ID, reader); err != ErrNotMessageSender {
		t.Errorf("non-author should be denied, got %v", err)
	}
}

func TestGetMessageReadBy_ExcludesDepartedMembers(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	author := int64(10)
	reader := int64(20)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "readby-leave", CreatedByUserID: &author,
		InitialMemberIDs: []int64{reader},
	})
	msg, _ := svc.SendMessage(ctx, ch.ID, nil, &author, textContent("hi"), nil)
	svc.MarkRead(ctx, ch.ID, reader, msg.ID)
	if err := svc.LeaveUserChannel(ctx, ch.ID, reader); err != nil {
		t.Fatalf("LeaveUserChannel: %v", err)
	}

	ids, _ := svc.GetMessageReadBy(ctx, ch.ID, msg.ID, author)
	if containsID(ids, reader) {
		t.Errorf("departed member should be excluded from read-by, got %v", ids)
	}
}

// The summary's mention count must come from the dialect-branched JSON mention
// predicate (only the @member message counts, not the plain one).
func TestGetChannelSummaries_CountsUserMention(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)
	member := int64(20)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "mentions", CreatedByUserID: &creator,
		InitialMemberIDs: []int64{member},
	})
	svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("plain"), nil)
	svc.SendMessage(ctx, ch.ID, nil, &creator, userMentionContent(member), nil)

	counts, _ := svc.GetChannelSummaries(ctx, member)
	s := counts[ch.ID]
	if s.Unread != 2 {
		t.Errorf("expected 2 unread, got %d", s.Unread)
	}
	if s.Mention != 1 {
		t.Errorf("expected 1 mention (the @member message), got %d", s.Mention)
	}
}
