package blockstore

import (
	"time"

	"github.com/google/uuid"
)

type BlockWorkspace struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	OrganizationID int64      `gorm:"not null;index" json:"organization_id"`
	Slug           string     `gorm:"size:64;not null" json:"slug"`
	Name           string     `gorm:"size:200;not null" json:"name"`
	RootBlockID    *uuid.UUID `gorm:"type:uuid" json:"root_block_id,omitempty"`
	CreatedBy      int64      `gorm:"not null" json:"created_by"`
	CreatedAt      time.Time  `gorm:"not null;default:current_timestamp" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"not null;default:current_timestamp" json:"updated_at"`
}

func (BlockWorkspace) TableName() string {
	return "block_workspaces"
}

const DefaultWorkspaceSlug = "default"
