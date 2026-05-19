package blockstoreservice

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/google/uuid"
)

type updateBlockPayload struct {
	ID                uuid.UUID          `json:"id"`
	Data              blockstore.JSONMap `json:"data,omitempty"`
	Text              *string            `json:"text,omitempty"`
	Meta              blockstore.JSONMap `json:"meta,omitempty"`
	ExpectedUpdatedAt *time.Time         `json:"expected_updated_at,omitempty"`
}

func (s *Service) applyUpdateBlock(
	ctx context.Context,
	tx blockstore.TxWriter,
	actor ActorContext,
	raw map[string]any,
	wsID uuid.UUID,
) (*blockstore.BlockOp, error) {
	p, err := payloadAs[updateBlockPayload](raw)
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
	if p.ExpectedUpdatedAt != nil && !existing.UpdatedAt.Equal(*p.ExpectedUpdatedAt) {
		return nil, blockstore.ErrStaleUpdate
	}
	// SSRF guard runs on update too — an edit that swaps in a private webhook URL must reject.
	if existing.Type == blockstore.BlockTypeTriggerDef && p.Data != nil {
		if err := validateTriggerDefData(p.Data); err != nil {
			return nil, err
		}
	}

	fields := map[string]any{}
	inverse := blockstore.JSONMap{"id": p.ID}
	forward := blockstore.JSONMap{"id": p.ID}
	if p.Data != nil {
		fields["data"] = p.Data
		inverse["data"] = existing.Data.Clone()
		forward["data"] = p.Data
	}
	if p.Text != nil {
		fields["text"] = p.Text
		inverse["text"] = existing.Text
		forward["text"] = p.Text
	}
	if p.Meta != nil {
		fields["meta"] = p.Meta
		inverse["meta"] = existing.Meta.Clone()
		forward["meta"] = p.Meta
	}
	now := timeNowUTC()
	if len(fields) > 0 {
		if err := tx.UpdateBlockFields(ctx, p.ID, fields); err != nil {
			return nil, err
		}
	}
	target := p.ID
	return &blockstore.BlockOp{
		WorkspaceID: wsID,
		ActorType:   actor.ActorType,
		ActorID:     actor.ActorID,
		Op:          blockstore.OpUpdateBlock,
		TargetBlock: &target,
		Payload:     blockstore.JSONMap(raw),
		Forward:     forward,
		Inverse:     inverse,
		Context:     buildOpContext(actor),
		AppliedAt:   now,
	}, nil
}
