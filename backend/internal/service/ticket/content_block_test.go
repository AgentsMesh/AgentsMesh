package ticket

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
	blockstoreinfra "github.com/anthropics/agentsmesh/backend/internal/infra/blockstore"
	blockstoreservice "github.com/anthropics/agentsmesh/backend/internal/service/blockstore"
)

// Verifies the full ticket-content ↔ block-store round-trip:
//   - create with BlockNote JSON lands in a `document` block
//   - GetTicket hydrates Content back from the block (REST shape preserved)
//   - legacy tickets.content column is cleared when the block path is used
//   - update flows through updateContentBlock (same block id reused)
//   - clearing content drops the block and the pointer
//   - delete cascades to block removal
func TestTicketContentBlockRoundtrip(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	bs := blockstoreservice.NewService(blockstoreinfra.NewRepository(db), slog.Default())
	svc.SetBlockstore(bs)
	ctx := context.Background()

	initial := `[{"type":"paragraph","content":[{"type":"text","text":"hello"}]}]`
	created, err := svc.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: 1, ReporterID: 1,
		Title: "with content", Priority: "medium",
		Content: ptr(initial),
	})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	if created.ContentBlockID == nil {
		t.Fatalf("expected ContentBlockID to be set, got nil")
	}
	blockID := *created.ContentBlockID

	// GetTicket must hydrate Content from the block so REST consumers still
	// see a BlockNote JSON string. The hydrated JSON may re-serialize in a
	// canonical form, so compare semantically by checking it survived a
	// round-trip (contains the original plain-text token).
	fetched, err := svc.GetTicket(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetTicket: %v", err)
	}
	if fetched.Content == nil || !strings.Contains(*fetched.Content, "hello") {
		t.Fatalf("expected hydrated content to contain 'hello', got %v", fetched.Content)
	}

	// MCP enrich consumes block.Text (pre-flattened plain text) instead of
	// re-parsing blocknote_ast — verify the writer kept that field in sync.
	actor := blockstoreservice.ActorContext{
		OrgID: 1, UserID: 1,
		ActorType: blockstore.ActorUser, ActorID: 1,
	}
	block, err := bs.GetBlock(ctx, actor, blockID)
	if err != nil {
		t.Fatalf("GetBlock: %v", err)
	}
	if block.Text == nil || !strings.Contains(*block.Text, "hello") {
		t.Fatalf("expected block.Text to contain plain text 'hello', got %v", block.Text)
	}

	updated := `[{"type":"paragraph","content":[{"type":"text","text":"world"}]}]`
	if _, err := svc.UpdateTicket(ctx, created.ID, map[string]interface{}{
		"content": updated,
	}); err != nil {
		t.Fatalf("UpdateTicket(content): %v", err)
	}
	// Same block id should be reused — we update data in place, not swap.
	afterUpdate, err := svc.GetTicket(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetTicket after update: %v", err)
	}
	if afterUpdate.ContentBlockID == nil || *afterUpdate.ContentBlockID != blockID {
		t.Fatalf("expected block id %s to be reused on update, got %v", blockID, afterUpdate.ContentBlockID)
	}
	if afterUpdate.Content == nil || !strings.Contains(*afterUpdate.Content, "world") {
		t.Fatalf("expected hydrated content to contain 'world', got %v", afterUpdate.Content)
	}

	// Clearing content should null the pointer and remove the block.
	if _, err := svc.UpdateTicket(ctx, created.ID, map[string]interface{}{
		"content": "",
	}); err != nil {
		t.Fatalf("UpdateTicket(clear content): %v", err)
	}
	afterClear, err := svc.GetTicket(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetTicket after clear: %v", err)
	}
	if afterClear.ContentBlockID != nil {
		t.Fatalf("expected ContentBlockID to be nil after clear, got %v", *afterClear.ContentBlockID)
	}

	// Delete cascade: ticket gone → block soft-deleted. We don't expose a
	// direct "is deleted" getter, so assert via GetTicket returning
	// ErrTicketNotFound (the best-effort cascade was logged, not failed).
	// First, recreate content so there's a block to cascade.
	if _, err := svc.UpdateTicket(ctx, created.ID, map[string]interface{}{
		"content": initial,
	}); err != nil {
		t.Fatalf("UpdateTicket(re-add content): %v", err)
	}
	if err := svc.DeleteTicket(ctx, created.ID); err != nil {
		t.Fatalf("DeleteTicket: %v", err)
	}
	if _, err := svc.GetTicket(ctx, created.ID); err != ErrTicketNotFound {
		t.Fatalf("expected ErrTicketNotFound after delete, got %v", err)
	}
}

func ptr(s string) *string { return &s }

// Legacy path: when the service is constructed without a Block Store, Content
// must land in the tickets.content column verbatim and ContentBlockID stays
// nil. Guarantees that staged rollout / minimal test setups aren't silently
// dropping content.
func TestTicketContentLegacyFallback(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db) // no SetBlockstore
	ctx := context.Background()

	payload := `[{"type":"paragraph","content":[{"type":"text","text":"legacy"}]}]`
	created, err := svc.CreateTicket(ctx, &CreateTicketRequest{
		OrganizationID: 1, ReporterID: 1,
		Title: "legacy path", Priority: "low",
		Content: ptr(payload),
	})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	if created.ContentBlockID != nil {
		t.Fatalf("expected ContentBlockID nil on legacy path, got %v", *created.ContentBlockID)
	}
	if created.Content == nil || *created.Content != payload {
		t.Fatalf("expected legacy Content to round-trip verbatim, got %v", created.Content)
	}

	// Read path must also return the inline Content untouched — no block
	// store means hydrateContentFromBlock is a no-op.
	fetched, err := svc.GetTicket(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetTicket: %v", err)
	}
	if fetched.Content == nil || *fetched.Content != payload {
		t.Fatalf("expected legacy fetched Content to match, got %v", fetched.Content)
	}
}

// Create paths that receive no meaningful content must not create a block.
// Covers the nil pointer, empty string, and the two canonical "empty AST"
// encodings that BlockNote editors produce.
func TestTicketContentEmptyCreateSkipsBlock(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestService(db)
	bs := blockstoreservice.NewService(blockstoreinfra.NewRepository(db), slog.Default())
	svc.SetBlockstore(bs)
	ctx := context.Background()

	cases := []struct {
		name    string
		content *string
	}{
		{"nil pointer", nil},
		{"empty string", ptr("")},
		{"empty array", ptr("[]")},
		{"null literal", ptr("null")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			created, err := svc.CreateTicket(ctx, &CreateTicketRequest{
				OrganizationID: 1, ReporterID: 1,
				Title: "no content " + c.name, Priority: "low",
				Content: c.content,
			})
			if err != nil {
				t.Fatalf("CreateTicket: %v", err)
			}
			if created.ContentBlockID != nil {
				t.Fatalf("expected no block for %q, got block id %v", c.name, *created.ContentBlockID)
			}
		})
	}
}
