package runner

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/stretchr/testify/assert"
)

func TestDetectNotifications_OSC777(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected []OSCNotification
	}{
		{
			name:     "no notifications",
			data:     []byte("hello world"),
			expected: nil,
		},
		{
			name:     "OSC 777 with notify keyword",
			data:     []byte("\x1b]777;notify;Build Complete;Your build has finished\x07"),
			expected: []OSCNotification{{Title: "Build Complete", Body: "Your build has finished"}},
		},
		{
			name:     "OSC 777 without notify keyword",
			data:     []byte("\x1b]777;Alert;Something happened\x07"),
			expected: []OSCNotification{{Title: "Alert", Body: "Something happened"}},
		},
		{
			name: "multiple OSC 777 notifications",
			data: []byte("\x1b]777;notify;First;Body1\x07some text\x1b]777;notify;Second;Body2\x07"),
			expected: []OSCNotification{
				{Title: "First", Body: "Body1"},
				{Title: "Second", Body: "Body2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectNotifications(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectNotifications_OSC9(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected []OSCNotification
	}{
		{
			name:     "OSC 9 notification",
			data:     []byte("\x1b]9;Claude Code notification\x07"),
			expected: []OSCNotification{{Title: "Terminal Notification", Body: "Claude Code notification"}},
		},
		{
			name: "multiple OSC 9 notifications",
			data: []byte("\x1b]9;First message\x07\x1b]9;Second message\x07"),
			expected: []OSCNotification{
				{Title: "Terminal Notification", Body: "First message"},
				{Title: "Terminal Notification", Body: "Second message"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectNotifications(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectNotifications_Mixed(t *testing.T) {
	// Test mixed OSC 777 and OSC 9 notifications
	data := []byte("\x1b]777;notify;Build;Done\x07some output\x1b]9;Claude says hi\x07")
	result := DetectNotifications(data)

	assert.Len(t, result, 2)
	assert.Equal(t, "Build", result[0].Title)
	assert.Equal(t, "Done", result[0].Body)
	assert.Equal(t, "Terminal Notification", result[1].Title)
	assert.Equal(t, "Claude says hi", result[1].Body)
}

func TestDetectTitle_OSC0(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected *OSCTitleUpdate
	}{
		{
			name:     "no title sequence",
			data:     []byte("hello world"),
			expected: nil,
		},
		{
			name:     "OSC 0 title",
			data:     []byte("\x1b]0;My Terminal Title\x07"),
			expected: &OSCTitleUpdate{Title: "My Terminal Title"},
		},
		{
			name:     "OSC 0 with surrounding text",
			data:     []byte("some output\x1b]0;Custom Title\x07more output"),
			expected: &OSCTitleUpdate{Title: "Custom Title"},
		},
		{
			name:     "multiple OSC 0 returns last",
			data:     []byte("\x1b]0;First\x07\x1b]0;Second\x07"),
			expected: &OSCTitleUpdate{Title: "Second"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectTitle(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectTitle_OSC2(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected *OSCTitleUpdate
	}{
		{
			name:     "OSC 2 title",
			data:     []byte("\x1b]2;Window Title\x07"),
			expected: &OSCTitleUpdate{Title: "Window Title"},
		},
		{
			name:     "OSC 2 takes precedence over OSC 0",
			data:     []byte("\x1b]0;Icon Title\x07\x1b]2;Window Title\x07"),
			expected: &OSCTitleUpdate{Title: "Window Title"},
		},
		{
			name:     "last title wins when mixed",
			data:     []byte("\x1b]2;Window\x07\x1b]0;Icon\x07"),
			expected: &OSCTitleUpdate{Title: "Window"}, // OSC 2 processed after OSC 0 in code
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectTitle(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectTitle_Empty(t *testing.T) {
	result := DetectTitle([]byte("\x1b]0;\x07"))
	assert.Nil(t, result)
}

func TestOSCDetector_DetectAndPublishTitle_Success(t *testing.T) {
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

	// Detect and publish OSC 0 title
	ctx := context.Background()
	data := []byte("\x1b]0;My Custom Title\x07")
	result := detector.DetectAndPublishTitle(ctx, "pod-test-123", data)

	assert.True(t, result)
	assert.Equal(t, "My Custom Title", getter.updatedTitle)
}

func TestOSCDetector_DetectAndPublishTitle_NoTitle(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	getter := &mockPodInfoGetter{orgID: 100, creatorID: 1}
	detector := NewOSCDetector(eb, getter)

	ctx := context.Background()
	data := []byte("regular terminal output without title")
	result := detector.DetectAndPublishTitle(ctx, "pod-test-123", data)

	assert.False(t, result)
}

func TestOSCDetector_DetectAndPublishTitle_NilEventBus(t *testing.T) {
	getter := &mockPodInfoGetter{orgID: 100, creatorID: 1}
	detector := &OSCDetector{
		eventBus:      nil,
		podInfoGetter: getter,
	}

	ctx := context.Background()
	data := []byte("\x1b]0;test\x07")
	result := detector.DetectAndPublishTitle(ctx, "pod-123", data)

	assert.False(t, result)
}

func TestOSCDetector_DetectAndPublishTitle_NilPodInfoGetter(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	detector := &OSCDetector{
		eventBus:      eb,
		podInfoGetter: nil,
	}

	ctx := context.Background()
	data := []byte("\x1b]0;test\x07")
	result := detector.DetectAndPublishTitle(ctx, "pod-123", data)

	assert.False(t, result)
}

func TestOSCDetector_DetectAndPublishTitle_PodInfoError(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	getter := &mockPodInfoGetter{err: errors.New("pod not found")}
	detector := NewOSCDetector(eb, getter)

	ctx := context.Background()
	data := []byte("\x1b]0;test title\x07")
	result := detector.DetectAndPublishTitle(ctx, "pod-unknown", data)

	assert.False(t, result)
}

func TestOSCDetector_DetectAndPublishTitle_UpdateTitleError(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	getter := &mockPodInfoGetter{orgID: 100, creatorID: 1, titleErr: errors.New("db error")}
	detector := NewOSCDetector(eb, getter)

	ctx := context.Background()
	data := []byte("\x1b]0;test title\x07")
	result := detector.DetectAndPublishTitle(ctx, "pod-123", data)

	// Should return true because event is still published (best effort persistence)
	assert.True(t, result)
}

func TestEscapeJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{`hello "world"`, `hello \"world\"`},
		{"hello\nworld", `hello\nworld`},
		{"hello\tworld", `hello\tworld`},
		{"hello\\world", `hello\\world`},
		{"hello\rworld", `hello\rworld`},
		{`"quoted"`, `\"quoted\"`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeJSON(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestOSCDetector_NilDependencies(t *testing.T) {
	// Test with nil eventBus
	detector := &OSCDetector{
		eventBus:      nil,
		podInfoGetter: nil,
	}

	// Should return 0 without panic
	count := detector.DetectAndPublish(nil, "pod-123", []byte("\x1b]9;test\x07"))
	assert.Equal(t, 0, count)
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

// mockPodInfoGetter implements PodInfoGetter for testing
type mockPodInfoGetter struct {
	orgID       int64
	creatorID   int64
	err         error
	titleErr    error
	updatedTitle string
}

func (m *mockPodInfoGetter) GetPodOrganizationAndCreator(ctx context.Context, podKey string) (orgID, creatorID int64, err error) {
	return m.orgID, m.creatorID, m.err
}

func (m *mockPodInfoGetter) UpdatePodTitle(ctx context.Context, podKey, title string) error {
	m.updatedTitle = title
	return m.titleErr
}

func TestOSCDetector_DetectAndPublish_Success(t *testing.T) {
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

	// Detect and publish OSC 9 notification
	ctx := context.Background()
	data := []byte("\x1b]9;Build completed successfully\x07")
	count := detector.DetectAndPublish(ctx, "pod-test-123", data)

	assert.Equal(t, 1, count)

	// Wait briefly for async event processing
	// Note: Events are published synchronously in tests
}

func TestOSCDetector_DetectAndPublish_NoNotifications(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	getter := &mockPodInfoGetter{orgID: 100, creatorID: 1}
	detector := NewOSCDetector(eb, getter)

	// No OSC sequences in data
	ctx := context.Background()
	data := []byte("regular terminal output without notifications")
	count := detector.DetectAndPublish(ctx, "pod-test-123", data)

	assert.Equal(t, 0, count)
}

func TestOSCDetector_DetectAndPublish_PodInfoError(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	getter := &mockPodInfoGetter{err: errors.New("pod not found")}
	detector := NewOSCDetector(eb, getter)

	// Should return 0 when pod info lookup fails
	ctx := context.Background()
	data := []byte("\x1b]9;test notification\x07")
	count := detector.DetectAndPublish(ctx, "pod-unknown", data)

	assert.Equal(t, 0, count)
}

func TestOSCDetector_DetectAndPublish_MultipleNotifications(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	getter := &mockPodInfoGetter{orgID: 100, creatorID: 1}
	detector := NewOSCDetector(eb, getter)

	// Multiple OSC notifications
	ctx := context.Background()
	data := []byte("\x1b]777;notify;Build;Started\x07 output \x1b]9;Build finished\x07")
	count := detector.DetectAndPublish(ctx, "pod-test-123", data)

	assert.Equal(t, 2, count)
}

func TestOSCDetector_DetectAndPublish_NilEventBus(t *testing.T) {
	getter := &mockPodInfoGetter{orgID: 100, creatorID: 1}
	detector := &OSCDetector{
		eventBus:      nil,
		podInfoGetter: getter,
	}

	ctx := context.Background()
	data := []byte("\x1b]9;test\x07")
	count := detector.DetectAndPublish(ctx, "pod-123", data)

	assert.Equal(t, 0, count)
}

func TestOSCDetector_DetectAndPublish_NilPodInfoGetter(t *testing.T) {
	eb := eventbus.NewEventBus(nil, newTestLogger())
	defer eb.Close()

	detector := &OSCDetector{
		eventBus:      eb,
		podInfoGetter: nil,
	}

	ctx := context.Background()
	data := []byte("\x1b]9;test\x07")
	count := detector.DetectAndPublish(ctx, "pod-123", data)

	assert.Equal(t, 0, count)
}
