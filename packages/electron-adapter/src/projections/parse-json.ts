// Shared safe JSON-object parse for proto JSON-string fields (config_overrides,
// prompt_variables, autopilot_config, ...). Mirrors parseJSONObject in
// clients/web/src/lib/api/connect/loopConnect.ts; collapses the copies that
// were inlined per-domain.
export function parseJSONObject(s: string): Record<string, unknown> | undefined {
  if (!s) return undefined;
  try {
    const v = JSON.parse(s);
    return typeof v === "object" && v !== null ? (v as Record<string, unknown>) : undefined;
  } catch {
    return undefined;
  }
}

// Typed variant for JSON-string fields whose shape is known (e.g. runner
// host_info). parseJSONObject stays for the Record<string, unknown> fields
// (config_overrides / prompt_variables / autopilot_config) that need the
// object-shape guard.
export function parseJSON<T>(s: string): T | undefined {
  if (!s) return undefined;
  try {
    return JSON.parse(s) as T;
  } catch {
    return undefined;
  }
}
