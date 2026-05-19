package blockstoreinfra

import (
	"context"
	"encoding/json"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	"github.com/google/uuid"
)

func init() {
	eventbus.DefaultRegistry.Register(&eventbus.EventDefinition{
		Type:        eventbus.EventBlockstoreOp,
		Category:    eventbus.CategoryEntity,
		EntityType:  "block_workspace",
		Description: "A Block Store op was applied inside a workspace",
	})
}

type OpPublisher struct {
	bus *eventbus.EventBus
}

func NewOpPublisher(bus *eventbus.EventBus) *OpPublisher {
	return &OpPublisher{bus: bus}
}

// OpEnvelope is the payload carried on the event bus. It mirrors the relevant
// subset of block_ops so clients can apply the op locally without a DB round-trip.
type OpEnvelope struct {
	ID             int64      `json:"id"`
	WorkspaceID    uuid.UUID  `json:"workspace_id"`
	IdempotencyKey *string    `json:"idempotency_key,omitempty"`
	ActorType      string     `json:"actor_type"`
	ActorID        int64      `json:"actor_id"`
	Op             string     `json:"op"`
	TargetBlock    *uuid.UUID `json:"target_block,omitempty"`
	TargetRef      *int64     `json:"target_ref,omitempty"`
	Payload        any        `json:"payload"`
	Forward        any        `json:"forward"`
	ParentOpID     *int64     `json:"parent_op_id,omitempty"`
	AppliedAt      int64      `json:"applied_at"` // unix ms
}

func (p *OpPublisher) PublishBatch(ctx context.Context, organizationID int64, ops []*blockstore.BlockOp) {
	if p == nil || p.bus == nil || len(ops) == 0 {
		return
	}
	for _, op := range ops {
		env := OpEnvelope{
			ID:             op.ID,
			WorkspaceID:    op.WorkspaceID,
			IdempotencyKey: op.IdempotencyKey,
			ActorType:      op.ActorType,
			ActorID:        op.ActorID,
			Op:             op.Op,
			TargetBlock:    op.TargetBlock,
			TargetRef:      op.TargetRef,
			Payload:        op.Payload,
			Forward:        op.Forward,
			ParentOpID:     op.ParentOpID,
			AppliedAt:      op.AppliedAt.UnixMilli(),
		}
		raw, err := json.Marshal(env)
		if err != nil {
			continue
		}
		_ = p.bus.Publish(ctx, &eventbus.Event{
			Type:           eventbus.EventBlockstoreOp,
			Category:       eventbus.CategoryEntity,
			OrganizationID: organizationID,
			EntityType:     "block_workspace",
			EntityID:       op.WorkspaceID.String(),
			Data:           raw,
		})
	}
}
