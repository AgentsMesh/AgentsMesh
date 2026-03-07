package agent

import (
	"errors"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

var (
	ErrMessageNotFound = errors.New("message not found")
	ErrNotAuthorized   = errors.New("not authorized to access this message")
)

// MessageService handles agent message operations
type MessageService struct {
	repo agent.MessageRepository
}

// NewMessageService creates a new message service
func NewMessageService(repo agent.MessageRepository) *MessageService {
	return &MessageService{repo: repo}
}
