/**
 * ServiceError — typed mirror of Rust `agentsmesh-types::ServiceError`.
 *
 * FFI wire format: Rust's `ServiceError::to_wire()` produces JSON that parses
 * directly into this discriminated union. Front-end catches see either a JSON
 * string (new) or a plain message (legacy paths) — `parseServiceError` handles
 * both and always returns a typed object so callers can discriminate on `kind`
 * instead of regex-matching error messages.
 */
import { SERVICE_ERROR_KIND_SET } from "@agentsmesh/service-interface";

export type ServiceError =
  | { kind: "http"; status: number; code?: string; message: string }
  | { kind: "auth_expired" }
  | { kind: "network"; message: string }
  | { kind: "invalid_json"; message: string }
  | { kind: "resource_not_found"; resource: string; id?: string }
  | { kind: "unknown"; message: string };

function extractMessage(err: unknown): string {
  if (err == null) return "";
  if (typeof err === "string") return err;
  if (err instanceof Error) return err.message;
  if (typeof err === "object" && "message" in err) {
    const m = (err as { message: unknown }).message;
    if (typeof m === "string") return m;
  }
  try {
    return String(err);
  } catch {
    return "";
  }
}

function tryParseJson(msg: string): ServiceError | null {
  const trimmed = msg.trim();
  if (!trimmed.startsWith("{")) return null;
  try {
    const parsed = JSON.parse(trimmed) as { kind?: unknown };
    if (typeof parsed?.kind === "string" && SERVICE_ERROR_KIND_SET.has(parsed.kind)) {
      return parsed as ServiceError;
    }
  } catch {
    // fall through to legacy extraction
  }
  return null;
}

function fallback(msg: string): ServiceError {
  // Legacy paths that still emit human strings (e.g. HTTP clients outside the
  // Rust pipeline). Best-effort shape recovery so downstream typed helpers
  // still work during the migration.
  const status404 = /HTTP\s*404|RESOURCE_NOT_FOUND/i.test(msg);
  if (status404) {
    return { kind: "resource_not_found", resource: "resource" };
  }
  const httpMatch = /^HTTP\s*(\d{3}):\s*(.*)$/i.exec(msg);
  if (httpMatch) {
    return {
      kind: "http",
      status: Number(httpMatch[1]),
      message: httpMatch[2] ?? msg,
    };
  }
  if (/auth\s*expired/i.test(msg)) {
    return { kind: "auth_expired" };
  }
  return { kind: "unknown", message: msg };
}

export function parseServiceError(err: unknown): ServiceError {
  const msg = extractMessage(err);
  return tryParseJson(msg) ?? fallback(msg);
}

export function isResourceNotFound(
  err: unknown,
  resource?: string,
): boolean {
  const svc = parseServiceError(err);
  if (svc.kind !== "resource_not_found") return false;
  if (!resource) return true;
  return svc.resource.toLowerCase() === resource.toLowerCase();
}

export function isAuthExpired(err: unknown): boolean {
  return parseServiceError(err).kind === "auth_expired";
}

// GetPodConnection rejects with `failed_precondition: "pod is not active"` while
// a pod is still spinning up or has just completed — a normal lifecycle
// transient, not a connection failure. There is no distinct ServiceError kind
// for it (connect_call.rs leaves `code: None`, not parsing the Connect-JSON
// body), so match the stable backend message (backend connection.go).
export function isPodNotConnectable(err: unknown): boolean {
  return /pod is not active/i.test(extractMessage(err));
}

export function getErrorStatus(err: unknown): number | undefined {
  const svc = parseServiceError(err);
  if (svc.kind === "http") return svc.status;
  if (svc.kind === "resource_not_found") return 404;
  if (svc.kind === "auth_expired") return 401;
  return undefined;
}

export function getErrorCode(err: unknown): string | undefined {
  const svc = parseServiceError(err);
  if (svc.kind === "http") return svc.code;
  return undefined;
}

export function getErrorMessage(err: unknown): string {
  const svc = parseServiceError(err);
  switch (svc.kind) {
    case "http":
      return svc.message;
    case "auth_expired":
      return "auth expired";
    case "network":
      return svc.message;
    case "invalid_json":
      return svc.message;
    case "resource_not_found":
      return `${svc.resource} not found`;
    case "unknown":
      return svc.message;
  }
}
