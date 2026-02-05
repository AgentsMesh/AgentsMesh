package agent

import (
	"errors"

	"gorm.io/gorm"
)

var (
	ErrMessageNotFound = errors.New("message not found")
	ErrNotAuthorized   = errors.New("not authorized to access this message")
)

// MessageService handles agent message operations
type MessageService struct {
	db *gorm.DB
}

// NewMessageService creates a new message service
func NewMessageService(db *gorm.DB) *MessageService {
	return &MessageService{db: db}
}
