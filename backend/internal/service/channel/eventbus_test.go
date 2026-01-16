package channel

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

func TestSetEventBus(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)

	// Test that SetEventBus doesn't panic with nil
	svc.SetEventBus(nil)
	if svc.eventBus != nil {
		t.Error("eventBus should be nil")
	}
}

func TestSendMessage_EventBusIntegration(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()
	svc.SetEventBus(eb)

	ch, err := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 42, Name: "integration-test"})
	if err != nil {
		t.Fatalf("CreateChannel failed: %v", err)
	}

	var receivedEvents []*eventbus.Event
	var mu sync.Mutex
	eb.Subscribe(eventbus.EventChannelMessage, func(event *eventbus.Event) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
	})

	senderPod := "test-pod-123"
	senderUserID := int64(99)
	msg, err := svc.SendMessage(ctx, ch.ID, &senderPod, &senderUserID, "text", "Hello from integration test", nil)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if msg.Content != "Hello from integration test" {
		t.Errorf("Content = %s, want Hello from integration test", msg.Content)
	}

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(receivedEvents) != 1 {
		t.Errorf("Expected 1 event, got %d", len(receivedEvents))
		return
	}

	event := receivedEvents[0]
	if event.Type != eventbus.EventChannelMessage {
		t.Errorf("Event type = %s, want %s", event.Type, eventbus.EventChannelMessage)
	}
	if event.OrganizationID != 42 {
		t.Errorf("OrganizationID = %d, want 42", event.OrganizationID)
	}
	if event.Category != eventbus.CategoryEntity {
		t.Errorf("Category = %s, want %s", event.Category, eventbus.CategoryEntity)
	}
}

func TestSendMessage_EventBusIntegration_MultipleMessages(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()
	svc.SetEventBus(eb)

	ch, err := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "multi-msg-channel"})
	if err != nil {
		t.Fatalf("CreateChannel failed: %v", err)
	}

	var eventCount int
	var mu sync.Mutex
	eb.Subscribe(eventbus.EventChannelMessage, func(event *eventbus.Event) {
		mu.Lock()
		eventCount++
		mu.Unlock()
	})

	for i := 0; i < 3; i++ {
		if _, err := svc.SendMessage(ctx, ch.ID, nil, nil, "text", "Message content", nil); err != nil {
			t.Fatalf("SendMessage %d failed: %v", i, err)
		}
	}

	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if eventCount != 3 {
		t.Errorf("Expected 3 events, got %d", eventCount)
	}
}

func TestSendMessage_ArchivedChannel_NoEvent(t *testing.T) {
	db := setupTestDB(t)
	svc := NewService(db)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "archived-channel"})
	svc.ArchiveChannel(ctx, ch.ID)

	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()
	svc.SetEventBus(eb)

	var eventCount int
	eb.Subscribe(eventbus.EventChannelMessage, func(event *eventbus.Event) {
		eventCount++
	})

	_, err := svc.SendMessage(ctx, ch.ID, nil, nil, "text", "Should fail", nil)
	if err != ErrChannelArchived {
		t.Errorf("Expected ErrChannelArchived, got %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if eventCount != 0 {
		t.Errorf("Expected 0 events for archived channel, got %d", eventCount)
	}
}
