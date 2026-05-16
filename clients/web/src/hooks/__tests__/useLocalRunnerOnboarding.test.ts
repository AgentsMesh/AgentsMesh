import { renderHook, act, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { useLocalRunnerOnboarding } from "../useLocalRunnerOnboarding";

type Status = "running" | "stopped" | "unknown" | "not_installed" | "stale";

interface FakeService {
  state: {
    installed: boolean;
    registered: boolean;
    serviceInstalled: boolean;
    running: boolean;
    nodeId: string | null;
    failAtStep?: "install" | "register" | "service_install" | "service_start";
  };
  calls: {
    install_binary: Array<{ url: string; sha256: string | null }>;
    register: Array<{ token: string }>;
    service_install: number;
    service_start: number;
  };
  fallback_version: () => Promise<string>;
  host_target: () => Promise<string>;
  is_installed: () => Promise<boolean>;
  is_registered: () => Promise<boolean>;
  local_node_id: () => Promise<string | null>;
  service_status: () => Promise<Status>;
  install_binary: (url: string, sha256: string | null) => Promise<void>;
  register: (token: string) => Promise<void>;
  service_install: () => Promise<void>;
  service_start: () => Promise<void>;
}

let svc: FakeService | undefined;
let createTokenResp: { token: string; id: number } | null;
let latestReleaseResp: { version: string; sha256?: Record<string, string> } | null;

function makeService(): FakeService {
  const state = {
    installed: false,
    registered: false,
    serviceInstalled: false,
    running: false,
    nodeId: null as string | null,
    failAtStep: undefined as FakeService["state"]["failAtStep"],
  };
  const calls = {
    install_binary: [] as Array<{ url: string; sha256: string | null }>,
    register: [] as Array<{ token: string }>,
    service_install: 0,
    service_start: 0,
  };
  return {
    state,
    calls,
    fallback_version: async () => "0.29.0",
    host_target: async () => "darwin_arm64",
    is_installed: async () => state.installed,
    is_registered: async () => state.registered,
    local_node_id: async () => state.nodeId,
    service_status: async () => {
      if (!state.installed) return "not_installed";
      if (state.running) return "running";
      if (state.serviceInstalled) return "stopped";
      return "not_installed";
    },
    install_binary: async (url, sha256) => {
      calls.install_binary.push({ url, sha256 });
      if (state.failAtStep === "install") throw new Error("install boom");
      state.installed = true;
    },
    register: async (token) => {
      calls.register.push({ token });
      if (state.failAtStep === "register") throw new Error("register boom");
      state.registered = true;
      state.nodeId = "test-mac";
    },
    service_install: async () => {
      calls.service_install++;
      if (state.failAtStep === "service_install") throw new Error("service_install boom");
      state.serviceInstalled = true;
    },
    service_start: async () => {
      calls.service_start++;
      if (state.failAtStep === "service_start") throw new Error("service_start boom");
      state.running = true;
    },
  };
}

vi.mock("@agentsmesh/service-runtime", () => ({
  getLocalRunnerService: () => svc,
  getRunnerService: () => ({
    create_token: vi.fn(async () => JSON.stringify(createTokenResp ?? { token: "tok-xyz", id: 1 })),
  }),
}));

vi.mock("@/lib/wasm-core", () => ({
  getApiClient: () => ({
    get: vi.fn(async () => JSON.stringify(latestReleaseResp ?? { version: "0.29.0", sha256: {} })),
  }),
}));

describe("useLocalRunnerOnboarding", () => {
  beforeEach(() => {
    svc = makeService();
    createTokenResp = null;
    latestReleaseResp = null;
  });

  afterEach(() => {
    svc = undefined;
  });

  it("reports unsupported when service is unavailable (web bundle)", () => {
    svc = undefined;
    const { result } = renderHook(() => useLocalRunnerOnboarding());
    expect(result.current.unsupported).toBe(true);
    expect(result.current.phase).toEqual({ kind: "idle", status: "not_installed" });
  });

  it("starts in loading and resolves to idle.not_installed on mount", async () => {
    const { result } = renderHook(() => useLocalRunnerOnboarding());
    expect(result.current.phase.kind).toBe("loading");
    await waitFor(() => expect(result.current.phase).toEqual({ kind: "idle", status: "not_installed" }));
  });

  it("runs full onboarding flow from fresh state", async () => {
    const { result } = renderHook(() => useLocalRunnerOnboarding());
    await waitFor(() => expect(result.current.phase.kind).toBe("idle"));

    await act(async () => { await result.current.onRegister(); });

    expect(svc!.calls.install_binary).toHaveLength(1);
    expect(svc!.calls.register).toHaveLength(1);
    expect(svc!.calls.service_install).toBe(1);
    expect(svc!.calls.service_start).toBe(1);
    expect(result.current.phase).toEqual({ kind: "idle", status: "running" });
    expect(result.current.localNodeId).toBe("test-mac");
  });

  // Regression #1 + #2: GitHub URL must point to AgentsMesh/AgentsMesh
  // (not anthropics/agentsmesh) and use the version returned by the
  // backend's latest-release endpoint.
  it("builds GitHub release URL with correct owner + version", async () => {
    latestReleaseResp = { version: "1.2.3" };
    const { result } = renderHook(() => useLocalRunnerOnboarding());
    await waitFor(() => expect(result.current.phase.kind).toBe("idle"));
    await act(async () => { await result.current.onRegister(); });

    const { url } = svc!.calls.install_binary[0];
    expect(url).toContain("github.com/AgentsMesh/AgentsMesh");
    expect(url).not.toContain("anthropics");
    expect(url).toContain("v1.2.3");
    expect(url).toContain("agentsmesh-runner_1.2.3_darwin_arm64.tar.gz");
  });

  it("falls back to crate version when backend latest-release is unreachable", async () => {
    latestReleaseResp = { version: "" };
    const { result } = renderHook(() => useLocalRunnerOnboarding());
    await waitFor(() => expect(result.current.phase.kind).toBe("idle"));
    await act(async () => { await result.current.onRegister(); });
    expect(svc!.calls.install_binary[0].url).toContain("v0.29.0");
  });

  it("skips install_binary when binary already exists (idempotent retry)", async () => {
    svc!.state.installed = true;
    const { result } = renderHook(() => useLocalRunnerOnboarding());
    await waitFor(() => expect(result.current.phase.kind).toBe("idle"));
    await act(async () => { await result.current.onRegister(); });
    expect(svc!.calls.install_binary).toHaveLength(0);
    expect(svc!.calls.register).toHaveLength(1);
  });

  it("skips token + register when already registered", async () => {
    svc!.state.installed = true;
    svc!.state.registered = true;
    svc!.state.nodeId = "existing-mac";
    const { result } = renderHook(() => useLocalRunnerOnboarding());
    await waitFor(() => expect(result.current.phase.kind).toBe("idle"));
    await act(async () => { await result.current.onRegister(); });
    expect(svc!.calls.register).toHaveLength(0);
    expect(svc!.calls.service_install).toBe(1);
    expect(svc!.calls.service_start).toBe(1);
  });

  it("skips service_start when already running", async () => {
    svc!.state.installed = true;
    svc!.state.registered = true;
    svc!.state.serviceInstalled = true;
    svc!.state.running = true;
    const { result } = renderHook(() => useLocalRunnerOnboarding());
    await waitFor(() => expect(result.current.phase.kind).toBe("idle"));
    await act(async () => { await result.current.onRegister(); });
    expect(svc!.calls.service_install).toBe(0);
    expect(svc!.calls.service_start).toBe(0);
    expect(result.current.phase).toEqual({ kind: "idle", status: "running" });
  });

  it("reports step-aware error when a step throws", async () => {
    svc!.state.failAtStep = "service_start";
    const { result } = renderHook(() => useLocalRunnerOnboarding());
    await waitFor(() => expect(result.current.phase.kind).toBe("idle"));
    await act(async () => { await result.current.onRegister(); });
    expect(result.current.phase.kind).toBe("error");
    if (result.current.phase.kind === "error") {
      expect(result.current.phase.step).toBe("service_start");
      expect(result.current.phase.message).toContain("service_start boom");
    }
  });

  // Regression #5: refresh() must update phase even mid-onboarding.
  // Earlier the hook had `if (phaseRef.current.kind === "installing") return`
  // *inside* refresh, which made onRegister's terminal `await refresh()` a
  // no-op and pinned the UI in "Working… / Starting service…" forever even
  // after every step succeeded.
  it("flips phase from installing to idle.running at end of onRegister", async () => {
    const { result } = renderHook(() => useLocalRunnerOnboarding());
    await waitFor(() => expect(result.current.phase.kind).toBe("idle"));
    await act(async () => { await result.current.onRegister(); });
    expect(result.current.phase.kind).toBe("idle");
    expect(result.current.phase).toEqual({ kind: "idle", status: "running" });
  });

  // Regression for TICKET-145: isRegistered is the SSOT for "this Mac is
  // registered as a runner" — it must be derived from localNodeId (i.e.
  // ~/.agentsmesh/config.yaml exists with node_id) NOT from service status.
  // Earlier the card read phase.status === "running" directly and a stale
  // launchd job from a previous registration falsely claimed registration.
  it("isRegistered is false when localNodeId is null, regardless of service status", async () => {
    svc!.state.installed = true;
    svc!.state.running = true;
    svc!.state.registered = false;
    svc!.state.nodeId = null;
    const { result } = renderHook(() => useLocalRunnerOnboarding());
    await waitFor(() => expect(result.current.phase.kind).toBe("idle"));
    expect(result.current.isRegistered).toBe(false);
  });

  it("isRegistered is true once localNodeId is populated", async () => {
    svc!.state.installed = true;
    svc!.state.registered = true;
    svc!.state.nodeId = "macmini-03";
    svc!.state.running = false;
    svc!.state.serviceInstalled = true;
    const { result } = renderHook(() => useLocalRunnerOnboarding());
    await waitFor(() => expect(result.current.phase.kind).toBe("idle"));
    expect(result.current.isRegistered).toBe(true);
    expect(result.current.localNodeId).toBe("macmini-03");
    expect(result.current.phase).toEqual({ kind: "idle", status: "stopped" });
  });

  it("surfaces the new 'stale' service status from the IPC contract", async () => {
    const fake = makeService();
    fake.service_status = async () => "stale" as Status;
    svc = fake;
    const { result } = renderHook(() => useLocalRunnerOnboarding());
    await waitFor(() => expect(result.current.phase.kind).toBe("idle"));
    expect(result.current.phase).toEqual({ kind: "idle", status: "stale" });
    expect(result.current.isRegistered).toBe(false);
  });
});
