-- Ticket content migration to Block Store.
-- Ticket.content (a raw BlockNote JSON TEXT column) is being replaced by a
-- reference to a Block Store `document` block, so the text gets the full
-- block pipeline: op log, time travel, realtime WS broadcast, semantic
-- embeddings, unified renderer.
--
-- Design notes:
--   * No FOREIGN KEY between tickets.content_block_id and blocks.id.
--     Tickets and Block Store are separate aggregates; coupling them with a
--     DB-level FK would force one module to know about the other's lifecycle
--     at the schema level. Instead the ticket service handles cascade
--     semantics in code (delete-ticket → delete block; block soft-deleted →
--     ticket reads gracefully treat content as empty).
--   * Index is needed because migrations and maintenance queries (e.g. "find
--     tickets still pointing at block X") run by content_block_id.
--   * Old `content` TEXT column stays put in this migration; it's cleared in
--     a follow-up once all historical rows have been backfilled into blocks.

ALTER TABLE tickets ADD COLUMN content_block_id UUID NULL;
CREATE INDEX idx_tickets_content_block ON tickets(content_block_id)
    WHERE content_block_id IS NOT NULL;
