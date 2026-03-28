package agent

import (
	"context"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
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

	if err := s.repo.Create(ctx, message); err != nil {
		slog.Error("failed to send agent message", "sender", senderPod, "receiver", receiverPod, "type", messageType, "error", err)
		return nil, err
	}

	slog.Info("agent message sent", "message_id", message.ID, "sender", senderPod, "receiver", receiverPod, "type", messageType)
	return message, nil
}

// GetMessage returns a message by ID
func (s *MessageService) GetMessage(ctx context.Context, messageID int64) (*agent.AgentMessage, error) {
	message, err := s.repo.GetByID(ctx, messageID)
	if err != nil {
		return nil, err
	}
	if message == nil {
		return nil, ErrMessageNotFound
	}
	return message, nil
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
	return s.repo.UpdateStatus(ctx, messageID, map[string]interface{}{
		"status":  agent.MessageStatusRead,
		"read_at": now,
	})
}

// MarkDelivered marks a message as delivered
func (s *MessageService) MarkDelivered(ctx context.Context, messageID int64) error {
	now := time.Now()
	return s.repo.UpdateStatus(ctx, messageID, map[string]interface{}{
		"status":       agent.MessageStatusDelivered,
		"delivered_at": now,
	})
}

// MarkAllRead marks all messages for a pod as read
func (s *MessageService) MarkAllRead(ctx context.Context, podKey string) (int64, error) {
	return s.repo.MarkAllRead(ctx, podKey)
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

	if err := s.repo.Delete(ctx, message); err != nil {
		slog.Error("failed to delete agent message", "message_id", messageID, "pod_key", podKey, "error", err)
		return err
	}
	slog.Info("agent message deleted", "message_id", messageID, "pod_key", podKey)
	return nil
}
