package channel

import "time"

type MessageEdit struct {
	ID              int64           `gorm:"primaryKey" json:"id"`
	MessageID       int64           `gorm:"not null;index" json:"message_id"`
	EditorUserID    *int64          `json:"editor_user_id,omitempty"`
	EditorPod       *string         `gorm:"size:100" json:"editor_pod,omitempty"`
	PreviousBody    string          `gorm:"type:text;not null" json:"previous_body"`
	PreviousContent *MessageContent `gorm:"type:jsonb" json:"previous_content,omitempty"`
	CreatedAt       time.Time       `gorm:"not null;default:now()" json:"created_at"`
}

func (MessageEdit) TableName() string {
	return "channel_message_edits"
}
