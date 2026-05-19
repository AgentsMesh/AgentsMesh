package notification

import (
	"context"
	"strconv"
)

type ChannelMemberProvider interface {
	GetMemberUserIDs(ctx context.Context, channelID int64) ([]int64, error)
	GetNonMutedMemberUserIDs(ctx context.Context, channelID int64) ([]int64, error)
}

type ChannelMemberResolver struct {
	provider ChannelMemberProvider
}

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
