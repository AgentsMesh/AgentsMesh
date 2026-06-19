import { describe, it, expect } from "vitest";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { TicketSchema, ListTicketsResponseSchema } from "@agentsmesh/proto/ticket/v1/ticket_pb";
import { ReplaceCachedTicketsRequestSchema } from "@agentsmesh/proto/ticket_state/v1/ticket_state_pb";
import { ticketToCache } from "./projections/ticket";
import { ticketsBytes } from "./ticket_cache_to_bytes";
import { ElectronTicketState } from "./ticket_state";

const wireTicket = (id: bigint, slug: string) => create(TicketSchema, {
  id, number: 7, slug, title: "t", content: "c", status: "todo", priority: "high",
  severity: "s", estimate: 3, dueDate: "2026-01-03", startedAt: "2026-01-01",
  completedAt: "", repositoryId: 9n, createdAt: "2026-01-01", updatedAt: "2026-01-02",
});

// cache→bytes must round-trip every field ticketToCache reads, or desktop
// diverges from web (which decodes the same bytes through ticketToCache).
describe("ticket cache→bytes round-trip", () => {
  it("preserves ticket fields through cache → bytes → state", () => {
    const cache = ticketToCache(wireTicket(1n, "T-1"));
    const decoded = fromBinary(ReplaceCachedTicketsRequestSchema, ticketsBytes(JSON.stringify([cache])));
    expect(ticketToCache(decoded.tickets[0])).toEqual(cache);
  });
});

describe("ElectronTicketState fetch→state", () => {
  it("apply_fetched_tickets caches + reads back via tickets_bytes", () => {
    const st = new ElectronTicketState();
    const bytes = toBinary(ListTicketsResponseSchema, create(ListTicketsResponseSchema, {
      items: [wireTicket(1n, "T-1"), wireTicket(2n, "T-2")],
    }));
    st.apply_fetched_tickets(bytes);
    const decoded = fromBinary(ReplaceCachedTicketsRequestSchema, st.tickets_bytes());
    expect(decoded.tickets.map((t) => t.slug)).toEqual(["T-1", "T-2"]);
  });
});
