package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"log/slog"
	"os"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
}

// --- Mock PreferenceRepository ---

type mockPrefRepo struct {
	prefs map[string]*notifDomain.PreferenceRecord
}

func newMockPrefRepo() *mockPrefRepo {
	return &mockPrefRepo{prefs: make(map[string]*notifDomain.PreferenceRecord)}
}

func (m *mockPrefRepo) key(userID int64, source string, entityID string) string {
	return fmt.Sprintf("%d:%s:%s", userID, source, entityID)
}

func (m *mockPrefRepo) GetPreference(_ context.Context, userID int64, source string, entityID string) (*notifDomain.PreferenceRecord, error) {
	rec, ok := m.prefs[m.key(userID, source, entityID)]
	if !ok {
		return nil, nil
	}
	return rec, nil
}

func (m *mockPrefRepo) SetPreference(_ context.Context, rec *notifDomain.PreferenceRecord) error {
	m.prefs[m.key(rec.UserID, rec.Source, rec.EntityID)] = rec
	return nil
}

func (m *mockPrefRepo) ListPreferences(_ context.Context, userID int64) ([]notifDomain.PreferenceRecord, error) {
	var result []notifDomain.PreferenceRecord
	prefix := fmt.Sprintf("%d:", userID)
	for k, v := range m.prefs {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			result = append(result, *v)
		}
	}
	return result, nil
}

func (m *mockPrefRepo) DeletePreference(_ context.Context, userID int64, source string, entityID string) error {
	delete(m.prefs, m.key(userID, source, entityID))
	return nil
}

// --- Mock RecipientResolver ---

type mockResolver struct {
	userIDs []int64
}

func (r *mockResolver) Resolve(_ context.Context, _ string) ([]int64, error) {
	return r.userIDs, nil
}

// --- Tests ---

func TestDispatcher_Dispatch_DirectRecipients(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	repo := newMockPrefRepo()
	store := NewPreferenceStore(repo)
	dispatcher := NewDispatcher(eb, store)

	var received []*eventbus.Event
	var mu sync.Mutex
	eb.Subscribe(eventbus.EventNotification, func(event *eventbus.Event) {
		mu.Lock()
		received = append(received, event)
		mu.Unlock()
	})

	err := dispatcher.Dispatch(context.Background(), &notifDomain.NotificationRequest{
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

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(received))
	}

	// Verify payload
	var payload eventbus.NotificationPayload
	json.Unmarshal(received[0].Data, &payload)
	if payload.Source != "channel:message" {
		t.Errorf("Source = %s, want channel:message", payload.Source)
	}
	if !payload.Channels["toast"] || !payload.Channels["browser"] {
		t.Errorf("Default prefs should enable toast+browser, got channels=%v", payload.Channels)
	}
}

func TestDispatcher_Dispatch_MutedUser(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	repo := newMockPrefRepo()
	// Mute user 10 for channel:message
	repo.SetPreference(context.Background(), &notifDomain.PreferenceRecord{
		UserID: 10, Source: "channel:message", IsMuted: true,
		Channels: notifDomain.ChannelsJSON{"toast": true, "browser": true},
	})
	store := NewPreferenceStore(repo)
	dispatcher := NewDispatcher(eb, store)

	var received []*eventbus.Event
	var mu sync.Mutex
	eb.Subscribe(eventbus.EventNotification, func(event *eventbus.Event) {
		mu.Lock()
		received = append(received, event)
		mu.Unlock()
	})

	err := dispatcher.Dispatch(context.Background(), &notifDomain.NotificationRequest{
		OrganizationID:   1,
		Source:           "channel:message",
		RecipientUserIDs: []int64{10, 20},
		Title:            "Test",
		Priority:         "normal",
	})
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	// User 10 is muted, only user 20 should receive
	if len(received) != 1 {
		t.Fatalf("Expected 1 event (muted user filtered), got %d", len(received))
	}
	if *received[0].TargetUserID != 20 {
		t.Errorf("TargetUserID = %d, want 20", *received[0].TargetUserID)
	}
}

