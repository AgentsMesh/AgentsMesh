package channel

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

// NewEventPublishHook creates a hook that publishes channel:message events.
// userNames is optional — when provided the sender display name is resolved
// and included in the event payload so that realtime consumers can render it
// without a separate lookup.
func NewEventPublishHook(eb *eventbus.EventBus, userNames UserNameResolver) PostSendHook {
	return func(ctx context.Context, mc *MessageContext) error {
		if eb == nil {
			return nil
		}

		msgData := eventbus.ChannelMessageData{
			ID:           mc.Message.ID,
			ChannelID:    mc.Message.ChannelID,
			SenderPod:    mc.Message.SenderPod,
			SenderUserID: mc.Message.SenderUserID,
			SenderName:   resolveSenderName(ctx, mc, userNames),
			MessageType:  mc.Message.MessageType,
			Content:      mc.Message.Content,
			Metadata:     mc.Message.Metadata,
			CreatedAt:    mc.Message.CreatedAt.Format(time.RFC3339),
		}
		data, err := json.Marshal(msgData)
		if err != nil {
			slog.Error("failed to marshal channel message event", "error", err)
			return err
		}

		eb.Publish(ctx, &eventbus.Event{
			Type:           eventbus.EventChannelMessage,
			Category:       eventbus.CategoryEntity,
			OrganizationID: mc.Channel.OrganizationID,
			EntityType:     "channel",
			EntityID:       strconv.FormatInt(mc.Message.ChannelID, 10),
			Data:           data,
		})

		return nil
	}
}
