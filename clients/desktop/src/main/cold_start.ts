import { isValidServerUrl } from "../shared/server-config-types";
import * as serverConfig from "./server_config";

// Cold-start API URL resolution: (1) AGENTSMESH_API_URL env override (dev/e2e,
// process-only, does NOT pollute SSOT) → (2) activeUrl(currentCfg). Returns the
// resolution error (config invalid) rather than mutating module state, so the
// caller owns the pending-dialog decision. After save, serverConfig:set rebinds
// directly, shadowing env.
export function resolveColdStartApiUrl(
  cfg: serverConfig.ServerConfig,
): { apiUrl: string; error: string | null } {
  const envOverride = process.env.AGENTSMESH_API_URL;
  if (envOverride && isValidServerUrl(envOverride)) return { apiUrl: envOverride, error: null };
  try {
    return { apiUrl: serverConfig.activeUrl(cfg), error: null };
  } catch (e) {
    return { apiUrl: serverConfig.activeUrl(serverConfig.DEFAULT), error: (e as Error).message };
  }
}
