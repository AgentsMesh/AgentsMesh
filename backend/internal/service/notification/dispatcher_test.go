package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
)

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
	rec := m.prefs[m.key(userID, source, entityID)]
	if rec != nil {
		return rec, nil
	}
	return nil, nil
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

type mockResolver struct {
	userIDs []int64
}

func (r *mockResolver) Resolve(_ context.Context, _ string) ([]int64, error) {
	return r.userIDs, nil
}

type pushedMessage struct {
	UserID int64
	Data   []byte
}

type mockPusher struct {
	mu     sync.Mutex
	pushes []pushedMessage
}

func (p *mockPusher) PushToUser(_ context.Context, userID int64, data []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pushes = append(p.pushes, pushedMessage{UserID: userID, Data: data})
	return nil
}

func (p *mockPusher) getPushes() []pushedMessage {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]pushedMessage{}, p.pushes...)
}

func newTestDispatcher(repo *mockPrefRepo) (*Dispatcher, *mockPusher) {
	pusher := &mockPusher{}
	store := NewPreferenceStore(repo)
	return NewDispatcher(pusher, store), pusher
}

func decodeWirePayload(t *testing.T, data []byte) notificationPayload {
	t.Helper()
	var wire notificationWireEvent
	if err := json.Unmarshal(data, &wire); err != nil {
		t.Fatalf("unmarshal wire event: %v", err)
	}
	var payload notificationPayload
	if err := json.Unmarshal(wire.Data, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	return payload
}
