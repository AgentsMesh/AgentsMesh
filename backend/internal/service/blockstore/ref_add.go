package blockstoreservice

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/google/uuid"
)

type addRefPayload struct {
	From     uuid.UUID          `json:"from"`
	To       uuid.UUID          `json:"to"`
	Rel      string             `json:"rel"`
	OrderKey *string            `json:"order_key,omitempty"`
	Anchor   *string            `json:"anchor,omitempty"`
	Meta     blockstore.JSONMap `json:"meta,omitempty"`
}

func (s *Service) applyAddRef(
	ctx context.Context,
	tx blockstore.TxWriter,
	actor ActorContext,
	raw map[string]any,
	wsID uuid.UUID,
) (*blockstore.BlockOp, error) {
	p, err := payloadAs[addRefPayload](raw)
	if err != nil {
		return nil, err
	}
	if p.Rel == "" {
		return nil, blockstore.ErrInvalidRel
	}
	if blockstore.IsOrderedRel(p.Rel) && p.OrderKey == nil {
		return nil, blockstore.ErrOrderKeyRequired
	}
	if err := checkSameWorkspace(ctx, tx, wsID, p.From, p.To); err != nil {
		return nil, err
	}
	if blockstore.IsUniqueParentRel(p.Rel) {
		if existing, err := tx.FindNestParent(ctx, p.To); err != nil {
			return nil, err
		} else if existing != nil {
			return nil, blockstore.ErrSingleNestParent
		}
		if err := ensureNoCycle(ctx, tx, p.From, p.To); err != nil {
			return nil, err
		}
		if err := s.ensureChildAllowed(ctx, tx, wsID, p.From, p.To); err != nil {
			return nil, err
		}
	}

	if p.Meta == nil {
		p.Meta = blockstore.JSONMap{}
	}
	// Clone so later mutations to the payload map (unlikely but cheap to
	// guard against) can't race the stored row / forward diff.
	storedMeta := p.Meta.Clone()
	now := timeNowUTC()
	ref := &blockstore.BlockRef{
		WorkspaceID: wsID,
		FromID:      p.From,
		ToID:        p.To,
		Rel:         p.Rel,
		OrderKey:    p.OrderKey,
		Anchor:      p.Anchor,
		Meta:        storedMeta,
		CreatedBy:   actor.UserID,
		CreatedAt:   now,
	}
	refID, err := tx.InsertRef(ctx, ref)
	if err != nil {
		return nil, err
	}
	forward := blockstore.JSONMap{
		"id":        refID,
		"from":      p.From,
		"to":        p.To,
		"rel":       p.Rel,
		"order_key": p.OrderKey,
		"anchor":    p.Anchor,
		"meta":      storedMeta.Clone(),
	}
	inverse := blockstore.JSONMap{"ref_id": refID}
	return &blockstore.BlockOp{
		WorkspaceID: wsID,
		ActorType:   actor.ActorType,
		ActorID:     actor.ActorID,
		Op:          blockstore.OpAddRef,
		TargetRef:   &refID,
		Payload:     blockstore.JSONMap(raw),
		Forward:     forward,
		Inverse:     inverse,
		Context:     buildOpContext(actor),
		AppliedAt:   now,
	}, nil
}
