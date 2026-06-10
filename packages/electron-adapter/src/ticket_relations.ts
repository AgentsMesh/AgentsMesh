import { invoke } from "./invoke";
import { coerceConnectResponse } from "./connect-response";
import type { ITicketRelationsService } from "@agentsmesh/service-interface";

// Forwards the Connect-RPC Uint8Array round-trip to the node-bridge's
// `ticket_relations_*_connect` napi commands. Naming mirrors the wasm
// surface (snake_case, `_connect` suffix) so the same TS adapter can
// switch between renderer and Electron transports.
export class ElectronTicketRelationsService implements ITicketRelationsService {
  async list_relations_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "ticketRelationsListRelationsConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async create_relation_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "ticketRelationsCreateRelationConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async delete_relation_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "ticketRelationsDeleteRelationConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async list_commits_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "ticketRelationsListCommitsConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async link_commit_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "ticketRelationsLinkCommitConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async unlink_commit_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "ticketRelationsUnlinkCommitConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async list_merge_requests_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "ticketRelationsListMergeRequestsConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async list_comments_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "ticketRelationsListCommentsConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async create_comment_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "ticketRelationsCreateCommentConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async update_comment_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "ticketRelationsUpdateCommentConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async delete_comment_connect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "ticketRelationsDeleteCommentConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }
}
