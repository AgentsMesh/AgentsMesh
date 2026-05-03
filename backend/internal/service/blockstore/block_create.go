package blockstoreservice

import (
	"context"
	"fmt"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	"github.com/google/uuid"
)

type createBlockPayload struct {
	ID   *uuid.UUID         `json:"id,omitempty"`
	Type string             `json:"type"`
	Data blockstore.JSONMap `json:"data,omitempty"`
	Text *string            `json:"text,omitempty"`
	Meta blockstore.JSONMap `json:"meta,omitempty"`
}

func (s *Service) applyCreateBlock(
	ctx context.Context,
	tx blockstore.TxWriter,
	actor ActorContext,
	raw map[string]any,
	wsID uuid.UUID,
) (*blockstore.BlockOp, error) {
	p, err := payloadAs[createBlockPayload](raw)
	if err != nil {
		return nil, err
	}
	spec, ok := s.resolveTypeSpecInTx(ctx, tx, p.Type)
	if !ok {
		return nil, blockstore.ErrUnknownBlockType
	}
	if p.Data == nil {
		p.Data = blockstore.JSONMap{}
	}
	if p.Meta == nil {
		p.Meta = blockstore.JSONMap{}
	}
	// Tier 1: a schema-driven type runs full record validation (required +
	// per-column type / options). Legacy types fall back to the old
	// required-key presence check inside the same call.
	if key, reason := spec.ValidateRecord(p.Data); key != "" {
		if reason == "required" {
			return nil, fmt.Errorf("%w: %s", blockstore.ErrMissingRequiredKey, key)
		}
		return nil, fmt.Errorf("%w: %s: %s", blockstore.ErrColumnValueInvalid, key, reason)
	}
	// Trigger-specific invariants (SSRF guard). Sinks here so both the gRPC
	// MCP path and the REST /blocks/ops path share a single validation point
	// — previously the guard lived in the REST MCP dispatcher and a direct
	// /blocks/ops writer would have bypassed it.
	if p.Type == blockstore.BlockTypeTriggerDef {
		if err := validateTriggerDefData(p.Data); err != nil {
			return nil, err
		}
	}

	id := uuid.New()
	if p.ID != nil {
		id = *p.ID
	}
	now := timeNowUTC()
	block := &blockstore.Block{
		ID:          id,
		WorkspaceID: wsID,
		Type:        p.Type,
		Data:        p.Data,
		Text:        p.Text,
		Meta:        p.Meta,
		CreatedBy:   actor.UserID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := tx.InsertBlock(ctx, block); err != nil {
		return nil, err
	}

	forward := blockstore.JSONMap{
		"id":         block.ID,
		"type":       block.Type,
		"data":       block.Data,
		"text":       block.Text,
		"meta":       block.Meta,
		"created_at": block.CreatedAt,
	}
	inverse := blockstore.JSONMap{"id": block.ID}
	target := block.ID
	return &blockstore.BlockOp{
		WorkspaceID: wsID,
		ActorType:   actor.ActorType,
		ActorID:     actor.ActorID,
		Op:          blockstore.OpCreateBlock,
		TargetBlock: &target,
		Payload:     blockstore.JSONMap(raw),
		Forward:     forward,
		Inverse:     inverse,
		Context:     buildOpContext(actor),
		AppliedAt:   now,
	}, nil
}
