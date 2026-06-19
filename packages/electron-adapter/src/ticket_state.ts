import type { ITicketState } from "@agentsmesh/service-interface";
import { fromBinary, create, toBinary } from "@bufbuild/protobuf";
import {
  ReplaceCachedTicketsRequestSchema, InsertCreatedTicketRequestSchema,
  PatchCachedTicketRequestSchema, ApplyTicketStatusEventRequestSchema,
  ApplyTicketDeletedEventRequestSchema, ReplaceBoardColumnsRequestSchema,
  AppendBoardColumnTicketsRequestSchema, SetCurrentTicketRequestSchema,
  ReplaceCachedLabelsRequestSchema, InsertCreatedLabelRequestSchema,
  RemoveCachedLabelRequestSchema, FilterTicketsRequestSchema,
  FilterTicketsResponseSchema,
} from "@agentsmesh/proto/ticket_state/v1/ticket_state_pb";
import {
  TicketSchema, BoardSchema, ListTicketsResponseSchema, ListLabelsResponseSchema,
} from "@agentsmesh/proto/ticket/v1/ticket_pb";
import {
  ticketToCache, boardColumnToCache, labelToCache, cacheTicketToProto,
  type CachedBoardColumn,
} from "./projections/ticket";
import {
  ticketsBytes, currentTicketBytes, boardColumnsBytes, labelsBytes, ticketPodsBytes,
} from "./ticket_cache_to_bytes";
import type { TicketData } from "@agentsmesh/service-interface";

// Desktop ticket-state mirror of clients/core/crates/wasm WasmTicketState
// (proto-bytes-in / json-out) so the shared web store reads identical payloads
// on both platforms. Renderer-local only — unlike pod.ts there is no
// fire-and-forget to main: ticket realtime rides the generic realtime:event
// channel (not a per-domain snapshot) and no app_ticket_* NAPI command exists.
export class ElectronTicketState implements ITicketState {
  private _ticketsCache = "[]";
  private _boardColumnsCache = "[]";
  private _labelsCache = "[]";
  private _currentTicket: string | null = null;
  // ticket→pods is mirrored here (not the Service) because getTicketState() —
  // not getTicketService() — backs useTicketPods' synchronous read/write.
  private _ticketPodsCache: Record<string, string> = {};

  tickets_json(): string { return this._ticketsCache; }
  board_columns_json(): string { return this._boardColumnsCache; }
  labels_json(): string { return this._labelsCache; }
  current_ticket_json(): unknown { return this._currentTicket; }

  set_ticket_pods(slug: string, podsJson: string): void { this._ticketPodsCache[slug] = podsJson; }
  ticket_pods_bytes(slug: string): Uint8Array { return ticketPodsBytes(this._ticketPodsCache[slug] ?? "[]"); }

  // Read side (B, zero-JSON): re-encode renderer cache into state proto bytes
  // (the web selector decodes these via ticketToCache for shape parity).
  tickets_bytes(): Uint8Array { return ticketsBytes(this._ticketsCache); }
  current_ticket_bytes(): Uint8Array { return currentTicketBytes(this._currentTicket); }
  board_columns_bytes(): Uint8Array { return boardColumnsBytes(this._boardColumnsCache); }
  labels_bytes(): Uint8Array { return labelsBytes(this._labelsCache); }

  // Fetch→state (B): wire Ticket == cache Ticket. Renderer-local only — ticket
  // realtime rides the generic realtime:event channel, no app_ticket_* NAPI.
  apply_fetched_tickets(respBytes: Uint8Array): void {
    const resp = fromBinary(ListTicketsResponseSchema, respBytes);
    this._ticketsCache = JSON.stringify(resp.items.map(ticketToCache));
  }

  apply_fetched_current_ticket(respBytes: Uint8Array): void {
    this._currentTicket = JSON.stringify(ticketToCache(fromBinary(TicketSchema, respBytes)));
  }

  apply_fetched_board_columns(respBytes: Uint8Array): void {
    const cols = fromBinary(BoardSchema, respBytes).columns.map(boardColumnToCache);
    this._boardColumnsCache = JSON.stringify(cols);
    this._ticketsCache = JSON.stringify(cols.flatMap((c) => c.tickets));
  }

  apply_appended_board_column_tickets(status: string, respBytes: Uint8Array): void {
    const resp = fromBinary(ListTicketsResponseSchema, respBytes);
    const cols = this._columns();
    const col = cols.find((c) => c.status === status);
    if (!col) return;
    const add = resp.items.map(ticketToCache);
    col.tickets.push(...add);
    this._boardColumnsCache = JSON.stringify(cols);
    const tickets = this._tickets();
    tickets.push(...add);
    this._ticketsCache = JSON.stringify(tickets);
  }

  apply_fetched_labels(respBytes: Uint8Array): void {
    const resp = fromBinary(ListLabelsResponseSchema, respBytes);
    this._labelsCache = JSON.stringify(resp.items.map(labelToCache));
  }

  insert_created_ticket(reqBytes: Uint8Array): void {
    const req = fromBinary(InsertCreatedTicketRequestSchema, reqBytes);
    if (!req.ticket) return;
    const tickets = this._tickets();
    tickets.push(ticketToCache(req.ticket));
    this._ticketsCache = JSON.stringify(tickets);
  }

