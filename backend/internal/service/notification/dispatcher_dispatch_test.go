package notification

import (
	"context"
	"encoding/json"
	"testing"

	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
)

func TestDispatcher_DirectRecipients(t *testing.T) {
	d, pusher := newTestDispatcher(newMockPrefRepo())

	err := d.Dispatch(context.Background(), &notifDomain.NotificationRequest{
		OrganizationID:   1,
		Source:           "channel:message",
		RecipientUserIDs: []int64{10, 20},
		Title:            "Test",
		Body:             "Hello",
		Priority:         "normal",
	})
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	pushes := pusher.getPushes()
	if len(pushes) != 2 {
		t.Fatalf("expected 2 pushes, got %d", len(pushes))
	}

	payload := decodeWirePayload(t, pushes[0].Data)
	if payload.Source != "channel:message" {
		t.Errorf("Source = %s, want channel:message", payload.Source)
	}
	if !payload.Channels["toast"] || !payload.Channels["browser"] {
		t.Errorf("default prefs should enable toast+browser, got %v", payload.Channels)
	}
}

func TestDispatcher_MutedUser(t *testing.T) {
	repo := newMockPrefRepo()
	repo.SetPreference(context.Background(), &notifDomain.PreferenceRecord{
		UserID: 10, Source: "channel:message", IsMuted: true,
		Channels: notifDomain.ChannelsJSON{"toast": true, "browser": true},
	})
	d, pusher := newTestDispatcher(repo)

	d.Dispatch(context.Background(), &notifDomain.NotificationRequest{
		OrganizationID:   1,
		Source:           "channel:message",
		RecipientUserIDs: []int64{10, 20},
		Title:            "Test",
		Priority:         "normal",
	})

	pushes := pusher.getPushes()
	if len(pushes) != 1 {
		t.Fatalf("expected 1 push (muted user filtered), got %d", len(pushes))
	}
	if pushes[0].UserID != 20 {
		t.Errorf("UserID = %d, want 20", pushes[0].UserID)
	}
}

func TestDispatcher_HighPriority_BypassesMute(t *testing.T) {
	repo := newMockPrefRepo()
	repo.SetPreference(context.Background(), &notifDomain.PreferenceRecord{
		UserID: 10, Source: "channel:mention", IsMuted: true,
		Channels: notifDomain.ChannelsJSON{"toast": true, "browser": true},
	})
	d, pusher := newTestDispatcher(repo)

	d.Dispatch(context.Background(), &notifDomain.NotificationRequest{
		OrganizationID:   1,
		Source:           "channel:mention",
		RecipientUserIDs: []int64{10},
		Title:            "@mention",
		Priority:         "high",
	})

	pushes := pusher.getPushes()
	if len(pushes) != 1 {
		t.Fatalf("expected 1 push (high priority bypasses mute), got %d", len(pushes))
	}
	payload := decodeWirePayload(t, pushes[0].Data)
	if !payload.Channels["toast"] || !payload.Channels["browser"] {
		t.Errorf("high priority should force channels on, got %v", payload.Channels)
	}
}

func TestDispatcher_ExcludeUserIDs(t *testing.T) {
	d, pusher := newTestDispatcher(newMockPrefRepo())
	d.RegisterResolver("test", &mockResolver{userIDs: []int64{10, 20, 30}})

	d.Dispatch(context.Background(), &notifDomain.NotificationRequest{
		OrganizationID:    1,
		Source:            "channel:message",
		RecipientResolver: "test:42",
		ExcludeUserIDs:    []int64{20},
		Title:             "Test",
		Priority:          "normal",
	})

	pushes := pusher.getPushes()
	if len(pushes) != 2 {
		t.Fatalf("expected 2 pushes (user 20 excluded), got %d", len(pushes))
	}
	for _, p := range pushes {
		if p.UserID == 20 {
			t.Errorf("user 20 should be excluded")
		}
	}
}

func TestDispatcher_Resolver(t *testing.T) {
	d, pusher := newTestDispatcher(newMockPrefRepo())
	d.RegisterResolver("channel_members", &mockResolver{userIDs: []int64{100, 200}})

	d.Dispatch(context.Background(), &notifDomain.NotificationRequest{
		OrganizationID:    1,
		Source:            "channel:message",
		RecipientResolver: "channel_members:42",
		Title:             "#general",
		Priority:          "normal",
	})

	pushes := pusher.getPushes()
	if len(pushes) != 2 {
		t.Fatalf("expected 2 pushes, got %d", len(pushes))
	}
}

func TestDispatcher_EmptyRecipients(t *testing.T) {
	d, pusher := newTestDispatcher(newMockPrefRepo())

	err := d.Dispatch(context.Background(), &notifDomain.NotificationRequest{
		OrganizationID: 1,
		Source:         "channel:message",
		Title:          "Test",
	})
	if err != nil {
		t.Fatalf("should succeed with no recipients: %v", err)
	}
	if len(pusher.getPushes()) != 0 {
		t.Error("expected 0 pushes for empty recipients")
	}
}

func TestDispatcher_WireEventFormat(t *testing.T) {
	d, pusher := newTestDispatcher(newMockPrefRepo())

	d.Dispatch(context.Background(), &notifDomain.NotificationRequest{
		OrganizationID:   1,
		Source:           "task:completed",
		SourceEntityID:   "pod-123",
		RecipientUserIDs: []int64{42},
		Title:            "Task Completed",
		Body:             "Pod pod-123 finished",
		Link:             "/workspace?pod=pod-123",
		Priority:         "normal",
	})

	pushes := pusher.getPushes()
	if len(pushes) != 1 {
		t.Fatalf("expected 1 push, got %d", len(pushes))
	}

	var wire notificationWireEvent
	json.Unmarshal(pushes[0].Data, &wire)
	if wire.Type != "notification" {
		t.Errorf("wire.Type = %s, want notification", wire.Type)
	}
	if wire.Category != "notification" {
		t.Errorf("wire.Category = %s, want notification", wire.Category)
	}
	if wire.OrganizationID != 1 {
		t.Errorf("wire.OrganizationID = %d, want 1", wire.OrganizationID)
	}
	if wire.TargetUserID == nil || *wire.TargetUserID != 42 {
		t.Error("wire.TargetUserID should be 42")
	}
	if wire.EntityID != "pod-123" {
		t.Errorf("wire.EntityID = %s, want pod-123", wire.EntityID)
	}
	if wire.Timestamp == 0 {
		t.Error("wire.Timestamp should be set")
	}
}
