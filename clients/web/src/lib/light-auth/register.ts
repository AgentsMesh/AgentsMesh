// Sign-up via REST. Backend returns the same shape as /auth/login when the
// account is created (with optional `is_email_verified: false`), so the user
// is technically "logged in" but the post-register flow redirects to
// /verify-email and waits for the link click before letting them into
// the dashboard.

import { lightFetch } from "./api-fetch";
import { persistLoginResponse, type AuthLoginResponse } from "./persist";

export interface LightRegisterInput {
  email: string;
  username: string;
  password: string;
  name: string;
}

export async function lightRegister(input: LightRegisterInput): Promise<AuthLoginResponse> {
  const resp = await lightFetch<AuthLoginResponse>("/api/v1/auth/register", {
    method: "POST",
    body: input,
  });
  persistLoginResponse(resp);
  return resp;
}
