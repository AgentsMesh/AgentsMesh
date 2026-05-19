package agent

import (
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

var (
	ErrMessageNotFound = errors.New("message not found")
	ErrNotAuthorized   = errors.New("not authorized to access this message")
)

type MessageService struct {
	repo agent.MessageRepository
}

func NewMessageService(repo agent.MessageRepository) *MessageService {
	return &MessageService{repo: repo}
}
