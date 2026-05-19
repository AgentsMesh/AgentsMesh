package ticket

import (
	"context"
	"encoding/json"
	"log/slog"

	domainTicket "github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

func (s *Service) hydrateContentFromBlock(ctx context.Context, t *domainTicket.Ticket) {
	if t == nil || t.ContentBlockID == nil || s.blockstore == nil {
		return
	}
	actor := actorForTicketUser(t.OrganizationID, t.ReporterID)
	block, err := s.blockstore.GetBlock(ctx, actor, *t.ContentBlockID)
	if err != nil {
		slog.WarnContext(ctx, "ticket content block fetch failed",
			"ticket_id", t.ID, "block_id", *t.ContentBlockID, "err", err)
		return
	}
	if block == nil {
		return
	}
	raw, ok := block.Data["blocknote_ast"]
	if !ok {
		return
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		slog.WarnContext(ctx, "ticket content block marshal failed",
			"ticket_id", t.ID, "block_id", *t.ContentBlockID, "err", err)
		return
	}
	s2 := string(encoded)
	t.Content = &s2
}
