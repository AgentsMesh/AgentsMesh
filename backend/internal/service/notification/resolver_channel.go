package notification

import (
	"context"
	"strconv"
)

// ChannelMemberProvider looks up channel member user IDs
type ChannelMemberProvider interface {
	GetMemberUserIDs(ctx context.Context, channelID int64) ([]int64, error)
	// GetNonMutedMemberUserIDs returns members that have not muted this channel.
	// Used by notification routing to respect channel-level mute.
	GetNonMutedMemberUserIDs(ctx context.Context, channelID int64) ([]int64, error)
}

// ChannelMemberResolver resolves a channel ID to its non-muted member user IDs
type ChannelMemberResolver struct {
	provider ChannelMemberProvider
}

// NewChannelMemberResolver creates a resolver that returns non-muted channel members
func NewChannelMemberResolver(provider ChannelMemberProvider) *ChannelMemberResolver {
	return &ChannelMemberResolver{provider: provider}
}

func (r *ChannelMemberResolver) Resolve(ctx context.Context, param string) ([]int64, error) {
	channelID, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		return nil, nil // invalid channel ID
	}

	return r.provider.GetNonMutedMemberUserIDs(ctx, channelID)
}
