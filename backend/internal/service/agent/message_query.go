package agent

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

func (s *MessageService) GetMessages(ctx context.Context, podKey string, unreadOnly bool, messageTypes []string, limit, offset int) ([]*agent.AgentMessage, error) {
	return s.repo.GetMessages(ctx, podKey, unreadOnly, messageTypes, limit, offset)
}

func (s *MessageService) GetUnreadMessages(ctx context.Context, podKey string, limit int) ([]*agent.AgentMessage, error) {
	return s.repo.GetUnreadMessages(ctx, podKey, limit)
}

func (s *MessageService) GetUnreadCount(ctx context.Context, podKey string) (int64, error) {
	return s.repo.GetUnreadCount(ctx, podKey)
}

func (s *MessageService) GetConversation(ctx context.Context, correlationID string, limit int) ([]*agent.AgentMessage, error) {
	return s.repo.GetConversation(ctx, correlationID, limit)
}

func (s *MessageService) GetThread(ctx context.Context, messageID int64) ([]*agent.AgentMessage, error) {
	root, err := s.GetMessage(ctx, messageID)
	if err != nil {
		return nil, err
	}

	messages := []*agent.AgentMessage{root}

	replies, err := s.repo.GetReplies(ctx, messageID)
	if err != nil {
		return nil, err
	}

	messages = append(messages, replies...)
	return messages, nil
}

func (s *MessageService) GetSentMessages(ctx context.Context, podKey string, limit, offset int) ([]*agent.AgentMessage, error) {
	return s.repo.GetSentMessages(ctx, podKey, limit, offset)
}

func (s *MessageService) GetMessagesBetween(ctx context.Context, podA, podB string, limit int) ([]*agent.AgentMessage, error) {
	return s.repo.GetMessagesBetween(ctx, podA, podB, limit)
}
