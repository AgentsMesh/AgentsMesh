package channel

import (
	"context"

	channelDomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

// MentionResult holds validated @mention data extracted from structured content.
type MentionResult struct {
	UserIDs []int64
	PodKeys []string
}

// MessageContext is passed through the PostSendHook pipeline.
type MessageContext struct {
	Channel  *channelDomain.Channel
	Message  *channelDomain.Message
	Mentions *MentionResult
}

// PostSendHook is a function executed after a message is persisted.
// Hooks run sequentially; errors are logged but do not block the message send.
type PostSendHook func(ctx context.Context, mc *MessageContext) error
