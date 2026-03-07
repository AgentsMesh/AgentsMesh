package runner

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/stretchr/testify/assert"
)

// mockPodInfoGetter implements PodInfoGetter for testing
type mockPodInfoGetter struct {
	orgID        int64
	creatorID    int64
	err          error
	titleErr     error
	updatedTitle string
}

func (m *mockPodInfoGetter) GetPodOrganizationAndCreator(ctx context.Context, podKey string) (orgID, creatorID int64, err error) {
	return m.orgID, m.creatorID, m.err
}

func (m *mockPodInfoGetter) UpdatePodTitle(ctx context.Context, podKey, title string) error {
	m.updatedTitle = title
	return m.titleErr
}

func TestNewOSCDetector(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	getter := &mockPodInfoGetter{orgID: 100, creatorID: 1}
	detector := NewOSCDetector(eb, getter)

	assert.NotNil(t, detector)
	assert.Equal(t, eb, detector.eventBus)
	assert.Equal(t, getter, detector.podInfoGetter)
}

func TestOSCDetector_PublishNotification_Success(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	getter := &mockPodInfoGetter{orgID: 100, creatorID: 1}
	detector := NewOSCDetector(eb, getter)

	// Subscribe to capture events
	var receivedEvents []*eventbus.Event
	var mu sync.Mutex
	eb.Subscribe(eventbus.EventTerminalNotification, func(event *eventbus.Event) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
	})

	// Publish notification
	ctx := context.Background()
	result := detector.PublishNotification(ctx, "pod-test-123", "Build Complete", "Your build finished successfully")

	assert.True(t, result)
}

func TestOSCDetector_PublishNotification_NilEventBus(t *testing.T) {
	getter := &mockPodInfoGetter{orgID: 100, creatorID: 1}
	detector := &OSCDetector{
		eventBus:      nil,
		podInfoGetter: getter,
	}

	ctx := context.Background()
	result := detector.PublishNotification(ctx, "pod-123", "Title", "Body")

	assert.False(t, result)
}

func TestOSCDetector_PublishNotification_NilPodInfoGetter(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	detector := &OSCDetector{
		eventBus:      eb,
		podInfoGetter: nil,
	}

	ctx := context.Background()
	result := detector.PublishNotification(ctx, "pod-123", "Title", "Body")

	assert.False(t, result)
}

func TestOSCDetector_PublishNotification_PodInfoError(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	getter := &mockPodInfoGetter{err: errors.New("pod not found")}
	detector := NewOSCDetector(eb, getter)

	ctx := context.Background()
	result := detector.PublishNotification(ctx, "pod-unknown", "Title", "Body")

	assert.False(t, result)
}

func TestOSCDetector_PublishTitle_Success(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	getter := &mockPodInfoGetter{orgID: 100, creatorID: 1}
	detector := NewOSCDetector(eb, getter)

	// Subscribe to capture events
	var receivedEvents []*eventbus.Event
	var mu sync.Mutex
	eb.Subscribe(eventbus.EventPodTitleChanged, func(event *eventbus.Event) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
	})

	// Publish title
	ctx := context.Background()
	result := detector.PublishTitle(ctx, "pod-test-123", "My Custom Title")

	assert.True(t, result)
	assert.Equal(t, "My Custom Title", getter.updatedTitle)
}

func TestOSCDetector_PublishTitle_NilEventBus(t *testing.T) {
	getter := &mockPodInfoGetter{orgID: 100, creatorID: 1}
	detector := &OSCDetector{
		eventBus:      nil,
		podInfoGetter: getter,
	}

	ctx := context.Background()
	result := detector.PublishTitle(ctx, "pod-123", "Title")

	assert.False(t, result)
}

func TestOSCDetector_PublishTitle_NilPodInfoGetter(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	detector := &OSCDetector{
		eventBus:      eb,
		podInfoGetter: nil,
	}

	ctx := context.Background()
	result := detector.PublishTitle(ctx, "pod-123", "Title")

	assert.False(t, result)
}

func TestOSCDetector_PublishTitle_PodInfoError(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	getter := &mockPodInfoGetter{err: errors.New("pod not found")}
	detector := NewOSCDetector(eb, getter)

	ctx := context.Background()
	result := detector.PublishTitle(ctx, "pod-unknown", "Title")

	assert.False(t, result)
}

func TestOSCDetector_PublishTitle_UpdateTitleError(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	getter := &mockPodInfoGetter{orgID: 100, creatorID: 1, titleErr: errors.New("db error")}
	detector := NewOSCDetector(eb, getter)

	ctx := context.Background()
	result := detector.PublishTitle(ctx, "pod-123", "Title")

	// Should return true because event is still published (best effort persistence)
	assert.True(t, result)
}

