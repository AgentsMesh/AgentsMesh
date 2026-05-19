package notification

const (
	SourceChannelMessage = "channel:message"
	SourceChannelMention = "channel:mention"
	SourceTerminalOSC    = "terminal:osc"
	SourceTaskCompleted  = "task:completed"
)

const (
	PriorityNormal = "normal"
	PriorityHigh   = "high"
)

type NotificationRequest struct {
	OrganizationID int64
	Source         string // e.g. "channel:message", "terminal:osc"
	SourceEntityID string // e.g. "42" (channel ID), "pod-abc"

	RecipientUserIDs  []int64 // Direct list
	RecipientResolver string  // e.g. "channel_members:42", "pod_creator:pod-abc"

	ExcludeUserIDs []int64

	Title string
	Body  string
	Link  string

	Priority string // "normal" | "high"
}

const (
	ChannelToast   = "toast"
	ChannelBrowser = "browser"
)

var BuiltinClientChannels = map[string]bool{
	ChannelToast:   true,
	ChannelBrowser: true,
}

type Preference struct {
	IsMuted  bool
	Channels map[string]bool
}

func (p *Preference) IsChannelEnabled(ch string) bool {
	if p.Channels == nil {
		return false
	}
	return p.Channels[ch]
}

func DefaultPreference() *Preference {
	return &Preference{
		IsMuted:  false,
		Channels: map[string]bool{ChannelToast: true, ChannelBrowser: true},
	}
}
