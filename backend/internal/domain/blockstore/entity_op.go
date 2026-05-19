package blockstore

import (
	"time"

	"github.com/google/uuid"
)

const (
	ActorUser   = "user"
	ActorAgent  = "agent"
	ActorSystem = "system"
)

// BlockOp is the single source of truth for collaboration, audit, and undo.
// Every mutation against blocks / block_refs produces exactly one BlockOp row.
// Streaming clients consume this table ordered by (workspace_id, id).
//
// Invariant: exactly one of (TargetBlock, TargetRef) is set. Enforced by a
// DB-level CHECK constraint (see migration 000115) and by the service layer.
//
// The Context JSONB is a forward-compatibility slot for audit metadata that
// doesn't belong in the op semantic payload (request_id, ip, user_agent,
// trace_id, ...). Keeping it separate from Payload means read-consumers of
// the collab protocol can keep ignoring it while auditors can read from it
// without a schema migration.
type BlockOp struct {
	ID             int64      `gorm:"primaryKey" json:"id"`
	WorkspaceID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"workspace_id"`
	IdempotencyKey *string    `gorm:"size:128;uniqueIndex" json:"idempotency_key,omitempty"`
	ActorType      string     `gorm:"size:16;not null" json:"actor_type"`
	ActorID        int64      `gorm:"not null" json:"actor_id"`
	Op             string     `gorm:"size:32;not null" json:"op"`
	TargetBlock    *uuid.UUID `gorm:"type:uuid" json:"target_block,omitempty"`
	TargetRef      *int64     `json:"target_ref,omitempty"`
	Payload        JSONMap    `gorm:"type:jsonb;not null" json:"payload"`
	Forward        JSONMap    `gorm:"type:jsonb;not null" json:"forward"`
	Inverse        JSONMap    `gorm:"type:jsonb;not null" json:"inverse"`
	Context        JSONMap    `gorm:"type:jsonb;default:'{}'" json:"context,omitempty"`
	ParentOpID     *int64     `json:"parent_op_id,omitempty"`
	AppliedAt      time.Time  `gorm:"not null;default:current_timestamp" json:"applied_at"`
}

func (BlockOp) TableName() string {
	return "block_ops"
}
