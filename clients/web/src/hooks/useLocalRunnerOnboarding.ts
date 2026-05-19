"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { getLocalRunnerService, getRunnerService } from "@agentsmesh/service-runtime";
import { getApiClient } from "@/lib/wasm-core";
import type { ILocalRunnerService, LocalRunnerStatus } from "@agentsmesh/service-interface";

export type StepKey =
  | "install"
  | "token"
  | "register"
  | "service_install"
  | "service_start";

export const STEP_LABELS: Record<StepKey, string> = {
  install: "Installing runner binary",
  token: "Minting registration token",
  register: "Registering with backend",
  service_install: "Installing OS service",
  service_start: "Starting service",
};

export type Phase =
  | { kind: "loading" }
  | { kind: "idle"; status: LocalRunnerStatus }
  | { kind: "installing"; step: StepKey }
  | { kind: "error"; status: LocalRunnerStatus; step: StepKey | null; message: string };

const STATUS_REFRESH_MS = 30_000;

type ReleaseInfo = { version: string; sha256?: Record<string, string> };

async function fetchReleaseInfo(svc: ILocalRunnerService): Promise<ReleaseInfo> {
  try {
    const raw = await getApiClient().get("/api/v1/runners/latest-release");
    const data = JSON.parse(raw);
    if (typeof data?.version === "string" && data.version) {
      return { version: data.version, sha256: data.sha256 };
    }
  } catch (err) {
    console.warn("[LocalRunner] backend latest-release unavailable, using bundled fallback", err);
  }
  return { version: await svc.fallback_version() };
}

async function buildArchiveSpec(
  svc: ILocalRunnerService,
): Promise<{ url: string; sha256: string | null } | null> {
  const target = await svc.host_target();
  if (!target) return null;
  const ext = target.startsWith("windows_") ? "zip" : "tar.gz";
  const release = await fetchReleaseInfo(svc);
  const filename = `agentsmesh-runner_${release.version}_${target}.${ext}`;
  const url = `https://github.com/AgentsMesh/AgentsMesh/releases/download/v${release.version}/${filename}`;
  const sha256 = release.sha256?.[`${target}.${ext}`] ?? release.sha256?.[target] ?? null;
  return { url, sha256 };
}

export interface UseLocalRunnerOnboarding {
  unsupported: boolean;
  localNodeId: string | null;
  // TICKET-145: derive from ~/.agentsmesh/config.yaml node_id, NOT phase.status —
  // a stale launchd job can falsely report Running after user removed the config.
  isRegistered: boolean;
  phase: Phase;
  onRegister: () => Promise<void>;
  refresh: () => Promise<void>;
}

export function useLocalRunnerOnboarding(): UseLocalRunnerOnboarding {
  const svc = getLocalRunnerService() as ILocalRunnerService | undefined;
  const [phase, setPhase] = useState<Phase>(svc ? { kind: "loading" } : { kind: "idle", status: "not_installed" });
  const [localNodeId, setLocalNodeId] = useState<string | null>(null);
  const phaseRef = useRef(phase);
  phaseRef.current = phase;

  const refresh = useCallback(async () => {
    if (!svc) return;
    try {
      const [status, nodeId] = await Promise.all([
        svc.service_status(),
        svc.local_node_id(),
      ]);
      setLocalNodeId(nodeId);
      setPhase({ kind: "idle", status });
    } catch (err) {
      // Defense-in-depth against IPC layers that surface errors as rejections.
      console.warn("[LocalRunner] service_status failed; treating as not_installed", err);
      setPhase({ kind: "idle", status: "not_installed" });
    }
  }, [svc]);

  useEffect(() => {
    if (!svc) return;
    void refresh();
    const id = window.setInterval(() => {
      if (phaseRef.current.kind === "installing") return;
      void refresh();
    }, STATUS_REFRESH_MS);
    return () => window.clearInterval(id);
  }, [svc, refresh]);

  const onRegister = useCallback(async () => {
    if (!svc) return;
    let currentStep: StepKey = "install";
    try {
      if (!(await svc.is_installed())) {
        currentStep = "install";
        setPhase({ kind: "installing", step: currentStep });
        const spec = await buildArchiveSpec(svc);
        if (!spec) throw new Error("unsupported platform for runner auto-install");
        await svc.install_binary(spec.url, spec.sha256);
      }

      if (!(await svc.is_registered())) {
        currentStep = "token";
        setPhase({ kind: "installing", step: currentStep });
        const tokenResp = JSON.parse(
          await getRunnerService().create_token(JSON.stringify({ name: "Desktop" })),
        );
        const token: string | undefined = tokenResp?.token;
        if (!token) throw new Error("backend returned empty registration token");

        currentStep = "register";
        setPhase({ kind: "installing", step: currentStep });
        await svc.register(token);
      }

      const status = await svc.service_status();
      if (status === "not_installed") {
        currentStep = "service_install";
        setPhase({ kind: "installing", step: currentStep });
        await svc.service_install();
      }

      if (status !== "running") {
        currentStep = "service_start";
        setPhase({ kind: "installing", step: currentStep });
        await svc.service_start();
      }

      await refresh();
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      const status = await svc.service_status().catch(() => "unknown" as const);
      const nodeId = await svc.local_node_id().catch(() => null);
      setLocalNodeId(nodeId);
      setPhase({ kind: "error", status, step: currentStep, message });
    }
  }, [svc, refresh]);

  return { unsupported: !svc, localNodeId, isRegistered: localNodeId !== null, phase, onRegister, refresh };
}
