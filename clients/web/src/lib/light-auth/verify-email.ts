// /auth/verify-email and /auth/resend-verification — REST handlers in
// backend/internal/api/rest/v1/auth_verification.go. Verify-email returns
// a fresh token pair (so the user is logged in after confirming), resend
// returns a status message only.

import { lightFetch } from "./api-fetch";
import { persistLoginResponse, type AuthLoginResponse } from "./persist";

export async function lightVerifyEmail(token: string): Promise<AuthLoginResponse> {
  const resp = await lightFetch<AuthLoginResponse>("/api/v1/auth/verify-email", {
    method: "POST",
    body: { token },
  });
  persistLoginResponse(resp);
  return resp;
}

export async function lightResendVerification(email: string): Promise<void> {
  await lightFetch<void>("/api/v1/auth/resend-verification", {
    method: "POST",
    body: { email },
  });
}