  patch_cached_ticket(reqBytes: Uint8Array): void {
    const req = fromBinary(PatchCachedTicketRequestSchema, reqBytes);
    if (!req.ticket) return;
    const updated = ticketToCache(req.ticket);
    this._ticketsCache = JSON.stringify(this._tickets().map((t) => (t.slug === req.slug ? updated : t)));
    this._replaceInColumns(req.slug, updated);
    if (this._currentSlug() === req.slug) this._currentTicket = JSON.stringify(updated);
  }

  apply_ticket_status_event(reqBytes: Uint8Array): void {
    const req = fromBinary(ApplyTicketStatusEventRequestSchema, reqBytes);
    this._ticketsCache = JSON.stringify(
      this._tickets().map((t) => (t.slug === req.slug ? { ...t, status: req.status } : t)),
    );
    const cur = this._current();
    if (cur && cur.slug === req.slug) this._currentTicket = JSON.stringify({ ...cur, status: req.status });
  }

  apply_ticket_deleted_event(reqBytes: Uint8Array): void {
    const req = fromBinary(ApplyTicketDeletedEventRequestSchema, reqBytes);
    this._ticketsCache = JSON.stringify(this._tickets().filter((t) => t.slug !== req.slug));
    const cols = this._columns();
    for (const c of cols) c.tickets = c.tickets.filter((t) => t.slug !== req.slug);
    this._boardColumnsCache = JSON.stringify(cols);
    if (this._currentSlug() === req.slug) this._currentTicket = null;
  }

  // set_board_columns flattens column tickets into the flat list AND stores the
  // columns — the board renders cards from the flat list, counts from columns.
  replace_board_columns(reqBytes: Uint8Array): void {
    const req = fromBinary(ReplaceBoardColumnsRequestSchema, reqBytes);
    const cols = req.columns.map(boardColumnToCache);
    this._boardColumnsCache = JSON.stringify(cols);
    this._ticketsCache = JSON.stringify(cols.flatMap((c) => c.tickets));
  }

  append_board_column_tickets(reqBytes: Uint8Array): void {
    const req = fromBinary(AppendBoardColumnTicketsRequestSchema, reqBytes);
    const cols = this._columns();
    const col = cols.find((c) => c.status === req.status);
    if (!col) return;
    const add = req.tickets.map(ticketToCache);
    col.tickets.push(...add);
    this._boardColumnsCache = JSON.stringify(cols);
    const tickets = this._tickets();
    tickets.push(...add);
    this._ticketsCache = JSON.stringify(tickets);
  }

  set_current_ticket(reqBytes: Uint8Array): void {
    const req = fromBinary(SetCurrentTicketRequestSchema, reqBytes);
    this._currentTicket = req.ticket ? JSON.stringify(ticketToCache(req.ticket)) : null;
  }

  replace_cached_labels(reqBytes: Uint8Array): void {
    const req = fromBinary(ReplaceCachedLabelsRequestSchema, reqBytes);
    this._labelsCache = JSON.stringify(req.labels.map(labelToCache));
  }

  insert_created_label(reqBytes: Uint8Array): void {
    const req = fromBinary(InsertCreatedLabelRequestSchema, reqBytes);
    if (!req.label) return;
    const labels = JSON.parse(this._labelsCache) as unknown[];
    labels.push(labelToCache(req.label));
    this._labelsCache = JSON.stringify(labels);
  }

  remove_cached_label(reqBytes: Uint8Array): void {
    const req = fromBinary(RemoveCachedLabelRequestSchema, reqBytes);
    const id = Number(req.id);
    const labels = JSON.parse(this._labelsCache) as { id: number }[];
    this._labelsCache = JSON.stringify(labels.filter((l) => l.id !== id));
  }

  filter_tickets(reqBytes: Uint8Array): Uint8Array {
    const req = fromBinary(FilterTicketsRequestSchema, reqBytes);
    const search = req.search.toLowerCase();
    const repoIds = req.repositoryIds.map(Number);
    const matches = this._tickets().filter((t) => {
      if (search && !t.title.toLowerCase().includes(search) && !t.slug.toLowerCase().includes(search)) return false;
      if (req.statuses.length && !req.statuses.includes(t.status)) return false;
      if (req.priorities.length && !req.priorities.includes(t.priority)) return false;
      if (repoIds.length && !repoIds.includes(t.repository_id ?? 0)) return false;
      return true;
    });
    const resp = create(FilterTicketsResponseSchema, { tickets: matches.map(cacheTicketToProto) });
    return toBinary(FilterTicketsResponseSchema, resp);
  }

  private _tickets(): TicketData[] { return JSON.parse(this._ticketsCache) as TicketData[]; }
  private _columns(): CachedBoardColumn[] { return JSON.parse(this._boardColumnsCache) as CachedBoardColumn[]; }
  private _current(): TicketData | null {
    return this._currentTicket ? (JSON.parse(this._currentTicket) as TicketData) : null;
  }
  private _currentSlug(): string | null { return this._current()?.slug ?? null; }

  private _replaceInColumns(slug: string, updated: TicketData): void {
    const cols = this._columns();
    for (const c of cols) c.tickets = c.tickets.map((t) => (t.slug === slug ? updated : t));
    this._boardColumnsCache = JSON.stringify(cols);
  }
}
