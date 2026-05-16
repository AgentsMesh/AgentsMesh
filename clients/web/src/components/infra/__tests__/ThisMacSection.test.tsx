import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, act } from "@testing-library/react";
import type { UseLocalRunnerOnboarding } from "@/hooks/useLocalRunnerOnboarding";
import type { Runner } from "@/stores/runner";
import { ThisMacSection } from "../ThisMacSection";

const hookReturn: { current: UseLocalRunnerOnboarding } = {
  current: {} as UseLocalRunnerOnboarding,
};
const runnersList: { current: Runner[] } = { current: [] };

vi.mock("@/hooks/useLocalRunnerOnboarding", async () => {
  const actual = await vi.importActual<typeof import("@/hooks/useLocalRunnerOnboarding")>(
    "@/hooks/useLocalRunnerOnboarding",
  );
  return { ...actual, useLocalRunnerOnboarding: () => hookReturn.current };
});

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), back: vi.fn(), forward: vi.fn(), refresh: vi.fn(), prefetch: vi.fn() }),
}));

vi.mock("@/stores/auth", () => ({
  useCurrentOrg: () => ({ slug: "acme" }),
}));

vi.mock("@/stores/runner", async () => {
  const actual = await vi.importActual<typeof import("@/stores/runner")>("@/stores/runner");
  return { ...actual, useRunners: () => runnersList.current };
});

function setHook(overrides: Partial<UseLocalRunnerOnboarding>) {
  hookReturn.current = {
    unsupported: false,
    localNodeId: null,
    isRegistered: false,
    phase: { kind: "idle", status: "not_installed" },
    onRegister: vi.fn(),
    refresh: vi.fn(),
    ...overrides,
  };
}

function makeRunner(node_id: string): Runner {
  return {
    id: 1,
    node_id,
    organization_id: 1,
    status: "online",
    is_enabled: true,
    visibility: "private",
    current_pods: 0,
    max_concurrent_pods: 4,
    registered_by_user_id: 1,
    description: "",
    tags: [],
    host_info: { os: "darwin", arch: "arm64", cpu_cores: 8, memory: 16_000_000_000 },
  } as unknown as Runner;
}

describe("ThisMacSection", () => {
  beforeEach(() => {
    runnersList.current = [];
    setHook({});
  });

  it("renders nothing when service is unsupported (web bundle)", () => {
    setHook({ unsupported: true });
    const { container } = render(<ThisMacSection />);
    expect(container.firstChild).toBeNull();
  });

  // Reverse of TICKET-145: registered+stopped should NOT fall to the
  // onboarding "Register This Mac" button — that made users think they
  // weren't registered when in fact only the service was down.
  it("renders the registered row (not the Register CTA) when service is stopped", () => {
    runnersList.current = [makeRunner("macmini-03")];
    setHook({
      isRegistered: true,
      localNodeId: "macmini-03",
      phase: { kind: "idle", status: "stopped" },
    });
    render(<ThisMacSection />);
    expect(screen.getByText("macmini-03")).toBeDefined();
    expect(screen.getByText(/service stopped/i)).toBeDefined();
    expect(screen.queryByTestId("this-mac-register-btn")).toBeNull();
  });

  it("renders the registered row with 'active' label when service is running", () => {
    runnersList.current = [makeRunner("macmini-03")];
    setHook({
      isRegistered: true,
      localNodeId: "macmini-03",
      phase: { kind: "idle", status: "running" },
    });
    render(<ThisMacSection />);
    expect(screen.getByText(/^active$/i)).toBeDefined();
  });

  it("renders orphaned block when registered locally but backend list excludes it (after grace)", () => {
    // useOrphanGrace holds back the orphan verdict for 30s so a freshly-
    // registered runner doesn't flicker through OrphanedBlock while the
    // local→backend heartbeat catches up. Fast-forward past the grace.
    vi.useFakeTimers();
    try {
      runnersList.current = [makeRunner("other-runner")];
      setHook({
        isRegistered: true,
        localNodeId: "macmini-03",
        phase: { kind: "idle", status: "running" },
      });
      render(<ThisMacSection />);
      // Within grace: still syncing, not yet orphaned.
      expect(screen.queryByTestId("this-mac-orphaned")).toBeNull();
      expect(screen.getByTestId("this-mac-syncing")).toBeDefined();
      act(() => {
        vi.advanceTimersByTime(30_000);
      });
      expect(screen.getByTestId("this-mac-orphaned")).toBeDefined();
      expect(screen.getByTestId("this-mac-reregister-btn")).toBeDefined();
      expect(screen.queryByTestId("this-mac-syncing")).toBeNull();
    } finally {
      vi.useRealTimers();
    }
  });

  it("renders syncing block when registered locally but list is empty (still loading)", () => {
    runnersList.current = [];
    setHook({
      isRegistered: true,
      localNodeId: "macmini-03",
      phase: { kind: "idle", status: "running" },
    });
    render(<ThisMacSection />);
    expect(screen.getByTestId("this-mac-syncing")).toBeDefined();
  });

  it("renders onboarding Register CTA when not registered", () => {
    setHook({
      isRegistered: false,
      localNodeId: null,
      phase: { kind: "idle", status: "not_installed" },
    });
    render(<ThisMacSection />);
    expect(screen.getByTestId("this-mac-register-btn")).toBeDefined();
    expect(screen.getByText(/Run pods locally/i)).toBeDefined();
  });

  it("renders stale-recovery CTA when phase.status is 'stale'", () => {
    setHook({
      isRegistered: false,
      localNodeId: null,
      phase: { kind: "idle", status: "stale" },
    });
    render(<ThisMacSection />);
    expect(screen.getByText(/Old Runner service is installed/i)).toBeDefined();
    expect(screen.getByRole("button", { name: /Re-register This Mac/i })).toBeDefined();
  });
});
