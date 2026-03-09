package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

// NotifyFunc sends a notification via the notification dispatcher.
// Parameters: ctx, orgID, source, entityID, title, body, link, recipientResolver.
type NotifyFunc func(ctx context.Context, orgID int64, source, entityID, title, body, link, resolver string)

// OSCDetector publishes OSC terminal notification and title events to EventBus.
// OSC sequences are now parsed by Runner and sent as discrete gRPC messages,
// so this component only handles EventBus publishing.
type OSCDetector struct {
	eventBus      *eventbus.EventBus
	podInfoGetter PodInfoGetter
	notifyFunc    NotifyFunc // optional: routes notifications via dispatcher
}

// NewOSCDetector creates a new OSC detector
func NewOSCDetector(eventBus *eventbus.EventBus, podInfoGetter PodInfoGetter) *OSCDetector {
	return &OSCDetector{
		eventBus:      eventBus,
		podInfoGetter: podInfoGetter,
	}
}

// PublishNotification publishes a pre-parsed OSC notification.
// If notifyFunc is set, routes through NotificationDispatcher for preference-aware delivery.
// Otherwise falls back to direct EventBus publish.
func (d *OSCDetector) PublishNotification(ctx context.Context, podKey, title, body string) bool {
	if d.podInfoGetter == nil {
		return false
	}

	orgID, _, err := d.podInfoGetter.GetPodOrganizationAndCreator(ctx, podKey)
	if err != nil {
		return false
	}

	// Prefer dispatcher-based notification (preference-aware, unified format)
	if d.notifyFunc != nil {
		resolver := fmt.Sprintf("pod_creator:%s", podKey)
		link := fmt.Sprintf("/workspace?pod=%s", podKey)
		d.notifyFunc(ctx, orgID, "terminal:osc", podKey, title, body, link, resolver)
		return true
	}

	// No dispatcher configured — log warning and skip (legacy EventBus path removed for format consistency)
	slog.Warn("OSC notification dropped: notifyFunc not configured", "pod_key", podKey)
	return false
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
