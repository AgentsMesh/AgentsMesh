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
	svc := newTestService(db)

	// Test that SetEventBus doesn't panic with nil
	svc.SetEventBus(nil)
	if svc.eventBus != nil {
		t.Error("eventBus should be nil")
	}
}

func TestSendMessage_EventBusIntegration(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()
	svc.SetEventBus(eb)
	svc.AddPostSendHook(NewEventPublishHook(eb, nil, nil))

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
	msg, err := svc.SendMessage(ctx, ch.ID, &senderPod, &senderUserID, textContent("Hello from integration test"), nil)
	if err != nil {
		t.Fatalf("SendMessage failed: %v", err)
	}

	if msg.Body != "Hello from integration test" {
		t.Errorf("Content = %s, want Hello from integration test", msg.Body)
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
	svc := newTestService(db)
	ctx := context.Background()

	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()
	svc.SetEventBus(eb)
	svc.AddPostSendHook(NewEventPublishHook(eb, nil, nil))

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
		if _, err := svc.SendMessage(ctx, ch.ID, nil, nil, textContent("Message content"), nil); err != nil {
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
	svc := newTestService(db)
	ctx := context.Background()

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "archived-channel"})
	svc.ArchiveChannel(ctx, ch.ID)

	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()
	svc.SetEventBus(eb)
	svc.AddPostSendHook(NewEventPublishHook(eb, nil, nil))

	var eventCount int
	eb.Subscribe(eventbus.EventChannelMessage, func(event *eventbus.Event) {
		eventCount++
	})

	_, err := svc.SendMessage(ctx, ch.ID, nil, nil, textContent("Should fail"), nil)
	if err != ErrChannelArchived {
		t.Errorf("Expected ErrChannelArchived, got %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	if eventCount != 0 {
		t.Errorf("Expected 0 events for archived channel, got %d", eventCount)
	}
}

func TestEventPublishHook_TargetUserIDs(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()
	svc.SetEventBus(eb)
	// Pass svc as MemberIDProvider — it implements GetMemberUserIDs
	svc.AddPostSendHook(NewEventPublishHook(eb, nil, svc))

	creator := int64(10)
	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "target-test",
		CreatedByUserID: &creator, InitialMemberIDs: []int64{20, 30},
	})

	var receivedEvent *eventbus.Event
	var mu sync.Mutex
	eb.Subscribe(eventbus.EventChannelMessage, func(event *eventbus.Event) {
		mu.Lock()
		receivedEvent = event
		mu.Unlock()
	})

	svc.SendMessage(ctx, ch.ID, nil, &creator, textContent("targeted"), nil)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if receivedEvent == nil {
		t.Fatal("Expected event to be received")
	}
	if len(receivedEvent.TargetUserIDs) == 0 {
		t.Fatal("Expected TargetUserIDs to be non-empty")
	}
	// Should contain all 3 members (creator + 2 initial)
	targetSet := make(map[int64]bool)
	for _, uid := range receivedEvent.TargetUserIDs {
		targetSet[uid] = true
	}
	for _, uid := range []int64{10, 20, 30} {
		if !targetSet[uid] {
			t.Errorf("TargetUserIDs should contain user %d", uid)
		}
	}
}

func TestEventPublishHook_NilMemberProvider(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()
	svc.SetEventBus(eb)
	// Pass nil as MemberIDProvider — should still work (no TargetUserIDs)
	svc.AddPostSendHook(NewEventPublishHook(eb, nil, nil))

	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{OrganizationID: 1, Name: "nil-provider"})

	var receivedEvent *eventbus.Event
	var mu sync.Mutex
	eb.Subscribe(eventbus.EventChannelMessage, func(event *eventbus.Event) {
		mu.Lock()
		receivedEvent = event
		mu.Unlock()
	})

	svc.SendMessage(ctx, ch.ID, nil, nil, textContent("broadcast"), nil)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if receivedEvent == nil {
		t.Fatal("Expected event to be received")
	}
	if len(receivedEvent.TargetUserIDs) != 0 {
		t.Errorf("TargetUserIDs should be empty when MemberIDProvider is nil, got %v", receivedEvent.TargetUserIDs)
	}
}

func TestMemberEvents_JoinPublishesAdded(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()
	svc.SetEventBus(eb)

	creator := int64(10)
	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "member-event-test", CreatedByUserID: &creator,
	})

	var receivedEvents []*eventbus.Event
	var mu sync.Mutex
	eb.Subscribe(eventbus.EventChannelMemberAdded, func(event *eventbus.Event) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
	})

	svc.JoinPublicChannel(ctx, ch.ID, 20)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(receivedEvents) != 1 {
		t.Fatalf("Expected 1 member_added event, got %d", len(receivedEvents))
	}
	if len(receivedEvents[0].TargetUserIDs) < 2 {
		t.Errorf("TargetUserIDs should include both existing members and new member, got %v", receivedEvents[0].TargetUserIDs)
	}
}

func TestMemberEvents_LeavePublishesRemoved(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	ctx := context.Background()

	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()
	svc.SetEventBus(eb)

	creator := int64(10)
	ch, _ := svc.CreateChannel(ctx, &CreateChannelRequest{
		OrganizationID: 1, Name: "leave-event-test", CreatedByUserID: &creator,
		InitialMemberIDs: []int64{20},
	})

	var receivedEvents []*eventbus.Event
	var mu sync.Mutex
	eb.Subscribe(eventbus.EventChannelMemberRemoved, func(event *eventbus.Event) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
	})

	svc.LeaveUserChannel(ctx, ch.ID, 20)
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(receivedEvents) != 1 {
		t.Fatalf("Expected 1 member_removed event, got %d", len(receivedEvents))
	}
	// Removed user should be in TargetUserIDs so they get the notification
	found := false
	for _, uid := range receivedEvents[0].TargetUserIDs {
		if uid == 20 {
			found = true
			break
		}
	}
	if !found {
		t.Error("Removed user (20) should be in TargetUserIDs")
	}
}
