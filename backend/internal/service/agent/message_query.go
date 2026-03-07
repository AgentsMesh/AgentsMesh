package agent

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

// GetMessages returns messages for a pod
func (s *MessageService) GetMessages(ctx context.Context, podKey string, unreadOnly bool, messageTypes []string, limit, offset int) ([]*agent.AgentMessage, error) {
	return s.repo.GetMessages(ctx, podKey, unreadOnly, messageTypes, limit, offset)
}

// GetUnreadMessages returns unread messages for a pod
func (s *MessageService) GetUnreadMessages(ctx context.Context, podKey string, limit int) ([]*agent.AgentMessage, error) {
	return s.repo.GetUnreadMessages(ctx, podKey, limit)
}

// GetUnreadCount returns the count of unread messages for a pod
func (s *MessageService) GetUnreadCount(ctx context.Context, podKey string) (int64, error) {
	return s.repo.GetUnreadCount(ctx, podKey)
}

// GetConversation returns all messages with a correlation ID
func (s *MessageService) GetConversation(ctx context.Context, correlationID string, limit int) ([]*agent.AgentMessage, error) {
	return s.repo.GetConversation(ctx, correlationID, limit)
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
	replies, err := s.repo.GetReplies(ctx, messageID)
	if err != nil {
		return nil, err
	}

	messages = append(messages, replies...)
	return messages, nil
}

// GetSentMessages returns messages sent by a pod
func (s *MessageService) GetSentMessages(ctx context.Context, podKey string, limit, offset int) ([]*agent.AgentMessage, error) {
	return s.repo.GetSentMessages(ctx, podKey, limit, offset)
}

// GetMessagesBetween returns messages between two pods
func (s *MessageService) GetMessagesBetween(ctx context.Context, podA, podB string, limit int) ([]*agent.AgentMessage, error) {
	return s.repo.GetMessagesBetween(ctx, podA, podB, limit)
}
