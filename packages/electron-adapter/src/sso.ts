import { invoke } from "./invoke";
import { coerceConnectResponse } from "./connect-response";
import type { ISSOService } from "@agentsmesh/service-interface";

export class ElectronSSOService implements ISSOService {
  async discoverConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "ssoDiscoverConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async ldapAuthConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "ssoLdapAuthConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }
}
