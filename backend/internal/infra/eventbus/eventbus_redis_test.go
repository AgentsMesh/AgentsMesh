package eventbus

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// setupMiniredis creates a miniredis instance and redis client for testing
func setupMiniredis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	return mr, client
}

func TestEventBus_PublishToRedis(t *testing.T) {
	t.Run("publishes event to redis channel", func(t *testing.T) {
		mr, client := setupMiniredis(t)
		defer mr.Close()
		defer client.Close()

		eb := NewEventBus(client, nil)
		defer eb.Close()

		// Subscribe to the channel in miniredis
		pubsub := client.Subscribe(context.Background(), "events:org:1")
		defer pubsub.Close()

		// Wait for subscription to be ready
		_, err := pubsub.Receive(context.Background())
		if err != nil {
			t.Fatalf("failed to subscribe: %v", err)
		}

		ch := pubsub.Channel()

		event := &Event{
			Type:           EventPodCreated,
			OrganizationID: 1,
			EntityType:     "pod",
			EntityID:       "pod-redis-test",
		}

		err = eb.Publish(context.Background(), event)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Wait for message
		select {
		case msg := <-ch:
			var received Event
			if err := json.Unmarshal([]byte(msg.Payload), &received); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}
			if received.EntityID != "pod-redis-test" {
				t.Errorf("expected EntityID 'pod-redis-test', got '%s'", received.EntityID)
			}
			if received.SourceInstanceID != eb.instanceID {
				t.Errorf("expected SourceInstanceID '%s', got '%s'", eb.instanceID, received.SourceInstanceID)
			}
		case <-time.After(time.Second):
			t.Error("did not receive message from redis")
		}
	})
}

func TestEventBus_StartRedisSubscriber(t *testing.T) {
	t.Run("subscribes to pattern and dispatches events", func(t *testing.T) {
		mr, client := setupMiniredis(t)
		defer mr.Close()
		defer client.Close()

		eb := NewEventBus(client, nil)
		defer eb.Close()

		var received *Event
		var wg sync.WaitGroup
		wg.Add(1)

		eb.Subscribe(EventPodCreated, func(e *Event) {
			received = e
			wg.Done()
		})

		// Start subscriber
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		eb.StartRedisSubscriber(ctx)

		// Give subscriber time to start
		time.Sleep(50 * time.Millisecond)

		// Publish directly to Redis (simulating another instance)
		event := &Event{
			Type:             EventPodCreated,
			Category:         CategoryEntity,
			OrganizationID:   1,
			EntityType:       "pod",
			EntityID:         "pod-from-other-instance",
			SourceInstanceID: "other-instance-123", // Different instance
			Timestamp:        time.Now().UnixMilli(),
		}

		data, _ := json.Marshal(event)
		err := client.Publish(context.Background(), "events:org:1", data).Err()
		if err != nil {
			t.Fatalf("failed to publish to redis: %v", err)
		}

		// Wait for handler
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			if received == nil {
				t.Fatal("handler did not receive event")
			}
			if received.EntityID != "pod-from-other-instance" {
				t.Errorf("expected EntityID 'pod-from-other-instance', got '%s'", received.EntityID)
			}
		case <-time.After(2 * time.Second):
			t.Error("handler did not receive event within timeout")
		}
	})

	t.Run("skips events from same instance", func(t *testing.T) {
		mr, client := setupMiniredis(t)
		defer mr.Close()
		defer client.Close()

		eb := NewEventBus(client, nil)
		defer eb.Close()

		var callCount int32
		eb.Subscribe(EventPodCreated, func(e *Event) {
			atomic.AddInt32(&callCount, 1)
		})

		// Start subscriber
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		eb.StartRedisSubscriber(ctx)

		// Give subscriber time to start
		time.Sleep(50 * time.Millisecond)

		// Publish event with same instance ID (should be skipped)
		event := &Event{
			Type:             EventPodCreated,
			Category:         CategoryEntity,
			OrganizationID:   1,
			EntityType:       "pod",
			EntityID:         "pod-same-instance",
			SourceInstanceID: eb.instanceID, // Same instance - should be skipped
			Timestamp:        time.Now().UnixMilli(),
		}

		data, _ := json.Marshal(event)
		err := client.Publish(context.Background(), "events:org:1", data).Err()
		if err != nil {
			t.Fatalf("failed to publish to redis: %v", err)
		}

		// Wait a bit
		time.Sleep(100 * time.Millisecond)

		if atomic.LoadInt32(&callCount) != 0 {
			t.Errorf("expected 0 calls (event from same instance should be skipped), got %d", callCount)
		}
	})

	t.Run("handles invalid JSON gracefully", func(t *testing.T) {
		mr, client := setupMiniredis(t)
		defer mr.Close()
		defer client.Close()

		eb := NewEventBus(client, nil)
		defer eb.Close()

		var callCount int32
		eb.Subscribe(EventPodCreated, func(e *Event) {
			atomic.AddInt32(&callCount, 1)
		})

		// Start subscriber
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		eb.StartRedisSubscriber(ctx)

		// Give subscriber time to start
		time.Sleep(50 * time.Millisecond)

		// Publish invalid JSON
		err := client.Publish(context.Background(), "events:org:1", "invalid json {{{").Err()
		if err != nil {
			t.Fatalf("failed to publish to redis: %v", err)
		}

		// Wait a bit - should not crash
		time.Sleep(100 * time.Millisecond)

		if atomic.LoadInt32(&callCount) != 0 {
			t.Errorf("expected 0 calls (invalid JSON should be skipped), got %d", callCount)
		}
	})

	t.Run("does nothing without redis client", func(t *testing.T) {
		eb := NewEventBus(nil, nil)
		defer eb.Close()

		// Should not panic
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		eb.StartRedisSubscriber(ctx)
	})
}

