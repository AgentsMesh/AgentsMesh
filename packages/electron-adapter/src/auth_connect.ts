import { invoke } from "./invoke";
import { coerceConnectResponse } from "./connect-response";
import type { IAuthConnectService } from "@agentsmesh/service-interface";

export class ElectronAuthConnectService implements IAuthConnectService {
  async loginConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "authConnectLoginConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async registerConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "authConnectRegisterConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async refreshTokenConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "authConnectRefreshTokenConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async verifyEmailConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "authConnectVerifyEmailConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async resendVerificationConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "authConnectResendVerificationConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async forgotPasswordConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "authConnectForgotPasswordConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async resetPasswordConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "authConnectResetPasswordConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async oauthRedirectConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "authConnectOauthRedirectConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async oauthCallbackConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "authConnectOauthCallbackConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }

  async logoutConnect(request: Uint8Array): Promise<Uint8Array> {
    const bytes = await invoke<number[] | Uint8Array>(
      "authConnectLogoutConnect", Array.from(request),
    );
    return coerceConnectResponse(bytes);
  }
}
