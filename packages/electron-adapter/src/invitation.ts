import { invoke } from "./invoke";
import { coerceConnectResponse } from "./connect-response";
import type { IInvitationConnectService } from "@agentsmesh/service-interface";

export class ElectronInvitationConnectService implements IInvitationConnectService {
  async listInvitationsConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "invitationListInvitationsConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async createInvitationConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "invitationCreateInvitationConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async revokeInvitationConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "invitationRevokeInvitationConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async resendInvitationConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "invitationResendInvitationConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async acceptInvitationConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "invitationAcceptInvitationConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async listPendingInvitationsConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "invitationListPendingInvitationsConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async getInvitationByTokenConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "invitationGetInvitationByTokenConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }
}
