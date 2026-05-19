package channel

import "time"

type Member struct {
	ChannelID int64     `gorm:"primaryKey" json:"channel_id"`
	UserID    int64     `gorm:"primaryKey" json:"user_id"`
	Role      string    `gorm:"size:20;not null;default:'member'" json:"role"`
	IsMuted   bool      `gorm:"default:false" json:"is_muted"`
	JoinedAt  time.Time `gorm:"default:now()" json:"joined_at"`
}

func (Member) TableName() string { return "channel_members" }
