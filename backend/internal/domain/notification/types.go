package notification

// Source constants for notification categorization
const (
	SourceChannelMessage = "channel:message"
	SourceChannelMention = "channel:mention"
	SourceTerminalOSC    = "terminal:osc"
	SourceTaskCompleted  = "task:completed"
)

// Priority constants
const (
	PriorityNormal = "normal"
	PriorityHigh   = "high"
)

// NotificationRequest is what producers submit to the dispatcher
type NotificationRequest struct {
	OrganizationID int64
	Source         string // e.g. "channel:message", "terminal:osc"
	SourceEntityID string // e.g. "42" (channel ID), "pod-abc"

	// Recipients — mutually exclusive
	RecipientUserIDs  []int64 // Direct list
	RecipientResolver string  // e.g. "channel_members:42", "pod_creator:pod-abc"

	// ExcludeUserIDs filters out these users from resolved recipients (e.g. message sender)
	ExcludeUserIDs []int64

	// Content
	Title string
	Body  string
	Link  string

	// Delivery
	Priority string // "normal" | "high"
}

// Delivery channel constants
const (
	ChannelToast   = "toast"
	ChannelBrowser = "browser"
)

// BuiltinClientChannels lists channels delivered via WebSocket (handled client-side).
var BuiltinClientChannels = map[string]bool{
	ChannelToast:   true,
	ChannelBrowser: true,
}

// Preference represents user notification preferences
type Preference struct {
	IsMuted  bool
	Channels map[string]bool
}

// IsChannelEnabled returns whether a specific delivery channel is enabled.
func (p *Preference) IsChannelEnabled(ch string) bool {
	if p.Channels == nil {
		return false
	}
	return p.Channels[ch]
}

// DefaultPreference returns the default (all enabled) preference
func DefaultPreference() *Preference {
	return &Preference{
		IsMuted:  false,
		Channels: map[string]bool{ChannelToast: true, ChannelBrowser: true},
	}
}
