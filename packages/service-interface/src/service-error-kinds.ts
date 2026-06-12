// Mirror of the Rust SSOT `agentsmesh-types::ServiceError` discriminant tags
// (clients/core/crates/types/src/service_error.rs, serde `rename_all =
// "snake_case"`). Single TS copy — both the web parse layer and the
// electron-adapter IPC unwrapper validate against this set; adding a kind in
// Rust means updating exactly this list on the TS side.
export const SERVICE_ERROR_KINDS = [
  "http",
  "auth_expired",
  "network",
  "invalid_json",
  "resource_not_found",
  "unknown",
] as const;

export type ServiceErrorKind = (typeof SERVICE_ERROR_KINDS)[number];

export const SERVICE_ERROR_KIND_SET: ReadonlySet<string> = new Set(SERVICE_ERROR_KINDS);
