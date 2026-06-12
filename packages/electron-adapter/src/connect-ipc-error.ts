// Electron's ipcRenderer.invoke wraps a main-process throw as
// `Error: Error invoking remote method '<channel>': Error: <message>` — only
// the message string survives the IPC boundary. The main connectCall handler
// encodes failures as ServiceError wire JSON (main/connect-error.ts), so this
// strips the Electron prefix back off; the renderer's parseServiceError
// requires the message to START with `{` to recognize the wire format.

import { SERVICE_ERROR_KIND_SET } from "@agentsmesh/service-interface";

export function unwrapIpcServiceError(err: unknown): unknown {
  if (!(err instanceof Error)) return err;
  const idx = err.message.indexOf("{");
  if (idx < 0) return err;
  const candidate = err.message.slice(idx);
  try {
    const parsed = JSON.parse(candidate) as { kind?: unknown };
    if (typeof parsed?.kind === "string" && SERVICE_ERROR_KIND_SET.has(parsed.kind)) {
      return new Error(candidate);
    }
  } catch {
    // message contains a brace but no ServiceError payload — keep original
  }
  return err;
}
