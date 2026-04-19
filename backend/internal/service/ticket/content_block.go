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

// Ticket content lives in Block Store as a type="document" block. This file
// owns the small adapter layer between the ticket REST shape (a single
// BlockNote JSON string) and the block op surface (createBlock / updateBlock
// with the AST as data.blocknote_ast). The block is orphan — it has no nest
// ref — so it doesn't pollute the default workspace's document tree, but it
// still gets op logs, WS broadcast, embeddings, and time travel for free.

// actorForTicketUser builds the Block Store actor for operations performed
// "on behalf of" a ticket's reporter/editor. Reused by every content_block*
// file so the ActorType/ActorID audit tag stays consistent (user origin) and
// ACL checks resolve against the same user that owns the ticket row.
func actorForTicketUser(orgID, userID int64) blockstoreservice.ActorContext {
	return blockstoreservice.ActorContext{
		OrgID:     orgID,
		UserID:    userID,
		ActorType: blockstore.ActorUser,
		ActorID:   userID,
	}
}

// writeContentBlock creates a new document block containing `blocknoteJSON`
// and returns its id. Callers set the returned id on ticket.ContentBlockID.
// Returns uuid.Nil if the content is empty or whitespace-only — tickets with
// no description don't need a backing block.
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

// updateContentBlock replaces the data + text of an existing document block.
// Used when a ticket update carries a new content string. The block's ACL
// and meta are left intact; only data.blocknote_ast and block.text move.
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

// deleteContentBlock is the cascade: when a ticket is deleted, its content
// block is deleted too. No FK, so this cleanup must be explicit. Errors are
// returned to the caller so the ticket service can log but continue —
// dropping an orphan block is a GC concern, not a correctness bug.
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
	// Empty string, or explicit "null" from a client that serialised a nil
	// AST, or an empty BlockNote array — all count as "no content".
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
