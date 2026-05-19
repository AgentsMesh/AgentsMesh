package blockstore

import (
	"time"

	"github.com/google/uuid"
)

type Block struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	WorkspaceID uuid.UUID  `gorm:"type:uuid;not null;index" json:"workspace_id"`
	Type        string     `gorm:"size:64;not null" json:"type"`
	Data        JSONMap    `gorm:"type:jsonb;not null;default:'{}'" json:"data"`
	Text        *string    `gorm:"type:text" json:"text,omitempty"`
	Meta        JSONMap    `gorm:"type:jsonb;not null;default:'{}'" json:"meta"`
	CreatedBy   int64      `gorm:"not null" json:"created_by"`
	CreatedAt   time.Time  `gorm:"not null;default:current_timestamp" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"not null;default:current_timestamp" json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

func (Block) TableName() string {
	return "blocks"
}
