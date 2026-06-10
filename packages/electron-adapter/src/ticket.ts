import { invoke } from "./invoke";
import { coerceConnectResponse } from "./connect-response";
import type { ITicketService } from "@agentsmesh/service-interface";

export class ElectronTicketService implements ITicketService {
  private _ticketPodsCache: Record<string, string> = {};

  async get_ticket_pods(slug: string, activeOnly?: boolean | null): Promise<string> {
    const result = await invoke<string>("ticketGetTicketPods", slug, activeOnly);
    this._ticketPodsCache[slug] = result;
    return result;
  }

  ticket_pods_json(slug: string): string {
    const raw = this._ticketPodsCache[slug];
    if (!raw) return "[]";
    try {
      const parsed = JSON.parse(raw) as { pods?: unknown[] };
      return JSON.stringify(parsed.pods ?? []);
    } catch {
      return "[]";
    }
  }

  // ─── Connect-RPC bridges (Uint8Array round-trip) ──────────────────

  async list_tickets_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>("ticketListTicketsConnect", Array.from(request));
    return coerceConnectResponse(bytes);
  }

  async get_ticket_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>("ticketGetTicketConnect", Array.from(request));
    return coerceConnectResponse(bytes);
  }

  async create_ticket_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>("ticketCreateTicketConnect", Array.from(request));
    return coerceConnectResponse(bytes);
  }

  async update_ticket_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>("ticketUpdateTicketConnect", Array.from(request));
    return coerceConnectResponse(bytes);
  }

  async delete_ticket_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>("ticketDeleteTicketConnect", Array.from(request));
    return coerceConnectResponse(bytes);
  }

  async update_ticket_status_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>("ticketUpdateTicketStatusConnect", Array.from(request));
    return coerceConnectResponse(bytes);
  }

  async get_active_tickets_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>("ticketGetActiveTicketsConnect", Array.from(request));
    return coerceConnectResponse(bytes);
  }

  async get_board_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>("ticketGetBoardConnect", Array.from(request));
    return coerceConnectResponse(bytes);
  }

  async get_sub_tickets_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>("ticketGetSubTicketsConnect", Array.from(request));
    return coerceConnectResponse(bytes);
  }

  async add_assignee_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>("ticketAddAssigneeConnect", Array.from(request));
    return coerceConnectResponse(bytes);
  }

  async remove_assignee_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>("ticketRemoveAssigneeConnect", Array.from(request));
    return coerceConnectResponse(bytes);
  }

  async list_labels_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>("ticketListLabelsConnect", Array.from(request));
    return coerceConnectResponse(bytes);
  }

  async create_label_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>("ticketCreateLabelConnect", Array.from(request));
    return coerceConnectResponse(bytes);
  }

  async update_label_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>("ticketUpdateLabelConnect", Array.from(request));
    return coerceConnectResponse(bytes);
  }

  async delete_label_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>("ticketDeleteLabelConnect", Array.from(request));
    return coerceConnectResponse(bytes);
  }

  async add_label_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>("ticketAddLabelConnect", Array.from(request));
    return coerceConnectResponse(bytes);
  }

  async remove_label_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>("ticketRemoveLabelConnect", Array.from(request));
    return coerceConnectResponse(bytes);
  }
}