func TestDispatcher_Dispatch_HighPriority_BypassesMute(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	repo := newMockPrefRepo()
	repo.SetPreference(context.Background(), &notifDomain.PreferenceRecord{
		UserID: 10, Source: "channel:mention", IsMuted: true,
		Channels: notifDomain.ChannelsJSON{"toast": true, "browser": true},
	})
	store := NewPreferenceStore(repo)
	dispatcher := NewDispatcher(eb, store)

	var received []*eventbus.Event
	var mu sync.Mutex
	eb.Subscribe(eventbus.EventNotification, func(event *eventbus.Event) {
		mu.Lock()
		received = append(received, event)
		mu.Unlock()
	})

	// High priority should bypass mute
	err := dispatcher.Dispatch(context.Background(), &notifDomain.NotificationRequest{
		OrganizationID:   1,
		Source:           "channel:mention",
		RecipientUserIDs: []int64{10},
		Title:            "@mention",
		Priority:         "high",
	})
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("Expected 1 event (high priority bypasses mute), got %d", len(received))
	}
	var payload eventbus.NotificationPayload
	json.Unmarshal(received[0].Data, &payload)
	// High priority bypasses mute: channels should be forced on
	if !payload.Channels["toast"] || !payload.Channels["browser"] {
		t.Errorf("High priority should force channels on even when muted, got %v", payload.Channels)
	}
}

func TestDispatcher_Dispatch_ExcludeUserIDs(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	repo := newMockPrefRepo()
	store := NewPreferenceStore(repo)
	dispatcher := NewDispatcher(eb, store)
	dispatcher.RegisterResolver("test", &mockResolver{userIDs: []int64{10, 20, 30}})

	var received []*eventbus.Event
	var mu sync.Mutex
	eb.Subscribe(eventbus.EventNotification, func(event *eventbus.Event) {
		mu.Lock()
		received = append(received, event)
		mu.Unlock()
	})

	// Exclude user 20 (the sender)
	err := dispatcher.Dispatch(context.Background(), &notifDomain.NotificationRequest{
		OrganizationID:    1,
		Source:            "channel:message",
		RecipientResolver: "test:42",
		ExcludeUserIDs:    []int64{20},
		Title:             "Test",
		Priority:          "normal",
	})
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 2 {
		t.Fatalf("Expected 2 events (user 20 excluded), got %d", len(received))
	}
	for _, e := range received {
		if *e.TargetUserID == 20 {
			t.Errorf("User 20 should be excluded")
		}
	}
}

func TestDispatcher_Dispatch_Resolver(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	repo := newMockPrefRepo()
	store := NewPreferenceStore(repo)
	dispatcher := NewDispatcher(eb, store)
	dispatcher.RegisterResolver("channel_members", &mockResolver{userIDs: []int64{100, 200}})

	var received []*eventbus.Event
	var mu sync.Mutex
	eb.Subscribe(eventbus.EventNotification, func(event *eventbus.Event) {
		mu.Lock()
		received = append(received, event)
		mu.Unlock()
	})

	err := dispatcher.Dispatch(context.Background(), &notifDomain.NotificationRequest{
		OrganizationID:    1,
		Source:            "channel:message",
		RecipientResolver: "channel_members:42",
		Title:             "#general",
		Priority:          "normal",
	})
	if err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(received))
	}
}

func TestDispatcher_Dispatch_EmptyRecipients(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	repo := newMockPrefRepo()
	store := NewPreferenceStore(repo)
	dispatcher := NewDispatcher(eb, store)

	err := dispatcher.Dispatch(context.Background(), &notifDomain.NotificationRequest{
		OrganizationID: 1,
		Source:         "channel:message",
		Title:          "Test",
	})
	if err != nil {
		t.Fatalf("Dispatch should succeed with no recipients: %v", err)
	}
}
