package channel

// ChannelSummary is a member's per-channel read state in a single row: the
// unread message count, how many of those unread messages @-mention the member
// (directly or via @channel), and the monotonic last-read cursor (0 = nothing
// read yet — used to anchor the client's "new messages" divider on entry).
type ChannelSummary struct {
	Unread         int64
	Mention        int64
	LastRead       int64
	ManuallyUnread bool
}
