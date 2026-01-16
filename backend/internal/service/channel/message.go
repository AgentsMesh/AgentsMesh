package channel

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
)

// SendMessage sends a message to a channel
func (s *Service) SendMessage(ctx context.Context, channelID int64, senderPod *string, senderUserID *int64, messageType, content string, metadata channel.MessageMetadata) (*channel.Message, error) {
	ch, err := s.GetChannel(ctx, channelID)
	if err != nil {
		return nil, err
	}

	if ch.IsArchived {
		return nil, ErrChannelArchived
	}

	msg := &channel.Message{
		ChannelID:    channelID,
		SenderPod:    senderPod,
		SenderUserID: senderUserID,
		MessageType:  messageType,
		Content:      content,
		Metadata:     metadata,
	}

	if err := s.db.WithContext(ctx).Create(msg).Error; err != nil {
		return nil, err
	}

	// Update channel updated_at
	s.db.WithContext(ctx).Model(ch).Update("updated_at", time.Now())

	// Publish channel:message event via EventBus
	if s.eventBus != nil {
		msgData := eventbus.ChannelMessageData{
			ID:           msg.ID,
			ChannelID:    msg.ChannelID,
			SenderPod:    msg.SenderPod,
			SenderUserID: msg.SenderUserID,
			MessageType:  msg.MessageType,
			Content:      msg.Content,
			Metadata:     msg.Metadata,
			CreatedAt:    msg.CreatedAt.Format(time.RFC3339),
		}
		data, _ := json.Marshal(msgData)
		s.eventBus.Publish(ctx, &eventbus.Event{
			Type:           eventbus.EventChannelMessage,
			Category:       eventbus.CategoryEntity,
			OrganizationID: ch.OrganizationID,
			EntityType:     "channel",
			EntityID:       strconv.FormatInt(channelID, 10),
			Data:           data,
		})
	}

	return msg, nil
}

// GetMessages returns messages for a channel
func (s *Service) GetMessages(ctx context.Context, channelID int64, before *time.Time, limit int) ([]*channel.Message, error) {
	query := s.db.WithContext(ctx).Where("channel_id = ?", channelID)

	if before != nil {
		query = query.Where("created_at < ?", *before)
	}

	var messages []*channel.Message
	if err := query.
		Preload("SenderUser").
		Preload("SenderPodInfo").
		Preload("SenderPodInfo.AgentType").
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
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
	return s.SendMessage(ctx, channelID, nil, nil, channel.MessageTypeSystem, content, channel.MessageMetadata{})
}

// SendMessageAsUser sends a message as a user (human) to a channel
func (s *Service) SendMessageAsUser(ctx context.Context, channelID int64, userID int64, content string, metadata channel.MessageMetadata) (*channel.Message, error) {
	return s.SendMessage(ctx, channelID, nil, &userID, channel.MessageTypeText, content, metadata)
}

// SendMessageAsPod sends a message as a pod (agent) to a channel
func (s *Service) SendMessageAsPod(ctx context.Context, channelID int64, podKey string, content string, metadata channel.MessageMetadata) (*channel.Message, error) {
	return s.SendMessage(ctx, channelID, &podKey, nil, channel.MessageTypeText, content, metadata)
}

// GetMessagesMentioning returns messages mentioning a specific pod
func (s *Service) GetMessagesMentioning(ctx context.Context, channelID int64, podKey string, limit int) ([]*channel.Message, error) {
	var messages []*channel.Message
	pattern := "@" + podKey
	if err := s.db.WithContext(ctx).
		Where("channel_id = ? AND content LIKE ?", channelID, "%"+pattern+"%").
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// GetRecentMessages returns the most recent messages from a channel
func (s *Service) GetRecentMessages(ctx context.Context, channelID int64, limit int) ([]*channel.Message, error) {
	var messages []*channel.Message
	if err := s.db.WithContext(ctx).
		Where("channel_id = ?", channelID).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}

	// Reverse to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
