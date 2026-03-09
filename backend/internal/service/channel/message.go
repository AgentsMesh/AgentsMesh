package channel

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

// SendMessage sends a message to a channel.
// mentions is an optional list of structured mention declarations from the caller.
func (s *Service) SendMessage(ctx context.Context, channelID int64, senderPod *string, senderUserID *int64, messageType, content string, metadata channel.MessageMetadata, mentions []MentionInput) (*channel.Message, error) {
	ch, err := s.GetChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}

	if ch.IsArchived {
		return nil, ErrChannelArchived
	}

	// Pre-process structured mentions into metadata before persistence
	var mentionResult *MentionResult
	if len(mentions) > 0 {
		if metadata == nil {
			metadata = make(channel.MessageMetadata)
		}
		var userIDs []int64
		var podKeys []string
		for _, m := range mentions {
			switch m.Type {
			case "user":
				if id, err := strconv.ParseInt(m.ID, 10, 64); err == nil {
					userIDs = append(userIDs, id)
				}
			case "pod":
				podKeys = append(podKeys, m.ID)
			}
		}
		if len(userIDs) > 0 {
			metadata[MetaMentionedUsers] = userIDs
		}
		if len(podKeys) > 0 {
			metadata[MetaMentionedPods] = podKeys
		}
		mentionResult = &MentionResult{UserIDs: userIDs, PodKeys: podKeys}
	}

	msg := &channel.Message{
		ChannelID:    channelID,
		SenderPod:    senderPod,
		SenderUserID: senderUserID,
		MessageType:  messageType,
		Content:      content,
		Metadata:     metadata,
	}

	if err := s.repo.CreateMessage(ctx, msg); err != nil {
		return nil, err
	}

	// Auto-join human sender as channel member (idempotent)
	if senderUserID != nil {
		_ = s.repo.UpsertMember(ctx, channelID, *senderUserID)
	}

	// Update channel updated_at
	_ = s.repo.TouchChannel(ctx, channelID)

	// Run PostSendHooks (mention validation, event publish, notifications, etc.)
	if len(s.postSendHooks) > 0 {
		mc := &MessageContext{Channel: ch, Message: msg, Mentions: mentionResult}
		for _, hook := range s.postSendHooks {
			if err := hook(ctx, mc); err != nil {
				slog.Error("post-send hook failed", "error", err)
			}
		}
	}

	return msg, nil
}

// GetMessages returns messages for a channel
func (s *Service) GetMessages(ctx context.Context, channelID int64, before *time.Time, limit int) ([]*channel.Message, error) {
	messages, err := s.repo.GetMessages(ctx, channelID, before, limit)
	if err != nil {
		return nil, err
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// SendSystemMessage sends a system message to a channel
func (s *Service) SendSystemMessage(ctx context.Context, channelID int64, content string) (*channel.Message, error) {
	return s.SendMessage(ctx, channelID, nil, nil, channel.MessageTypeSystem, content, channel.MessageMetadata{}, nil)
}

// SendMessageAsUser sends a message as a user (human) to a channel
func (s *Service) SendMessageAsUser(ctx context.Context, channelID int64, userID int64, content string, metadata channel.MessageMetadata, mentions []MentionInput) (*channel.Message, error) {
	return s.SendMessage(ctx, channelID, nil, &userID, channel.MessageTypeText, content, metadata, mentions)
}

// SendMessageAsPod sends a message as a pod (agent) to a channel
func (s *Service) SendMessageAsPod(ctx context.Context, channelID int64, podKey string, content string, metadata channel.MessageMetadata, mentions []MentionInput) (*channel.Message, error) {
	return s.SendMessage(ctx, channelID, &podKey, nil, channel.MessageTypeText, content, metadata, mentions)
}

// GetMessagesMentioning returns messages mentioning a specific pod.
// Uses JSONB query on structured metadata with text LIKE fallback for legacy messages.
func (s *Service) GetMessagesMentioning(ctx context.Context, channelID int64, podKey string, limit int) ([]*channel.Message, error) {
	return s.repo.GetMessagesMentioning(ctx, channelID, podKey, limit)
}

// GetMessagesByCursor returns messages before a given message ID (cursor-based pagination)
func (s *Service) GetMessagesByCursor(ctx context.Context, channelID int64, beforeID int64, limit int) ([]*channel.Message, error) {
	messages, err := s.repo.GetMessagesBefore(ctx, channelID, beforeID, limit)
	if err != nil {
		return nil, err
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetRecentMessages returns the most recent messages from a channel
func (s *Service) GetRecentMessages(ctx context.Context, channelID int64, limit int) ([]*channel.Message, error) {
	messages, err := s.repo.GetRecentMessages(ctx, channelID, limit)
	if err != nil {
		return nil, err
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// EditMessage edits a message's content. Only the original sender can edit.
func (s *Service) EditMessage(ctx context.Context, channelID, messageID, senderUserID int64, newContent string) (*channel.Message, error) {
	ch, err := s.GetChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}
	if ch.IsArchived {
		return nil, ErrChannelArchived
	}

	msg, err := s.repo.GetMessageByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if msg == nil || msg.ChannelID != channelID {
		return nil, ErrMessageNotFound
	}
	if msg.SenderUserID == nil || *msg.SenderUserID != senderUserID {
		return nil, ErrNotMessageSender
	}

	if err := s.repo.UpdateMessageContent(ctx, messageID, newContent); err != nil {
		return nil, err
	}

	// Publish edit event
	s.publishMessageEvent(ch.OrganizationID, eventbus.EventChannelMessageEdited, map[string]interface{}{
		"channel_id": channelID,
		"id":         messageID,
		"content":    newContent,
		"edited_at":  time.Now().Format(time.RFC3339),
	})

	return s.repo.GetMessageByID(ctx, messageID)
}

// DeleteMessage soft-deletes a message. Only the original sender can delete.
func (s *Service) DeleteMessage(ctx context.Context, channelID, messageID, senderUserID int64) error {
	ch, err := s.GetChannel(ctx, channelID)
	if err != nil {
		return err
	}
	if ch.IsArchived {
		return ErrChannelArchived
	}

	msg, err := s.repo.GetMessageByID(ctx, messageID)
	if err != nil {
		return err
	}
	if msg == nil || msg.ChannelID != channelID {
		return ErrMessageNotFound
	}
	if msg.SenderUserID == nil || *msg.SenderUserID != senderUserID {
		return ErrNotMessageSender
	}

	if err := s.repo.SoftDeleteMessage(ctx, messageID); err != nil {
		return err
	}

	s.publishMessageEvent(ch.OrganizationID, eventbus.EventChannelMessageDeleted, map[string]interface{}{
		"channel_id": channelID,
		"id":         messageID,
	})

	return nil
}

// publishMessageEvent publishes a channel message event via the event bus
func (s *Service) publishMessageEvent(orgID int64, eventType eventbus.EventType, data map[string]interface{}) {
	if s.eventBus == nil {
		return
	}
	payload, err := json.Marshal(data)
	if err != nil {
		slog.Error("failed to marshal message event", "error", err)
		return
	}
	channelID, _ := data["channel_id"].(int64)
	ctx := context.Background()
	s.eventBus.Publish(ctx, &eventbus.Event{
		Type:           eventType,
		Category:       eventbus.CategoryEntity,
		OrganizationID: orgID,
		EntityType:     "channel",
		EntityID:       fmt.Sprintf("%d", channelID),
		Data:           payload,
		Timestamp:      time.Now().UnixMilli(),
	})
}