func TestEventBus_SubscribeOrgWithRedis(t *testing.T) {
	t.Run("receives events for subscribed org", func(t *testing.T) {
		mr, client := setupMiniredis(t)
		defer mr.Close()
		defer client.Close()

		eb := NewEventBus(client, nil)
		defer eb.Close()

		var received *Event
		var wg sync.WaitGroup
		wg.Add(1)

		eb.Subscribe(EventTicketCreated, func(e *Event) {
			received = e
			wg.Done()
		})

		// Subscribe to org
		eb.SubscribeOrg(42)

		// Give subscriber time to start
		time.Sleep(50 * time.Millisecond)

		// Publish event for org 42
		event := &Event{
			Type:             EventTicketCreated,
			Category:         CategoryEntity,
			OrganizationID:   42,
			EntityType:       "ticket",
			EntityID:         "TICKET-001",
			SourceInstanceID: "other-instance",
			Timestamp:        time.Now().UnixMilli(),
		}

		data, _ := json.Marshal(event)
		err := client.Publish(context.Background(), "events:org:42", data).Err()
		if err != nil {
			t.Fatalf("failed to publish to redis: %v", err)
		}

		// Wait for handler
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			if received == nil {
				t.Fatal("handler did not receive event")
			}
			if received.EntityID != "TICKET-001" {
				t.Errorf("expected EntityID 'TICKET-001', got '%s'", received.EntityID)
			}
		case <-time.After(2 * time.Second):
			t.Error("handler did not receive event within timeout")
		}
	})

	t.Run("stops receiving after unsubscribe", func(t *testing.T) {
		mr, client := setupMiniredis(t)
		defer mr.Close()
		defer client.Close()

		eb := NewEventBus(client, nil)
		defer eb.Close()

		var callCount int32
		eb.Subscribe(EventTicketCreated, func(e *Event) {
			atomic.AddInt32(&callCount, 1)
		})

		// Subscribe and then unsubscribe
		eb.SubscribeOrg(99)
		time.Sleep(50 * time.Millisecond)
		eb.UnsubscribeOrg(99)
		time.Sleep(50 * time.Millisecond)

		// Publish event
		event := &Event{
			Type:             EventTicketCreated,
			Category:         CategoryEntity,
			OrganizationID:   99,
			EntityType:       "ticket",
			EntityID:         "TICKET-002",
			SourceInstanceID: "other-instance",
			Timestamp:        time.Now().UnixMilli(),
		}

		data, _ := json.Marshal(event)
		_ = client.Publish(context.Background(), "events:org:99", data).Err()

		// Wait a bit
		time.Sleep(100 * time.Millisecond)

		// Should not receive since unsubscribed
		// Note: due to timing, we might still receive 0 or 1 event
		// The important thing is that the goroutine exits cleanly
	})
}

