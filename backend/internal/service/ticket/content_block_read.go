package ticket

import (
	"context"
	"encoding/json"
	"log/slog"

	domainTicket "github.com/anthropics/agentsmesh/backend/internal/domain/ticket"
)

// hydrateContentFromBlock rebuilds the legacy Content string from the
// backing document block when ContentBlockID is set. REST consumers still
// expect `content` (BlockNote JSON) on ticket responses, so single-ticket
// reads route through here. List endpoints skip it on purpose — hydrating
// N blocks per list is too expensive and list rows usually omit content.
//
// Missing / deleted blocks are tolerated: Content is left as the DB value
// (usually nil) and the error is logged. The ticket row survives a dangling
// pointer by design — no FK, business-layer cascade best-effort.
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
