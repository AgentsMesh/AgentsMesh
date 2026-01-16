package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"regexp"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

// OSC (Operating System Command) escape sequence patterns for terminal notifications
var (
	// OSC 777 notification pattern: ESC ] 777 ; notify ; <title> ; <body> BEL
	// Also matches: ESC ] 777 ; <title> ; <body> BEL (without "notify" keyword)
	osc777Pattern = regexp.MustCompile(`\x1b\]777;(?:notify;)?([^;]*);([^\x07]*)\x07`)

	// OSC 9 notification pattern: ESC ] 9 ; <message> BEL (iTerm2/Windows Terminal style)
	// Used by Claude Code for notifications
	osc9Pattern = regexp.MustCompile(`\x1b\]9;([^\x07]*)\x07`)

	// OSC 0 title pattern: ESC ] 0 ; <title> BEL (set icon name and window title)
	osc0Pattern = regexp.MustCompile(`\x1b\]0;([^\x07]*)\x07`)

	// OSC 2 title pattern: ESC ] 2 ; <title> BEL (set window title)
	osc2Pattern = regexp.MustCompile(`\x1b\]2;([^\x07]*)\x07`)
)

// OSCNotification represents a parsed terminal notification
type OSCNotification struct {
	Title string
	Body  string
}

// OSCTitleUpdate represents a terminal title update from OSC 0/2
type OSCTitleUpdate struct {
	Title string
}

// OSCDetector detects and publishes OSC 777/9 terminal notification events
type OSCDetector struct {
	eventBus      *eventbus.EventBus
	podInfoGetter PodInfoGetter
}

// NewOSCDetector creates a new OSC notification detector
func NewOSCDetector(eventBus *eventbus.EventBus, podInfoGetter PodInfoGetter) *OSCDetector {
	return &OSCDetector{
		eventBus:      eventBus,
		podInfoGetter: podInfoGetter,
	}
}

// DetectNotifications parses terminal output data for OSC 777/9 notifications
// Returns all detected notifications without publishing events
func DetectNotifications(data []byte) []OSCNotification {
	var notifications []OSCNotification

	// Find OSC 777 matches (title ; body format)
	matches777 := osc777Pattern.FindAllSubmatch(data, -1)
	for _, match := range matches777 {
		if len(match) >= 3 {
			notifications = append(notifications, OSCNotification{
				Title: string(match[1]),
				Body:  string(match[2]),
			})
		}
	}

	// Find OSC 9 matches (single message format, used by Claude Code)
	matches9 := osc9Pattern.FindAllSubmatch(data, -1)
	for _, match := range matches9 {
		if len(match) >= 2 {
			notifications = append(notifications, OSCNotification{
				Title: "Terminal Notification",
				Body:  string(match[1]),
			})
		}
	}

	return notifications
}

// DetectTitle parses terminal output data for OSC 0/2 window title sequences
// Returns the last detected title (OSC 2 takes precedence over OSC 0)
func DetectTitle(data []byte) *OSCTitleUpdate {
	var title string

	// Check OSC 0 (icon name and window title)
	matches0 := osc0Pattern.FindAllSubmatch(data, -1)
	if len(matches0) > 0 {
		lastMatch := matches0[len(matches0)-1]
		if len(lastMatch) >= 2 {
			title = string(lastMatch[1])
		}
	}

	// Check OSC 2 (window title only) - takes precedence
	matches2 := osc2Pattern.FindAllSubmatch(data, -1)
	if len(matches2) > 0 {
		lastMatch := matches2[len(matches2)-1]
		if len(lastMatch) >= 2 {
			title = string(lastMatch[1])
		}
	}

	if title == "" {
		return nil
	}

	return &OSCTitleUpdate{Title: title}
}

// DetectAndPublish detects OSC notifications and publishes events to EventBus
// Returns the number of notifications published
func (d *OSCDetector) DetectAndPublish(ctx context.Context, podKey string, data []byte) int {
	if d.eventBus == nil || d.podInfoGetter == nil {
		return 0
	}

	notifications := DetectNotifications(data)
	if len(notifications) == 0 {
		return 0
	}

	// Get pod organization and creator info
	orgID, creatorID, err := d.podInfoGetter.GetPodOrganizationAndCreator(ctx, podKey)
	if err != nil {
		return 0
	}

	// Publish each notification
	for _, n := range notifications {
		d.eventBus.Publish(ctx, &eventbus.Event{
			Type:           eventbus.EventTerminalNotification,
			Category:       eventbus.CategoryNotification,
			OrganizationID: orgID,
			TargetUserID:   &creatorID,
			EntityType:     "pod",
			EntityID:       podKey,
			Data: json.RawMessage(`{
				"title": "` + escapeJSON(n.Title) + `",
				"body": "` + escapeJSON(n.Body) + `",
				"pod_key": "` + podKey + `"
			}`),
		})
	}

	return len(notifications)
}

// DetectAndPublishTitle detects OSC 0/2 title changes, persists to database, and publishes events to EventBus
// Returns true if a title change was detected and published
func (d *OSCDetector) DetectAndPublishTitle(ctx context.Context, podKey string, data []byte) bool {
	if d.eventBus == nil || d.podInfoGetter == nil {
		return false
	}

	titleUpdate := DetectTitle(data)
	if titleUpdate == nil {
		return false
	}

	// Get pod organization info
	orgID, _, err := d.podInfoGetter.GetPodOrganizationAndCreator(ctx, podKey)
	if err != nil {
		return false
	}

	// Persist title to database
	if err := d.podInfoGetter.UpdatePodTitle(ctx, podKey, titleUpdate.Title); err != nil {
		// Log error but continue to publish event (best effort persistence)
		// The frontend will still get the update in real-time
	}

	// Publish pod:title_changed event
	d.eventBus.Publish(ctx, &eventbus.Event{
		Type:           eventbus.EventPodTitleChanged,
		Category:       eventbus.CategoryEntity,
		OrganizationID: orgID,
		EntityType:     "pod",
		EntityID:       podKey,
		Data: json.RawMessage(`{
			"pod_key": "` + podKey + `",
			"title": "` + escapeJSON(titleUpdate.Title) + `"
		}`),
	})

	return true
}

// escapeJSON escapes special characters in JSON string values
func escapeJSON(s string) string {
	var result bytes.Buffer
	for _, r := range s {
		switch r {
		case '"':
			result.WriteString(`\"`)
		case '\\':
			result.WriteString(`\\`)
		case '\n':
			result.WriteString(`\n`)
		case '\r':
			result.WriteString(`\r`)
		case '\t':
			result.WriteString(`\t`)
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}
