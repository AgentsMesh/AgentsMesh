package channel

import "time"

// ReadState tracks a user's last read position in a channel
type ReadState struct {
	ChannelID         int64     `gorm:"primaryKey" json:"channel_id"`
	UserID            int64     `gorm:"primaryKey" json:"user_id"`
	LastReadMessageID *int64    `json:"last_read_message_id"`
	LastReadAt        time.Time `gorm:"default:now()" json:"last_read_at"`
}

func (ReadState) TableName() string { return "channel_read_states" }
