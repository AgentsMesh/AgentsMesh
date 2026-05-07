import { describe, it, expect } from "vitest";
import { buildTicketContext } from "../buildTicketContext";
import type { Ticket } from "@/stores/ticket";

const baseTicket: Ticket = {
  id: 7,
  number: 7,
  slug: "TICKET-7",
  title: "Investigate flaky tests",
  content: "Reproduce locally then bisect.",
  status: "todo",
  priority: "medium",
  repository_id: 3,
};

describe("buildTicketContext", () => {
  it("returns undefined when ticket.id is missing", () => {
    expect(buildTicketContext({ ...baseTicket, id: 0 }, "TICKET-7")).toBeUndefined();
  });

  it("returns a context with empty title when title is missing (matches legacy behavior)", () => {
    const ctx = buildTicketContext({ id: 1 }, "TICKET-1");
    expect(ctx).toEqual({
      id: 1,
      slug: "TICKET-1",
      title: "",
      description: undefined,
      repositoryId: undefined,
    });
  });

  it("maps ticket.content to description and copies repository_id", () => {
    expect(buildTicketContext(baseTicket, "TICKET-7")).toEqual({
      id: 7,
      slug: "TICKET-7",
      title: "Investigate flaky tests",
      description: "Reproduce locally then bisect.",
      repositoryId: 3,
    });
  });

  it("leaves description and repositoryId undefined when source fields are absent", () => {
    const ctx = buildTicketContext(
      { id: 9, title: "Bare", content: undefined, repository_id: undefined },
      "TICKET-9",
    );
    expect(ctx).toEqual({
      id: 9,
      slug: "TICKET-9",
      title: "Bare",
      description: undefined,
      repositoryId: undefined,
    });
  });

  it("uses the slug argument verbatim, not ticket.slug", () => {
    const ctx = buildTicketContext(baseTicket, "FROM-ROUTE");
    expect(ctx?.slug).toBe("FROM-ROUTE");
  });
});
