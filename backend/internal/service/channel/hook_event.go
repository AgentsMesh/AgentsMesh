package channel

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

type MemberIDProvider interface {
	GetMemberUserIDs(ctx context.Context, channelID int64) ([]int64, error)
}

func NewEventPublishHook(eb *eventbus.EventBus, userNames UserNameResolver, members MemberIDProvider) PostSendHook {
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
			Body:         mc.Message.Body,
			Content:      mc.Message.Content,
			Mentions:     mc.Message.Mentions,
			ReplyTo:      mc.Message.ReplyTo,
			CreatedAt:    mc.Message.CreatedAt.Format(time.RFC3339),
		}

		if mc.Message.SenderPodInfo != nil {
			info := &eventbus.SenderPodInfo{PodKey: mc.Message.SenderPodInfo.PodKey}
			if mc.Message.SenderPodInfo.Alias != nil {
				info.Alias = mc.Message.SenderPodInfo.Alias
			}
			if mc.Message.SenderPodInfo.Agent != nil {
				info.Agent = &eventbus.SenderPodAgent{Name: mc.Message.SenderPodInfo.Agent.Name}
			}
			msgData.SenderPodInfo = info
		}

		data, err := json.Marshal(msgData)
		if err != nil {
			slog.ErrorContext(ctx, "failed to marshal channel message event", "error", err)
			return err
		}

		var targetUserIDs []int64
		if members != nil {
			targetUserIDs, _ = members.GetMemberUserIDs(ctx, mc.Channel.ID)
		}

		eb.Publish(ctx, &eventbus.Event{
			Type:           eventbus.EventChannelMessage,
			Category:       eventbus.CategoryEntity,
			OrganizationID: mc.Channel.OrganizationID,
			EntityType:     "channel",
			EntityID:       strconv.FormatInt(mc.Message.ChannelID, 10),
			Data:           data,
			TargetUserIDs:  targetUserIDs,
		})

		return nil
	}
}
