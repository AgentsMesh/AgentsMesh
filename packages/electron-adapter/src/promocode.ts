import { invoke } from "./invoke";
import { coerceConnectResponse } from "./connect-response";
import type { IPromoCodeService } from "@agentsmesh/service-interface";

export class ElectronPromoCodeService implements IPromoCodeService {
  async validatePromoCodeConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "promocodeValidatePromoCodeConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async redeemPromoCodeConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "promocodeRedeemPromoCodeConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async getRedemptionHistoryConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "promocodeGetRedemptionHistoryConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }
}
