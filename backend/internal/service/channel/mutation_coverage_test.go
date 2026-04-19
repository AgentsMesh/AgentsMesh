package channel

import (
	"context"
	"sync"
	"testing"
	"time"

	channelDomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

func TestEditMessage_NotInChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch1, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "edit-ch1", CreatedByUserID: &creator,
	})
	ch2, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "edit-ch2", CreatedByUserID: &creator,
	})

	msg, _ := svc.SendMessage(ctx, ch1.ID, nil, &creator, textContent("hello"), nil)

	// Try editing a message from ch1 using ch2's ID
	_, err := svc.EditMessage(ctx, ch2.ID, msg.ID, creator, textContent("hack"))
	if err != ErrMessageNotFound {
		t.Errorf("Expected ErrMessageNotFound, got %v", err)
	}
}

func TestEditMessage_NilSenderUserID(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "edit-nil-sender",
	})

	// System message has nil SenderUserID
	msg, _ := svc.SendSystemMessage(ctx, ch.ID, "system msg")

	_, err := svc.EditMessage(ctx, ch.ID, msg.ID, 99, textContent("hack"))
	if err != ErrNotMessageSender {
		t.Errorf("Expected ErrNotMessageSender for nil sender, got %v", err)
	}
}

func TestEditMessage_NonExistentChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	_, err := svc.EditMessage(ctx, 99999, 1, 10, textContent("content"))
	if err != ErrChannelNotFound {
		t.Errorf("Expected ErrChannelNotFound, got %v", err)
	}
}

func TestDeleteMessage_ArchivedChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "del-archived", CreatedByUserID: &creator,
	})

	msg, _ := svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("to-delete"), nil)

	svc.ArchiveChannel(ctx, ch.ID)

	err := svc.DeleteMessage(ctx, ch.ID, msg.ID, creator)
	if err != ErrChannelArchived {
		t.Errorf("Expected ErrChannelArchived, got %v", err)
	}
}

func TestDeleteMessage_NotInChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()
	creator := int64(10)

	ch1, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "del-ch1", CreatedByUserID: &creator,
	})
	ch2, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "del-ch2", CreatedByUserID: &creator,
	})

	msg, _ := svc.SendMessage(ctx, ch1.ID, nil, &creator, textContent("hello"), nil)

	err := svc.DeleteMessage(ctx, ch2.ID, msg.ID, creator)
	if err != ErrMessageNotFound {
		t.Errorf("Expected ErrMessageNotFound, got %v", err)
	}
}

func TestDeleteMessage_NilSenderUserID(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "del-nil-sender",
	})

	msg, _ := svc.SendSystemMessage(ctx, ch.ID, "system msg")

	err := svc.DeleteMessage(ctx, ch.ID, msg.ID, 99)
	if err != ErrNotMessageSender {
		t.Errorf("Expected ErrNotMessageSender for nil sender, got %v", err)
	}
}

func TestDeleteMessage_NonExistentChannel(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	err := svc.DeleteMessage(ctx, 99999, 1, 10)
	if err != ErrChannelNotFound {
		t.Errorf("Expected ErrChannelNotFound, got %v", err)
	}
}

func TestPublishChannelEvent_WithEventBus(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()
	svc.SetEventBus(eb)

	creator := int64(10)
	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "pub-ch-evt", CreatedByUserID: &creator,
	})

	var receivedEdited []*eventbus.Event
	var receivedDeleted []*eventbus.Event
	var mu sync.Mutex

	eb.Subscribe(eventbus.EventChannelMessageEdited, func(event *eventbus.Event) {
		mu.Lock()
		receivedEdited = append(receivedEdited, event)
		mu.Unlock()
	})
	eb.Subscribe(eventbus.EventChannelMessageDeleted, func(event *eventbus.Event) {
		mu.Lock()
		receivedDeleted = append(receivedDeleted, event)
		mu.Unlock()
	})

	msg, _ := svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("test"), nil)

	svc.EditMessage(ctx, ch.ID, msg.ID, creator, textContent("edited"))
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if len(receivedEdited) != 1 {
		t.Errorf("Expected 1 edited event, got %d", len(receivedEdited))
	}
	mu.Unlock()

	msg2, _ := svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("del-target"), nil)
	svc.DeleteMessage(ctx, ch.ID, msg2.ID, creator)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if len(receivedDeleted) != 1 {
		t.Errorf("Expected 1 deleted event, got %d", len(receivedDeleted))
	}
	mu.Unlock()
}

func TestPublishChannelEvent_NilEventBus(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)

	// No eventBus — publishChannelEvent should return immediately
	svc.publishChannelEvent(1, 1, eventbus.EventChannelMessageEdited, map[string]interface{}{
		"channel_id": 1,
	})
}

func TestNewEventPublishHook_NilEventBus(t *testing.T) {
	hook := NewEventPublishHook(nil, nil, nil)

	mc := &MessageContext{
		Channel: &channelDomain.Channel{ID: 1, OrganizationID: 10},
		Message: &channelDomain.Message{ID: 1, Body: "test"},
	}

	err := hook(context.Background(), mc)
	if err != nil {
		t.Fatalf("Expected nil error for nil eventBus, got %v", err)
	}
}
