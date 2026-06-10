import { invoke } from "./invoke";
import { coerceConnectResponse } from "./connect-response";
import type { IUserApiService } from "@agentsmesh/service-interface";

export class ElectronUserService implements IUserApiService {
  async getMeConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "userGetMeConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async updateMeConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "userUpdateMeConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async changePasswordConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "userChangePasswordConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async listIdentitiesConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "userListIdentitiesConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async deleteIdentityConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "userDeleteIdentityConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async searchUsersConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "userSearchUsersConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }
}
