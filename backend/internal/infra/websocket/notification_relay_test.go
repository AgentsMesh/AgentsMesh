package websocket

import (
	"context"
	"testing"
	"time"
)

func TestNotificationRelay_PushToUser_LocalDelivery(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	relay := NewNotificationRelay(hub, nil, nil)
	err := relay.PushToUser(context.Background(), 42, []byte(`{"type":"notification"}`))
	if err != nil {
		t.Fatalf("PushToUser failed: %v", err)
	}
}

func TestNotificationRelay_PushToUser_NilRedis(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	relay := NewNotificationRelay(hub, nil, nil)

	// Should not panic with nil Redis
	err := relay.PushToUser(context.Background(), 1, []byte(`{}`))
	if err != nil {
		t.Fatalf("expected no error with nil redis, got: %v", err)
	}
}

func TestNotificationRelay_InstanceID_Unique(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	r1 := NewNotificationRelay(hub, nil, nil)
	r2 := NewNotificationRelay(hub, nil, nil)

	if r1.instanceID == r2.instanceID {
		t.Error("two relays should have different instance IDs")
	}
}

func TestNotificationRelay_StartSubscriber_NilRedis(t *testing.T) {
	hub := NewHub()
	defer hub.Close()

	relay := NewNotificationRelay(hub, nil, nil)

	// Should not panic, just log a warning
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	relay.StartSubscriber(ctx)
}
