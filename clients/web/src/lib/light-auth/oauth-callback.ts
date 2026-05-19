// OAuth & SSO callback handlers consume the URL query params the backend
// appended after a successful provider login. Tokens arrive as `token` and
// `refresh_token`; `error` is set instead when the user cancelled or the
// provider returned a failure. Callers strip the params off the URL via
// history.replaceState() to avoid leaking tokens through Referer / history.

import { persistOAuthTokens } from "./persist";

export type OAuthCallbackResult =
  | { status: "ok"; token: string; refreshToken: string }
  | { status: "error"; reason: string };

export function consumeOAuthCallbackParams(
  params: URLSearchParams | { get(key: string): string | null },
): OAuthCallbackResult {
  const error = params.get("error");
  if (error) return { status: "error", reason: error };
  const token = params.get("token");
  if (!token) return { status: "error", reason: "missing_token" };
  const refreshToken = params.get("refresh_token") ?? "";
  persistOAuthTokens({ token, refreshToken });
  return { status: "ok", token, refreshToken };
}
