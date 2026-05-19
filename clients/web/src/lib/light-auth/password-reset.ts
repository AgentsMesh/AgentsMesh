// /auth/forgot-password sends a reset email (always 200 to avoid email
// enumeration — backend returns a generic message regardless of whether
// the email exists). /auth/reset-password completes the reset with a
// token + new password and returns 200 with a success message.

import { lightFetch } from "./api-fetch";

export async function lightForgotPassword(email: string): Promise<void> {
  await lightFetch<void>("/api/v1/auth/forgot-password", {
    method: "POST",
    body: { email },
  });
}

export interface LightResetPasswordInput {
  token: string;
  newPassword: string;
}

export async function lightResetPassword(input: LightResetPasswordInput): Promise<void> {
  await lightFetch<void>("/api/v1/auth/reset-password", {
    method: "POST",
    body: { token: input.token, new_password: input.newPassword },
  });
}
