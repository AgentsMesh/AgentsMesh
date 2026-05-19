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
	if key, reason := spec.ValidateRecord(p.Data); key != "" {
		if reason == "required" {
			return nil, fmt.Errorf("%w: %s", blockstore.ErrMissingRequiredKey, key)
		}
		return nil, fmt.Errorf("%w: %s: %s", blockstore.ErrColumnValueInvalid, key, reason)
	}
	// Trigger SSRF guard runs here (not in REST MCP dispatcher) so the gRPC MCP path
	// and direct /blocks/ops both pass through it.
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
