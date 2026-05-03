package blockstoreservice

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/google/uuid"
)

type removeRefPayload struct {
	RefID int64 `json:"ref_id"`
}

func (s *Service) applyRemoveRef(
	ctx context.Context,
	tx blockstore.TxWriter,
	actor ActorContext,
	raw map[string]any,
	wsID uuid.UUID,
) (*blockstore.BlockOp, error) {
	p, err := payloadAs[removeRefPayload](raw)
	if err != nil {
		return nil, err
	}
	existing, err := tx.FindRefByID(ctx, p.RefID)
	if err != nil {
		return nil, err
	}
	if existing.WorkspaceID != wsID {
		return nil, blockstore.ErrRefNotFound
	}
	if err := tx.DeleteRefByID(ctx, p.RefID); err != nil {
		return nil, err
	}
	inverse := blockstore.JSONMap{
		"from":      existing.FromID,
		"to":        existing.ToID,
		"rel":       existing.Rel,
		"order_key": existing.OrderKey,
		"anchor":    existing.Anchor,
		"meta":      existing.Meta.Clone(),
	}
	return &blockstore.BlockOp{
		WorkspaceID: wsID,
		ActorType:   actor.ActorType,
		ActorID:     actor.ActorID,
		Op:          blockstore.OpRemoveRef,
		TargetRef:   &p.RefID,
		Payload:     blockstore.JSONMap(raw),
		Forward:     blockstore.JSONMap{"ref_id": p.RefID},
		Inverse:     inverse,
		Context:     buildOpContext(actor),
		AppliedAt:   timeNowUTC(),
	}, nil
}
