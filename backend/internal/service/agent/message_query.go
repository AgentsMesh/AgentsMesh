package agent

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

// GetMessages returns messages for a pod
func (s *MessageService) GetMessages(ctx context.Context, podKey string, unreadOnly bool, messageTypes []string, limit, offset int) ([]*agent.AgentMessage, error) {
	query := s.db.WithContext(ctx).Where("receiver_pod = ?", podKey)

	if unreadOnly {
		query = query.Where("status IN ?", []string{agent.MessageStatusPending, agent.MessageStatusDelivered})
	}

	if len(messageTypes) > 0 {
		query = query.Where("message_type IN ?", messageTypes)
	}

	var messages []*agent.AgentMessage
	if err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error; err != nil {
		return nil, err
	}

	return messages, nil
}

// GetUnreadMessages returns unread messages for a pod
func (s *MessageService) GetUnreadMessages(ctx context.Context, podKey string, limit int) ([]*agent.AgentMessage, error) {
	var messages []*agent.AgentMessage
	if err := s.db.WithContext(ctx).
		Where("receiver_pod = ? AND status IN ?", podKey,
			[]string{agent.MessageStatusPending, agent.MessageStatusDelivered}).
		Order("created_at ASC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// GetUnreadCount returns the count of unread messages for a pod
func (s *MessageService) GetUnreadCount(ctx context.Context, podKey string) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&agent.AgentMessage{}).
		Where("receiver_pod = ? AND status IN ?", podKey,
			[]string{agent.MessageStatusPending, agent.MessageStatusDelivered}).
		Count(&count).Error
	return count, err
}

// GetConversation returns all messages with a correlation ID
func (s *MessageService) GetConversation(ctx context.Context, correlationID string, limit int) ([]*agent.AgentMessage, error) {
	var messages []*agent.AgentMessage
	if err := s.db.WithContext(ctx).
		Where("correlation_id = ?", correlationID).
		Order("created_at ASC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// GetThread returns a message thread (original + all replies)
func (s *MessageService) GetThread(ctx context.Context, messageID int64) ([]*agent.AgentMessage, error) {
	// Get root message
	root, err := s.GetMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}

	messages := []*agent.AgentMessage{root}

	// Get replies
	var replies []*agent.AgentMessage
	if err := s.db.WithContext(ctx).
		Where("parent_message_id = ?", messageID).
		Order("created_at ASC").
		Find(&replies).Error; err != nil {
		return nil, err
	}

	messages = append(messages, replies...)
	return messages, nil
}

// GetSentMessages returns messages sent by a pod
func (s *MessageService) GetSentMessages(ctx context.Context, podKey string, limit, offset int) ([]*agent.AgentMessage, error) {
	var messages []*agent.AgentMessage
	if err := s.db.WithContext(ctx).
		Where("sender_pod = ?", podKey).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// GetMessagesBetween returns messages between two pods
func (s *MessageService) GetMessagesBetween(ctx context.Context, podA, podB string, limit int) ([]*agent.AgentMessage, error) {
	var messages []*agent.AgentMessage
	if err := s.db.WithContext(ctx).
		Where("(sender_pod = ? AND receiver_pod = ?) OR (sender_pod = ? AND receiver_pod = ?)",
			podA, podB, podB, podA).
		Order("created_at ASC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}
