package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"regexp"

	"github.com/anthropics/agentmesh/backend/internal/infra/eventbus"
)

// OSC (Operating System Command) escape sequence patterns for terminal notifications
var (
	// OSC 777 notification pattern: ESC ] 777 ; notify ; <title> ; <body> BEL
	// Also matches: ESC ] 777 ; <title> ; <body> BEL (without "notify" keyword)
	osc777Pattern = regexp.MustCompile(`\x1b\]777;(?:notify;)?([^;]*);([^\x07]*)\x07`)

	// OSC 9 notification pattern: ESC ] 9 ; <message> BEL (iTerm2/Windows Terminal style)
	// Used by Claude Code for notifications
	osc9Pattern = regexp.MustCompile(`\x1b\]9;([^\x07]*)\x07`)
)

// OSCNotification represents a parsed terminal notification
type OSCNotification struct {
	Title string
	Body  string
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
