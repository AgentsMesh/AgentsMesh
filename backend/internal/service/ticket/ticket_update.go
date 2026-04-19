package ticket

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	domainTicket "github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
	"github.com/google/uuid"
)

// UpdateTicket updates a ticket.
func (s *Service) UpdateTicket(ctx context.Context, ticketID int64, updates map[string]interface{}) (*domainTicket.Ticket, error) {
	oldTicket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	previousStatus := oldTicket.Status

	// Intercept any content update so it flows through Block Store rather
	// than the legacy tickets.content column. The handler still accepts a
	// content string (REST shape unchanged), but we route it to
	// createBlock/updateBlock and replace the field in `updates` with
	// content_block_id before hitting the ticket repo.
	if rawContent, has := updates["content"]; has && s.blockstore != nil {
		contentStr, ok := rawContent.(string)
		if !ok && rawContent != nil {
			// non-string content from a caller — surface clearly rather than
			// silently dropping the update.
			return nil, fmt.Errorf("content must be a string, got %T", rawContent)
		}
		delete(updates, "content")
		blockID, err := s.syncContentBlockForUpdate(ctx, oldTicket, contentStr)
		if err != nil {
			return nil, err
		}
		updates["content_block_id"] = blockID
		// Clear the legacy column so a stale inline copy doesn't shadow the
		// block-backed value during the transition period.
		updates["content"] = nil
	}

	if len(updates) > 0 {
		if err := s.repo.UpdateFields(ctx, ticketID, updates); err != nil {
			slog.ErrorContext(ctx, "failed to update ticket fields", "ticket_id", ticketID, "org_id", oldTicket.OrganizationID, "error", err)
			return nil, err
		}
	}

	updatedTicket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	if newStatus, ok := updates["status"].(string); ok && newStatus != previousStatus {
		slog.InfoContext(ctx, "ticket status changed", "ticket_id", ticketID, "slug", updatedTicket.Slug, "from", previousStatus, "to", newStatus)
		s.publishEvent(ctx, TicketEventStatusChanged, oldTicket.OrganizationID, updatedTicket.Slug, updatedTicket.Status, previousStatus)
	} else {
		s.publishEvent(ctx, TicketEventUpdated, oldTicket.OrganizationID, updatedTicket.Slug, updatedTicket.Status, previousStatus)
	}

	return updatedTicket, nil
}

// syncContentBlockForUpdate keeps the ticket's content block in sync with
// the incoming string: creates a new document block if none exists, updates
// the existing one if the ticket already has a backing block, or deletes
// the old block and clears the pointer when content goes empty. Returns
// the block id to stamp on the ticket row, or *uuid.Nil-typed nil when
// there's no block.
func (s *Service) syncContentBlockForUpdate(
	ctx context.Context,
	oldTicket *domainTicket.Ticket,
	newContent string,
) (*uuid.UUID, error) {
	if oldTicket.ContentBlockID != nil {
		if !hasRichContent(newContent) {
			// Content cleared — drop the block and null out the pointer.
			if err := s.deleteContentBlock(ctx, oldTicket.OrganizationID, oldTicket.ReporterID, *oldTicket.ContentBlockID); err != nil {
				slog.WarnContext(ctx, "ticket content block delete failed", "block_id", *oldTicket.ContentBlockID, "err", err)
			}
			return nil, nil
		}
		if err := s.updateContentBlock(ctx, oldTicket.OrganizationID, oldTicket.ReporterID, *oldTicket.ContentBlockID, newContent); err != nil {
			return nil, err
		}
		return oldTicket.ContentBlockID, nil
	}
	// No existing block. Create one only if the payload is non-empty.
	if !hasRichContent(newContent) {
		return nil, nil
	}
	id, err := s.writeContentBlock(ctx, oldTicket.OrganizationID, oldTicket.ReporterID, newContent)
	if err != nil {
		return nil, err
	}
	if id == uuid.Nil {
		return nil, nil
	}
	return &id, nil
}

// UpdateStatus updates a ticket's status.
func (s *Service) UpdateStatus(ctx context.Context, ticketID int64, status string) error {
	oldTicket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return err
	}
	previousStatus := oldTicket.Status

	updates := map[string]interface{}{"status": status}
	now := time.Now()
	switch status {
	case domainTicket.TicketStatusInProgress:
		updates["started_at"] = now
	case domainTicket.TicketStatusDone:
		updates["completed_at"] = now
	}

	if err := s.repo.UpdateFields(ctx, ticketID, updates); err != nil {
		slog.ErrorContext(ctx, "failed to update ticket status", "ticket_id", ticketID, "org_id", oldTicket.OrganizationID, "status", status, "error", err)
		return err
	}

	slog.InfoContext(ctx, "ticket status updated", "ticket_id", ticketID, "slug", oldTicket.Slug, "from", previousStatus, "to", status)
	s.publishEvent(ctx, TicketEventStatusChanged, oldTicket.OrganizationID, oldTicket.Slug, status, previousStatus)
	return nil
}

// DeleteTicket deletes a ticket and its associated comments within a transaction.
func (s *Service) DeleteTicket(ctx context.Context, ticketID int64) error {
	oldTicket, err := s.GetTicket(ctx, ticketID)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteTicketAtomic(ctx, ticketID); err != nil {
		slog.ErrorContext(ctx, "failed to delete ticket", "ticket_id", ticketID, "org_id", oldTicket.OrganizationID, "slug", oldTicket.Slug, "error", err)
		return err
	}

	// Cascade the content block (no DB FK — this is the hand-rolled
	// equivalent). Logged-not-returned because leaving a content block
	// behind is recoverable (later GC / admin tool) whereas refusing the
	// ticket delete after the ticket row is already gone would be worse.
	if oldTicket.ContentBlockID != nil {
		if err := s.deleteContentBlock(ctx, oldTicket.OrganizationID, oldTicket.ReporterID, *oldTicket.ContentBlockID); err != nil {
			slog.WarnContext(ctx, "ticket content block cascade delete failed",
				"ticket_id", ticketID, "block_id", *oldTicket.ContentBlockID, "err", err)
		}
	}

	slog.InfoContext(ctx, "ticket deleted", "ticket_id", ticketID, "slug", oldTicket.Slug, "org_id", oldTicket.OrganizationID)
	s.publishEvent(ctx, TicketEventDeleted, oldTicket.OrganizationID, oldTicket.Slug, "deleted", oldTicket.Status)
	return nil
}
