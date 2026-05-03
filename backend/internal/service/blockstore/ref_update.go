package blockstoreservice

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/google/uuid"
)

type updateRefPayload struct {
	RefID    int64              `json:"ref_id"`
	From     *uuid.UUID         `json:"from,omitempty"`
	OrderKey *string            `json:"order_key,omitempty"`
	Anchor   *string            `json:"anchor,omitempty"`
	Meta     blockstore.JSONMap `json:"meta,omitempty"`
}

func (s *Service) applyUpdateRef(
	ctx context.Context,
	tx blockstore.TxWriter,
	actor ActorContext,
	raw map[string]any,
	wsID uuid.UUID,
) (*blockstore.BlockOp, error) {
	p, err := payloadAs[updateRefPayload](raw)
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

	fields := map[string]any{}
	forward := blockstore.JSONMap{"ref_id": p.RefID}
	inverse := blockstore.JSONMap{"ref_id": p.RefID}
	if p.From != nil && *p.From != existing.FromID {
		if blockstore.IsUniqueParentRel(existing.Rel) {
			if err := ensureNoCycle(ctx, tx, *p.From, existing.ToID); err != nil {
				return nil, err
			}
			if err := s.ensureChildAllowed(ctx, tx, wsID, *p.From, existing.ToID); err != nil {
				return nil, err
			}
		}
		fields["from_id"] = *p.From
		forward["from"] = *p.From
		inverse["from"] = existing.FromID
	}
	if p.OrderKey != nil {
		fields["order_key"] = p.OrderKey
		forward["order_key"] = p.OrderKey
		inverse["order_key"] = existing.OrderKey
	}
	if p.Anchor != nil {
		fields["anchor"] = p.Anchor
		forward["anchor"] = p.Anchor
		inverse["anchor"] = existing.Anchor
	}
	if p.Meta != nil {
		cloned := p.Meta.Clone()
		fields["meta"] = cloned
		forward["meta"] = cloned
		inverse["meta"] = existing.Meta.Clone()
	}
	if len(fields) > 0 {
		if err := tx.UpdateRefFields(ctx, p.RefID, fields); err != nil {
			return nil, err
		}
	}
	return &blockstore.BlockOp{
		WorkspaceID: wsID,
		ActorType:   actor.ActorType,
		ActorID:     actor.ActorID,
		Op:          blockstore.OpUpdateRef,
		TargetRef:   &p.RefID,
		Payload:     blockstore.JSONMap(raw),
		Forward:     forward,
		Inverse:     inverse,
		Context:     buildOpContext(actor),
		AppliedAt:   timeNowUTC(),
	}, nil
}
