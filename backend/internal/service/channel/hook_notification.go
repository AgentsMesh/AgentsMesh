package channel

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
)

// NotificationDispatcher is the interface for dispatching notifications.
type NotificationDispatcher interface {
	Dispatch(ctx context.Context, req *notifDomain.NotificationRequest) error
}

// UserNameResolver resolves a user ID to a display name.
type UserNameResolver interface {
	GetUsername(ctx context.Context, userID int64) (string, error)
}

// NewNotificationHook creates a hook that dispatches notifications for new messages.
func NewNotificationHook(dispatcher NotificationDispatcher, userNames UserNameResolver) PostSendHook {
	return func(ctx context.Context, mc *MessageContext) error {
		if dispatcher == nil {
			return nil
		}

		channelIDStr := strconv.FormatInt(mc.Channel.ID, 10)
		channelName := mc.Channel.Name
		senderName := resolveSenderName(ctx, mc, userNames)

		body := mc.Message.Body
		if len(body) > 100 {
			body = body[:97] + "..."
		}

		var excludeIDs []int64
		if mc.Message.SenderUserID != nil {
			excludeIDs = []int64{*mc.Message.SenderUserID}
		}

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
			slog.ErrorContext(ctx, "failed to dispatch channel message notification", "error", err)
		}

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
				slog.ErrorContext(ctx, "failed to dispatch mention notification", "error", err)
			}
		}

		return nil
	}
}

// resolveSenderName extracts a human-readable sender name from the message context.
func resolveSenderName(ctx context.Context, mc *MessageContext, userNames UserNameResolver) string {
	if mc.Message.SenderPod != nil {
		if mc.Message.SenderPodInfo != nil && mc.Message.SenderPodInfo.Alias != nil && *mc.Message.SenderPodInfo.Alias != "" {
			return *mc.Message.SenderPodInfo.Alias
		}
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
