package agent

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"gorm.io/gorm"
)

// SendMessage creates and sends a message from one agent to another
func (s *MessageService) SendMessage(ctx context.Context, senderPod, receiverPod, messageType string, content agent.MessageContent, correlationID *string, parentMessageID *int64) (*agent.AgentMessage, error) {
	message := &agent.AgentMessage{
		SenderPod:       senderPod,
		ReceiverPod:     receiverPod,
		MessageType:     messageType,
		Content:         content,
		Status:          agent.MessageStatusPending,
		CorrelationID:   correlationID,
		ParentMessageID: parentMessageID,
		MaxRetries:      3,
	}

	if err := s.db.WithContext(ctx).Create(message).Error; err != nil {
		return nil, err
	}

	return message, nil
}

// GetMessage returns a message by ID
func (s *MessageService) GetMessage(ctx context.Context, messageID int64) (*agent.AgentMessage, error) {
	var message agent.AgentMessage
	if err := s.db.WithContext(ctx).First(&message, messageID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMessageNotFound
		}
		return nil, err
	}
	return &message, nil
}

// MarkRead marks a message as read
func (s *MessageService) MarkRead(ctx context.Context, messageID int64, podKey string) error {
	message, err := s.GetMessage(ctx, messageID)
	if err != nil {
		return err
	}

	if message.ReceiverPod != podKey {
		return ErrNotAuthorized
	}

	now := time.Now()
	return s.db.WithContext(ctx).Model(message).Updates(map[string]interface{}{
		"status":  agent.MessageStatusRead,
		"read_at": now,
	}).Error
}

// MarkDelivered marks a message as delivered
func (s *MessageService) MarkDelivered(ctx context.Context, messageID int64) error {
	now := time.Now()
	return s.db.WithContext(ctx).Model(&agent.AgentMessage{}).
		Where("id = ?", messageID).
		Updates(map[string]interface{}{
			"status":       agent.MessageStatusDelivered,
			"delivered_at": now,
		}).Error
}

// MarkAllRead marks all messages for a pod as read
func (s *MessageService) MarkAllRead(ctx context.Context, podKey string) (int64, error) {
	now := time.Now()
	result := s.db.WithContext(ctx).Model(&agent.AgentMessage{}).
		Where("receiver_pod = ? AND status IN ?", podKey,
			[]string{agent.MessageStatusPending, agent.MessageStatusDelivered}).
		Updates(map[string]interface{}{
			"status":  agent.MessageStatusRead,
			"read_at": now,
		})
	return result.RowsAffected, result.Error
}

// DeleteMessage soft deletes a message (only sender can delete)
func (s *MessageService) DeleteMessage(ctx context.Context, messageID int64, podKey string) error {
	message, err := s.GetMessage(ctx, messageID)
	if err != nil {
		return err
	}

	if message.SenderPod != podKey {
		return ErrNotAuthorized
	}

	return s.db.WithContext(ctx).Delete(message).Error
}
