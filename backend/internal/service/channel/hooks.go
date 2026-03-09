package channel

import (
	"context"

	channelDomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

// MentionInput represents a structured mention declaration from the caller.
// The sender explicitly declares who they are mentioning; the server validates, not parses.
type MentionInput struct {
	Type string `json:"type"` // "user" | "pod"
	ID   string `json:"id"`   // user_id (string) or pod_key
}

// MentionResult holds validated @mention data
type MentionResult struct {
	UserIDs []int64  // mentioned user IDs
	PodKeys []string // mentioned pod keys
}

// MessageContext is passed through the PostSendHook pipeline
type MessageContext struct {
	Channel  *channelDomain.Channel
	Message  *channelDomain.Message
	Mentions *MentionResult // populated from structured MentionInput before hooks run
}

// PostSendHook is a function executed after a message is persisted.
// Hooks run sequentially; errors are logged but do not block the message send.
type PostSendHook func(ctx context.Context, mc *MessageContext) error
