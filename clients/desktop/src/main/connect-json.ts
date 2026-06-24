// Shape adapters for the legacy IPC aliases (legacy_api_aliases.ts): bridge the
// Connect-JSON wire back to the REST-shaped envelopes desktop e2e specs + the
// renderer caches read. Pure functions, no AppState/network dependency.

// camelCase → snake_case deep key rename.
export function snakeCaseDeep(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(snakeCaseDeep);
  if (value !== null && typeof value === "object") {
    const out: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
      const sk = k.replace(/([A-Z])/g, "_$1").toLowerCase();
      out[sk] = snakeCaseDeep(v);
    }
    return out;
  }
  return value;
}

// Connect-JSON serializes int64 as a JSON string to preserve precision; IPC
// callers may forward it as either a number or string.
export const toIdNumber = (v: number | string): number =>
  typeof v === "number" ? v : Number(v);

// Recursively coerce numeric-looking strings to numbers for known int64 fields.
// Connect-JSON emits them as strings, but desktop e2e callers + the legacy Rust
// napi shape expect numbers (`expect(joined.agent_count).toBe(1)` fails on "1").
const INT64_KEYS = new Set([
  "id", "organization_id", "repository_id", "ticket_id",
  "created_by_user_id", "member_count", "agent_count",
  "message_count", "sender_user_id", "channel_id", "reply_to",
]);

export function coerceInt64(value: unknown): unknown {
  if (Array.isArray(value)) return value.map(coerceInt64);
  if (value !== null && typeof value === "object") {
    const out: Record<string, unknown> = {};
    for (const [k, v] of Object.entries(value as Record<string, unknown>)) {
      if (INT64_KEYS.has(k) && typeof v === "string" && /^-?\d+$/.test(v)) {
        out[k] = Number(v);
      } else {
        out[k] = coerceInt64(v);
      }
    }
    return out;
  }
  return value;
}

// Connect-JSON omits proto3 defaults (0/""/false); the legacy napi shape always
// emitted them. Channel responses need these present so e2e `toBe(0)` gets a
// number, not undefined.
export function ensureChannelDefaults(value: Record<string, unknown>): Record<string, unknown> {
  return { agent_count: 0, member_count: 0, is_member: false, is_archived: false, ...value };
}
