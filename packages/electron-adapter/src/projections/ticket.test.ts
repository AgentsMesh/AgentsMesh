import { describe, it, expect } from "vitest";
import { create } from "@bufbuild/protobuf";
import { TicketSchema, BoardColumnSchema, LabelSchema } from "@agentsmesh/proto/ticket/v1/ticket_pb";
import { ticketToCache, boardColumnToCache, labelToCache, cacheTicketToProto } from "./ticket";

// Guards the shared ticket projection against proto-schema drift.

describe("ticketToCache", () => {
  it("maps every scalar field the ticket UI reads", () => {
    const c = ticketToCache(create(TicketSchema, {
      id: 9n, number: 42, slug: "DEV-42", title: "Fix auth",
      content: "body", status: "in_progress", priority: "high", severity: "sev2",
      estimate: 5, dueDate: "2026-02-01", startedAt: "t1", completedAt: "t2",
      createdAt: "c0", updatedAt: "u0", repositoryId: 3n,
    }));
    expect(c.id).toBe(9);
    expect(typeof c.id).toBe("number");
    expect(c.number).toBe(42);
    expect(c.slug).toBe("DEV-42");
    expect(c.status).toBe("in_progress");
    expect(c.priority).toBe("high");
    expect(c.severity).toBe("sev2");
    expect(c.due_date).toBe("2026-02-01");
    expect(c.started_at).toBe("t1");
    expect(c.completed_at).toBe("t2");
    expect(c.repository_id).toBe(3);
  });

  it("leaves absent repository_id undefined", () => {
    const c = ticketToCache(create(TicketSchema, { id: 1n, slug: "DEV-1", title: "x" }));
    expect(c.repository_id).toBeUndefined();
  });
});

describe("boardColumnToCache", () => {
  it("mirrors wasm board JSON: keeps total_count (not count), projects tickets", () => {
    const c = boardColumnToCache(create(BoardColumnSchema, {
      status: "todo", totalCount: 7n,
      tickets: [create(TicketSchema, { id: 1n, slug: "DEV-1", title: "a" })],
    }));
    expect(c.status).toBe("todo");
    expect(c.total_count).toBe(7);
    expect(c.tickets).toHaveLength(1);
    expect(c.tickets[0].slug).toBe("DEV-1");
  });
});

describe("labelToCache", () => {
  it("maps id/name/color", () => {
    const c = labelToCache(create(LabelSchema, { id: 5n, name: "bug", color: "#f00" }));
    expect(c).toEqual({ id: 5, name: "bug", color: "#f00" });
  });
});

describe("cacheTicketToProto", () => {
  it("round-trips a TicketData back to proto", () => {
    const t = ticketToCache(create(TicketSchema, {
      id: 9n, number: 42, slug: "DEV-42", title: "Fix auth", status: "todo", priority: "low",
    }));
    const p = cacheTicketToProto(t);
    expect(p.id).toBe(9n);
    expect(p.slug).toBe("DEV-42");
    expect(p.status).toBe("todo");
  });
});
