package ticket

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	bnpkg "github.com/anthropics/agentsmesh/backend/pkg/blocknote"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	blockstoreservice "github.com/anthropics/agentsmesh/backend/internal/service/blockstore"
	"github.com/google/uuid"
)

func actorForTicketUser(orgID, userID int64) blockstoreservice.ActorContext {
	return blockstoreservice.ActorContext{
		OrgID:     orgID,
		UserID:    userID,
		ActorType: blockstore.ActorUser,
		ActorID:   userID,
	}
}

func (s *Service) writeContentBlock(
	ctx context.Context,
	orgID, userID int64,
	blocknoteJSON string,
) (uuid.UUID, error) {
	if s.blockstore == nil {
		return uuid.Nil, fmt.Errorf("blockstore not configured")
	}
	if !hasRichContent(blocknoteJSON) {
		return uuid.Nil, nil
	}
	actor := actorForTicketUser(orgID, userID)
	ws, err := s.blockstore.EnsureDefaultWorkspace(ctx, actor)
	if err != nil {
		return uuid.Nil, fmt.Errorf("ensure default workspace: %w", err)
	}
	ast, err := parseBlocknote(blocknoteJSON)
	if err != nil {
		return uuid.Nil, err
	}
	newID := uuid.New()
	plain := bnpkg.ToPlainText(blocknoteJSON)
	_, err = s.blockstore.ApplyOps(ctx, actor, blockstoreservice.ApplyOpsInput{
		WorkspaceID:    ws.ID.String(),
		IdempotencyKey: fmt.Sprintf("ticket-content-create-%s", newID),
		Ops: []blockstoreservice.OpEnvelope{
			{Op: blockstore.OpCreateBlock, Payload: map[string]any{
				"id":   newID.String(),
				"type": blockstore.BlockTypeDocument,
				"data": map[string]any{"blocknote_ast": ast},
				"text": plain,
			}},
		},
	})
	if err != nil {
		return uuid.Nil, fmt.Errorf("create content block: %w", err)
	}
	return newID, nil
}

func (s *Service) updateContentBlock(
	ctx context.Context,
	orgID, userID int64,
	blockID uuid.UUID,
	blocknoteJSON string,
) error {
	if s.blockstore == nil {
		return fmt.Errorf("blockstore not configured")
	}
	actor := actorForTicketUser(orgID, userID)
	ws, err := s.blockstore.EnsureDefaultWorkspace(ctx, actor)
	if err != nil {
		return fmt.Errorf("ensure default workspace: %w", err)
	}
	ast, err := parseBlocknote(blocknoteJSON)
	if err != nil {
		return err
	}
	plain := bnpkg.ToPlainText(blocknoteJSON)
	_, err = s.blockstore.ApplyOps(ctx, actor, blockstoreservice.ApplyOpsInput{
		WorkspaceID:    ws.ID.String(),
		IdempotencyKey: fmt.Sprintf("ticket-content-update-%s-%d", blockID, nowUnixNano()),
		Ops: []blockstoreservice.OpEnvelope{
			{Op: blockstore.OpUpdateBlock, Payload: map[string]any{
				"id":   blockID.String(),
				"data": map[string]any{"blocknote_ast": ast},
				"text": plain,
			}},
		},
	})
	if err != nil {
		return fmt.Errorf("update content block: %w", err)
	}
	return nil
}

func (s *Service) deleteContentBlock(
	ctx context.Context,
	orgID, userID int64,
	blockID uuid.UUID,
) error {
	if s.blockstore == nil {
		return nil
	}
	actor := actorForTicketUser(orgID, userID)
	ws, err := s.blockstore.EnsureDefaultWorkspace(ctx, actor)
	if err != nil {
		return err
	}
	_, err = s.blockstore.ApplyOps(ctx, actor, blockstoreservice.ApplyOpsInput{
		WorkspaceID:    ws.ID.String(),
		IdempotencyKey: fmt.Sprintf("ticket-content-delete-%s", blockID),
		Ops: []blockstoreservice.OpEnvelope{
			{Op: blockstore.OpDeleteBlock, Payload: map[string]any{
				"id": blockID.String(),
			}},
		},
	})
	return err
}

func hasRichContent(s string) bool {
	if len(s) == 0 || s == "null" || s == "[]" {
		return false
	}
	return true
}

func parseBlocknote(s string) (any, error) {
	var ast any
	if err := json.Unmarshal([]byte(s), &ast); err != nil {
		return nil, fmt.Errorf("invalid blocknote JSON: %w", err)
	}
	return ast, nil
}

func nowUnixNano() int64 { return time.Now().UnixNano() }
