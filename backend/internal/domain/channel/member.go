package channel

import "time"

// Member represents a user's membership in a channel
type Member struct {
	ChannelID int64     `gorm:"primaryKey" json:"channel_id"`
	UserID    int64     `gorm:"primaryKey" json:"user_id"`
	IsMuted   bool      `gorm:"default:false" json:"is_muted"`
	JoinedAt  time.Time `gorm:"default:now()" json:"joined_at"`
}

func (Member) TableName() string { return "channel_members" }
