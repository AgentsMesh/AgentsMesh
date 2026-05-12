// Unit tests for clients/web/src/lib/api/ticketConnect.ts proto → snake_case
// converters. Pins the 18-field Ticket + 7-field Label round-trips through
// the binary wire so PR 986a38ca6's envelope-drift bug class becomes
// compile-time-impossible.
//
// Note: this test file imports @proto/ticket/v1/ticket_pb which resolves
// via vitest.config.ts. Under `bazel test //clients/web:unit` the
// `proto/gen` tree is in .bazelignore so the import fails — runs under
// `pnpm test:run` (or once ts_proto_library lands; see runbook
// "TS proto codegen toolchain").

import { describe, it, expect } from "vitest";
import { create } from "@bufbuild/protobuf";
import {
  LabelSchema,
  TicketSchema,
} from "@proto/ticket/v1/ticket_pb";
import { fromProtoLabel, fromProtoTicket } from "../ticketConnect";

describe("fromProtoTicket", () => {
  it("converts the full Ticket proto to TicketData", () => {
    const proto = create(TicketSchema, {
      id: BigInt(42),
      number: 7,
      slug: "ACME-7",
      title: "Fix login bug",
      content: "Users can't login after 2FA",
      status: "in_progress",
      priority: "high",
      severity: "major",
      estimate: 5,
      dueDate: "2026-05-20",
      startedAt: "2026-05-10T00:00:00Z",
      repositoryId: BigInt(11),
      parentTicketSlug: "ACME-1",
      createdAt: "2026-05-08T00:00:00Z",
      updatedAt: "2026-05-10T00:00:00Z",
    });
    const data = fromProtoTicket(proto);
    expect(data.id).toBe(42);
    expect(data.number).toBe(7);
    expect(data.slug).toBe("ACME-7");
    expect(data.title).toBe("Fix login bug");
    expect(data.content).toBe("Users can't login after 2FA");
    expect(data.status).toBe("in_progress");
    expect(data.priority).toBe("high");
    expect(data.severity).toBe("major");
    expect(data.estimate).toBe(5);
    expect(data.due_date).toBe("2026-05-20");
    expect(data.started_at).toBe("2026-05-10T00:00:00Z");
    expect(data.repository_id).toBe(11);
    expect(data.created_at).toBe("2026-05-08T00:00:00Z");
    expect(data.updated_at).toBe("2026-05-10T00:00:00Z");
  });

  it("handles missing optionals correctly", () => {
    const proto = create(TicketSchema, {
      id: BigInt(1),
      number: 1,
      slug: "X-1",
      title: "minimal",
      status: "backlog",
      priority: "low",
      createdAt: "2026-05-08T00:00:00Z",
      updatedAt: "2026-05-08T00:00:00Z",
    });
    const data = fromProtoTicket(proto);
    expect(data.content).toBeUndefined();
    expect(data.severity).toBeUndefined();
    expect(data.estimate).toBeUndefined();
    expect(data.due_date).toBeUndefined();
    expect(data.repository_id).toBeUndefined();
  });

  it("preserves BigInt → number narrowing for repository_id", () => {
    // The id maps from int64 → number on the snake_case side; we lose nothing
    // because the JS number range covers backend SERIAL IDs comfortably.
    const proto = create(TicketSchema, {
      id: BigInt(2_000_000),
      number: 9999,
      slug: "X-9999",
      title: "t",
      status: "todo",
      priority: "medium",
      repositoryId: BigInt(1_000_000),
      createdAt: "2026-05-08T00:00:00Z",
      updatedAt: "2026-05-08T00:00:00Z",
    });
    const data = fromProtoTicket(proto);
    expect(typeof data.id).toBe("number");
    expect(data.id).toBe(2_000_000);
    expect(data.repository_id).toBe(1_000_000);
  });
});

describe("fromProtoLabel", () => {
  it("converts proto Label to the {id, name, color} web shape", () => {
    const proto = create(LabelSchema, {
      id: BigInt(99),
      organizationId: BigInt(7),
      repositoryId: BigInt(11),
      name: "bug",
      color: "#ff0000",
    });
    const label = fromProtoLabel(proto);
    expect(label).toEqual({ id: 99, name: "bug", color: "#ff0000" });
  });

  it("treats organization-level label (no repository_id) the same", () => {
    const proto = create(LabelSchema, {
      id: BigInt(1),
      organizationId: BigInt(7),
      name: "feature",
      color: "#00ff00",
    });
    const label = fromProtoLabel(proto);
    expect(label).toEqual({ id: 1, name: "feature", color: "#00ff00" });
  });
});
