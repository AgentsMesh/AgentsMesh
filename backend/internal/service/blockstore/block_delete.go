package blockstoreservice

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/google/uuid"
)

type deleteBlockPayload struct {
	ID uuid.UUID `json:"id"`
}

func (s *Service) applyDeleteBlock(
	ctx context.Context,
	tx blockstore.TxWriter,
	actor ActorContext,
	raw map[string]any,
	wsID uuid.UUID,
) (*blockstore.BlockOp, error) {
	p, err := payloadAs[deleteBlockPayload](raw)
	if err != nil {
		return nil, err
	}
	existing, err := tx.FindBlockByID(ctx, p.ID)
	if err != nil {
		return nil, err
	}
	if existing.WorkspaceID != wsID {
		return nil, blockstore.ErrBlockNotFound
	}
	if !extractACL(existing.Meta).allows(actor.UserID, existing.CreatedBy) {
		return nil, blockstore.ErrBlockForbidden
	}
	if err := tx.SoftDeleteBlock(ctx, p.ID); err != nil {
		return nil, err
	}
	inverse := blockstore.JSONMap{
		"id":   existing.ID,
		"type": existing.Type,
		"data": existing.Data.Clone(),
		"text": existing.Text,
		"meta": existing.Meta.Clone(),
	}
	target := p.ID
	return &blockstore.BlockOp{
		WorkspaceID: wsID,
		ActorType:   actor.ActorType,
		ActorID:     actor.ActorID,
		Op:          blockstore.OpDeleteBlock,
		TargetBlock: &target,
		Payload:     blockstore.JSONMap(raw),
		Forward:     blockstore.JSONMap{"id": p.ID},
		Inverse:     inverse,
		Context:     buildOpContext(actor),
		AppliedAt:   timeNowUTC(),
	}, nil
}
