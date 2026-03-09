package channel

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
)

// NotificationDispatcher is the interface for dispatching notifications.
// Defined here to avoid import cycle with notification service package.
type NotificationDispatcher interface {
	Dispatch(ctx context.Context, req *notifDomain.NotificationRequest) error
}

// UserNameResolver resolves a user ID to a display name.
type UserNameResolver interface {
	GetUsername(ctx context.Context, userID int64) (string, error)
}

// NewNotificationHook creates a hook that dispatches notifications for new messages.
// For regular messages: dispatches to channel_members via resolver, excluding sender.
// For @mentions: dispatches high-priority notifications directly to mentioned users.
func NewNotificationHook(dispatcher NotificationDispatcher, userNames UserNameResolver) PostSendHook {
	return func(ctx context.Context, mc *MessageContext) error {
		if dispatcher == nil {
			return nil
		}

		channelIDStr := strconv.FormatInt(mc.Channel.ID, 10)
		channelName := mc.Channel.Name

		// Build sender display name
		senderName := resolveSenderName(ctx, mc, userNames)

		// Truncate message body for preview
		body := mc.Message.Content
		if len(body) > 100 {
			body = body[:97] + "..."
		}

		// Exclude message sender from receiving their own notification
		var excludeIDs []int64
		if mc.Message.SenderUserID != nil {
			excludeIDs = []int64{*mc.Message.SenderUserID}
		}

		// 1. Regular channel message notification → all members (except sender)
		if err := dispatcher.Dispatch(ctx, &notifDomain.NotificationRequest{
			OrganizationID:    mc.Channel.OrganizationID,
			Source:            notifDomain.SourceChannelMessage,
			SourceEntityID:    channelIDStr,
			RecipientResolver: "channel_members:" + channelIDStr,
			ExcludeUserIDs:    excludeIDs,
			Title:             "#" + channelName,
			Body:              senderName + ": " + body,
			Link:              fmt.Sprintf("/channels?id=%d", mc.Channel.ID),
			Priority:          notifDomain.PriorityNormal,
		}); err != nil {
			slog.Error("failed to dispatch channel message notification", "error", err)
		}

		// 2. @mention notification → directly to mentioned users (high priority, still exclude sender)
		if mc.Mentions != nil && len(mc.Mentions.UserIDs) > 0 {
			if err := dispatcher.Dispatch(ctx, &notifDomain.NotificationRequest{
				OrganizationID:   mc.Channel.OrganizationID,
				Source:           notifDomain.SourceChannelMention,
				SourceEntityID:   channelIDStr,
				RecipientUserIDs: mc.Mentions.UserIDs,
				ExcludeUserIDs:   excludeIDs,
				Title:            fmt.Sprintf("@mention in #%s", channelName),
				Body:             senderName + ": " + body,
				Link:             fmt.Sprintf("/channels?id=%d", mc.Channel.ID),
				Priority:         notifDomain.PriorityHigh,
			}); err != nil {
				slog.Error("failed to dispatch mention notification", "error", err)
			}
		}

		return nil
	}
}

// resolveSenderName extracts a human-readable sender name from the message context.
func resolveSenderName(ctx context.Context, mc *MessageContext, userNames UserNameResolver) string {
	if mc.Message.SenderPod != nil {
		return *mc.Message.SenderPod
	}
	if mc.Message.SenderUserID != nil && userNames != nil {
		if name, err := userNames.GetUsername(ctx, *mc.Message.SenderUserID); err == nil && name != "" {
			return name
		}
	}
	if mc.Message.SenderUserID != nil {
		return fmt.Sprintf("User#%d", *mc.Message.SenderUserID)
	}
	return "System"
}
