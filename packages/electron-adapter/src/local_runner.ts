import { invoke } from "./invoke";
import type {
  ILocalRunnerService,
  LocalRunnerStatus,
} from "@agentsmesh/service-interface";

const VALID_STATUSES = new Set<LocalRunnerStatus>([
  "running",
  "stopped",
  "unknown",
  "not_installed",
  "stale",
]);

export class ElectronLocalRunnerService implements ILocalRunnerService {
  binary_path(): Promise<string> {
    return invoke<string>("localRunnerBinaryPath");
  }

  host_target(): Promise<string | null> {
    return invoke<string | null>("localRunnerHostTarget");
  }

  fallback_version(): Promise<string> {
    return invoke<string>("localRunnerFallbackVersion");
  }

  is_installed(): Promise<boolean> {
    return invoke<boolean>("localRunnerIsInstalled");
  }

  installed_version(): Promise<string | null> {
    return invoke<string | null>("localRunnerInstalledVersion");
  }

  install_binary(release_url: string, expected_sha256?: string | null): Promise<void> {
    return invoke<void>("localRunnerInstallBinary", release_url, expected_sha256 ?? null);
  }

  is_registered(): Promise<boolean> {
    return invoke<boolean>("localRunnerIsRegistered");
  }

  local_node_id(): Promise<string | null> {
    return invoke<string | null>("localRunnerLocalNodeId");
  }

  register(token: string): Promise<void> {
    return invoke<void>("localRunnerRegister", token);
  }

  service_install(): Promise<void> {
    return invoke<void>("localRunnerServiceInstall");
  }

  service_uninstall(): Promise<void> {
    return invoke<void>("localRunnerServiceUninstall");
  }

  service_start(): Promise<void> {
    return invoke<void>("localRunnerServiceStart");
  }

  service_stop(): Promise<void> {
    return invoke<void>("localRunnerServiceStop");
  }

  async service_status(): Promise<LocalRunnerStatus> {
    const raw = await invoke<string>("localRunnerServiceStatus");
    return VALID_STATUSES.has(raw as LocalRunnerStatus)
      ? (raw as LocalRunnerStatus)
      : "unknown";
  }
}
