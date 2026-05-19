package v1

import (
	agentSvc "github.com/anthropics/agentsmesh/backend/internal/service/agent"
)

type MessageHandler struct {
	msgSvc *agentSvc.MessageService
}

func NewMessageHandler(msgSvc *agentSvc.MessageService) *MessageHandler {
	return &MessageHandler{
		msgSvc: msgSvc,
	}
}

type AgentSendMessageRequest struct {
	ReceiverPod   string                 `json:"receiver_pod" binding:"required"`
	MessageType   string                 `json:"message_type" binding:"required"`
	Content       map[string]interface{} `json:"content" binding:"required"`
	CorrelationID *string                `json:"correlation_id,omitempty"`
	ReplyToID     *int64                 `json:"reply_to_id,omitempty"`
}

type MarkReadRequest struct {
	MessageIDs []int64 `json:"message_ids" binding:"required"`
}
