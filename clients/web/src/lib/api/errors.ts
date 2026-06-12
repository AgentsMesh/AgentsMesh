import { ApiError } from "./api-types";
import {
  getErrorCode as getServiceErrorCode,
  getErrorStatus as getServiceErrorStatus,
  getErrorMessage as getServiceErrorMessage,
  parseServiceError,
} from "@/lib/errors/serviceError";

/**
 * Extract the error code from an error.
 *
 * Handles both the legacy web `ApiError` class (direct HTTP calls) and the new
 * Rust `ServiceError` wire format (WASM / node-bridge). Callers don't need to
 * know which transport raised the error.
 */
export function getApiErrorCode(error: unknown): string | undefined {
  if (error instanceof ApiError) {
    return error.code;
  }
  return getServiceErrorCode(error);
}

export function getErrorSuggestion(error: unknown): string | undefined {
  if (!(error instanceof ApiError)) return undefined;
  const data = error.data as { suggestion?: unknown } | null | undefined;
  if (typeof data?.suggestion === "string" && data.suggestion.length > 0) {
    return data.suggestion;
  }
  return undefined;
}

export function isApiErrorCode(error: unknown, code: string): boolean {
  return getApiErrorCode(error) === code;
}

export function isApiStatus(error: unknown, status: number): boolean {
  if (error instanceof ApiError && error.status === status) return true;
  return getServiceErrorStatus(error) === status;
}

// Connect protocol codes whose meaning maps onto an existing backend business
// code — one apiErrors table serves both vocabularies after uppercasing.
const CONNECT_CODE_ALIASES: Record<string, string> = {
  NOT_FOUND: "RESOURCE_NOT_FOUND",
  INVALID_ARGUMENT: "INVALID_INPUT",
  INTERNAL: "INTERNAL_ERROR",
  UNAUTHENTICATED: "AUTH_REQUIRED",
  PERMISSION_DENIED: "ACCESS_DENIED",
};

/**
 * Get a localized error message, falling back through: i18n code translation
 * (context-scoped, then global) → server message → service-error message →
 * Error.message → `fallback`.
 *
 * `contextPrefix` lets a call site provide flow-specific wording per code:
 * lookup tries `<contextPrefix>.<CODE>` before `apiErrors.<CODE>` (e.g.
 * "runners.resume" → runners.resume.ALREADY_EXISTS).
 */
export function getLocalizedErrorMessage(
  error: unknown,
  t: (key: string, values?: Record<string, string>) => string,
  fallback: string,
  opts?: { contextPrefix?: string }
): string {
  const code = getApiErrorCode(error);
  if (code) {
    // Backend business codes are UPPER_SNAKE; Connect protocol codes arrive
    // lowercase (already_exists) — one apiErrors table serves both.
    const upper = code.toUpperCase();
    const canonical = CONNECT_CODE_ALIASES[upper] ?? upper;
    const keys = opts?.contextPrefix
      ? [`${opts.contextPrefix}.${canonical}`, `apiErrors.${canonical}`]
      : [`apiErrors.${canonical}`];
    for (const key of keys) {
      try {
        const translated = t(key);
        if (translated && translated !== key) {
          return translated;
        }
      } catch {
      }
    }
  }
  if (error instanceof ApiError) {
    if (error.serverMessage) return error.serverMessage;
  }
  const svc = parseServiceError(error);
  if (svc.kind !== "unknown") {
    return getServiceErrorMessage(error);
  }
  if (error instanceof Error) {
    return error.message;
  }
  return fallback;
}
