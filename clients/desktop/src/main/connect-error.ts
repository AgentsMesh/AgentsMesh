// Shapes connectCall failures into the Rust `ServiceError` wire format
// (`clients/web/src/lib/errors/serviceError.ts`) so the renderer's existing
// parse layer — built for wasm errors — handles desktop IPC errors too.
// Without this the renderer only sees a flattened string ("409 Conflict
// <url> <body>") and can't recover the Connect code for i18n mapping.

const MAX_MESSAGE_LEN = 500;

export async function connectErrorFromResponse(res: Response): Promise<Error> {
  const text = await res.text().catch(() => "");
  let code: string | undefined;
  let message = "";
  try {
    const body = JSON.parse(text) as { code?: unknown; message?: unknown };
    if (typeof body.code === "string") code = body.code;
    if (typeof body.message === "string") message = body.message;
  } catch {
    // non-Connect body (proxy HTML, plain text) — fall through to raw text
  }
  if (!message) message = text || res.statusText;
  return new Error(
    JSON.stringify({
      kind: "http",
      status: res.status,
      code,
      message: message.slice(0, MAX_MESSAGE_LEN),
    }),
  );
}

export function connectNetworkError(err: unknown): Error {
  const message = err instanceof Error ? err.message : String(err);
  return new Error(
    JSON.stringify({ kind: "network", message: message.slice(0, MAX_MESSAGE_LEN) }),
  );
}
