import { invoke } from "./invoke";
import type { ISupportTicketService } from "@agentsmesh/service-interface";

export class ElectronSupportTicketService implements ISupportTicketService {
  async create_ticket(title: string, category: string, content: string, priority: string | null | undefined, fileData: Uint8Array[], fileNames: string[]): Promise<string> {
    return invoke<string>("supportTicketCreateTicket", title, category, content, priority, fileData.map(d => Array.from(d)), fileNames);
  }

  async add_message(ticketId: bigint, content: string, fileData: Uint8Array[], fileNames: string[]): Promise<string> {
    return invoke<string>("supportTicketAddMessage", Number(ticketId), content, fileData.map(d => Array.from(d)), fileNames);
  }

  async listSupportTicketsConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "supportTicketListSupportTicketsConnect", Array.from(request),
    );
    return bytes instanceof Uint8Array ? bytes : new Uint8Array(bytes);
  }

  async getSupportTicketConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "supportTicketGetSupportTicketConnect", Array.from(request),
    );
    return bytes instanceof Uint8Array ? bytes : new Uint8Array(bytes);
  }

  async getAttachmentUrlConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "supportTicketGetAttachmentUrlConnect", Array.from(request),
    );
    return bytes instanceof Uint8Array ? bytes : new Uint8Array(bytes);
  }
}
