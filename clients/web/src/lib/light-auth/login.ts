// Email/password login via REST. On success, persists the session blob
// so dashboard's wasm bootstrap can hydrate the rest of the state.

import { lightFetch } from "./api-fetch";
import { persistLoginResponse, type AuthLoginResponse } from "./persist";

export interface LightLoginInput {
  email: string;
  password: string;
}

export async function lightLogin(input: LightLoginInput): Promise<AuthLoginResponse> {
  const resp = await lightFetch<AuthLoginResponse>("/api/v1/auth/login", {
    method: "POST",
    body: { email: input.email, password: input.password },
  });
  persistLoginResponse(resp);
  return resp;
}
