package channel

import (
	"context"

	channelDomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

type MentionResult struct {
	UserIDs []int64
	PodKeys []string
}

type MessageContext struct {
	Channel  *channelDomain.Channel
	Message  *channelDomain.Message
	Mentions *MentionResult
}

type PostSendHook func(ctx context.Context, mc *MessageContext) error