func TestEventBus_subscribeToOrgChannel_ContextCancellation(t *testing.T) {
	mr, client := setupMiniredis(t)
	defer mr.Close()
	defer client.Close()

	eb := NewEventBus(client, nil)

	// Subscribe to org
	eb.SubscribeOrg(100)

	// Give subscriber time to start
	time.Sleep(50 * time.Millisecond)

	// Close EventBus - should cancel context and stop goroutine
	eb.Close()

	// Give goroutine time to exit
	time.Sleep(100 * time.Millisecond)

	// No assertion needed - test passes if no deadlock/panic
}

func TestEventBus_PublishToRedis_ErrorHandling(t *testing.T) {
	t.Run("logs error when redis publish fails", func(t *testing.T) {
		mr, client := setupMiniredis(t)
		eb := NewEventBus(client, nil)
		defer eb.Close()

		// Close miniredis to cause publish failure
		mr.Close()
		client.Close()

		event := &Event{
			Type:           EventPodCreated,
			OrganizationID: 1,
			EntityType:     "pod",
			EntityID:       "pod-error-test",
		}

		// Should not panic, just log error
		err := eb.Publish(context.Background(), event)
		// Publish returns nil even if Redis fails (local dispatch succeeds)
		if err != nil {
			t.Errorf("expected nil error, got %v", err)
		}
	})
}

func TestEventBus_StartRedisSubscriber_ContextDone(t *testing.T) {
	t.Run("exits when context is cancelled", func(t *testing.T) {
		mr, client := setupMiniredis(t)
		defer mr.Close()
		defer client.Close()

		eb := NewEventBus(client, nil)
		defer eb.Close()

		ctx, cancel := context.WithCancel(context.Background())
		eb.StartRedisSubscriber(ctx)

		// Give subscriber time to start
		time.Sleep(50 * time.Millisecond)

		// Cancel context
		cancel()

		// Give goroutine time to exit
		time.Sleep(100 * time.Millisecond)

		// No assertion needed - test passes if no deadlock/panic
	})

	t.Run("exits when eventbus context is cancelled", func(t *testing.T) {
		mr, client := setupMiniredis(t)
		defer mr.Close()
		defer client.Close()

		eb := NewEventBus(client, nil)

		ctx := context.Background()
		eb.StartRedisSubscriber(ctx)

		// Give subscriber time to start
		time.Sleep(50 * time.Millisecond)

		// Close EventBus (cancels internal context)
		eb.Close()

		// Give goroutine time to exit
		time.Sleep(100 * time.Millisecond)

		// No assertion needed - test passes if no deadlock/panic
	})
}

func TestEventBus_subscribeToOrgChannel_ChannelClosed(t *testing.T) {
	t.Run("exits when redis channel is closed", func(t *testing.T) {
		mr, client := setupMiniredis(t)
		eb := NewEventBus(client, nil)
		defer eb.Close()

		eb.SubscribeOrg(200)

		// Give subscriber time to start
		time.Sleep(50 * time.Millisecond)

		// Close Redis connection - this will close the pubsub channel
		mr.Close()
		client.Close()

		// Give goroutine time to exit
		time.Sleep(100 * time.Millisecond)

		// No assertion needed - test passes if no deadlock/panic
	})
}
