package runner

import (
	"context"
	"encoding/json"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

// OSCDetector publishes OSC terminal notification and title events to EventBus.
// OSC sequences are now parsed by Runner and sent as discrete gRPC messages,
// so this component only handles EventBus publishing.
type OSCDetector struct {
	eventBus      *eventbus.EventBus
	podInfoGetter PodInfoGetter
}

// NewOSCDetector creates a new OSC detector
func NewOSCDetector(eventBus *eventbus.EventBus, podInfoGetter PodInfoGetter) *OSCDetector {
	return &OSCDetector{
		eventBus:      eventBus,
		podInfoGetter: podInfoGetter,
	}
}

// PublishNotification publishes a pre-parsed OSC notification to EventBus.
// Called when Runner sends OSC 777 (iTerm2/Kitty) or OSC 9 (ConEmu/Windows Terminal) events.
func (d *OSCDetector) PublishNotification(ctx context.Context, podKey, title, body string) bool {
	if d.eventBus == nil || d.podInfoGetter == nil {
		return false
	}

	// Get pod organization and creator info
	orgID, creatorID, err := d.podInfoGetter.GetPodOrganizationAndCreator(ctx, podKey)
	if err != nil {
		return false
	}

	// Publish notification event — use json.Marshal for safe encoding of user-controlled input
	notifData, _ := json.Marshal(map[string]string{
		"title":   title,
		"body":    body,
		"pod_key": podKey,
	})
	d.eventBus.Publish(ctx, &eventbus.Event{
		Type:           eventbus.EventTerminalNotification,
		Category:       eventbus.CategoryNotification,
		OrganizationID: orgID,
		TargetUserID:   &creatorID,
		EntityType:     "pod",
		EntityID:       podKey,
		Data:           json.RawMessage(notifData),
	})

	return true
}

// PublishTitle publishes a pre-parsed OSC title change to EventBus.
// Called when Runner sends OSC 0/2 (window/tab title) events.
func (d *OSCDetector) PublishTitle(ctx context.Context, podKey, title string) bool {
	if d.eventBus == nil || d.podInfoGetter == nil {
		return false
	}

	// Get pod organization info
	orgID, _, err := d.podInfoGetter.GetPodOrganizationAndCreator(ctx, podKey)
	if err != nil {
		return false
	}

	// Persist title to database
	if err := d.podInfoGetter.UpdatePodTitle(ctx, podKey, title); err != nil {
		// Log error but continue to publish event (best effort persistence)
		// The frontend will still get the update in real-time
	}

	// Publish pod:title_changed event — use json.Marshal for safe encoding
	titleData, _ := json.Marshal(map[string]string{
		"pod_key": podKey,
		"title":   title,
	})
	d.eventBus.Publish(ctx, &eventbus.Event{
		Type:           eventbus.EventPodTitleChanged,
		Category:       eventbus.CategoryEntity,
		OrganizationID: orgID,
		EntityType:     "pod",
		EntityID:       podKey,
		Data:           json.RawMessage(titleData),
	})

	return true
}
