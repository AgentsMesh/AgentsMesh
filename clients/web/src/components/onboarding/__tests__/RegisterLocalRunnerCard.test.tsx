import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import type { UseLocalRunnerOnboarding } from "@/hooks/useLocalRunnerOnboarding";
import { RegisterLocalRunnerCard } from "../RegisterLocalRunnerCard";

const hookReturn: { current: UseLocalRunnerOnboarding } = {
  current: {} as UseLocalRunnerOnboarding,
};

vi.mock("@/hooks/useLocalRunnerOnboarding", async () => {
  const actual = await vi.importActual<typeof import("@/hooks/useLocalRunnerOnboarding")>(
    "@/hooks/useLocalRunnerOnboarding",
  );
  return {
    ...actual,
    useLocalRunnerOnboarding: () => hookReturn.current,
  };
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

describe("RegisterLocalRunnerCard", () => {
  beforeEach(() => {
    setHook({});
  });

  it("renders nothing when service is unsupported (web bundle)", () => {
    setHook({ unsupported: true });
    const { container } = render(<RegisterLocalRunnerCard />);
    expect(container.firstChild).toBeNull();
  });

  // TICKET-145 regression: must not claim "registered" based on service status.
  it("does NOT claim registered when service is running but no localNodeId", () => {
    setHook({
      isRegistered: false,
      localNodeId: null,
      phase: { kind: "idle", status: "running" },
    });
    render(<RegisterLocalRunnerCard />);
    expect(screen.queryByText(/This Mac is registered as a Runner/i)).toBeNull();
    expect(screen.getByRole("button", { name: /register/i })).toBeDefined();
  });

  it("claims registered when localNodeId is set + service running", () => {
    setHook({
      isRegistered: true,
      localNodeId: "macmini-03",
      phase: { kind: "idle", status: "running" },
    });
    render(<RegisterLocalRunnerCard />);
    expect(screen.getByText(/This Mac is registered as a Runner/i)).toBeDefined();
    expect(screen.getByText(/Pods will run locally/i)).toBeDefined();
  });

  it("claims registered but warns when service is stopped", () => {
    setHook({
      isRegistered: true,
      localNodeId: "macmini-03",
      phase: { kind: "idle", status: "stopped" },
    });
    render(<RegisterLocalRunnerCard />);
    expect(screen.getByText(/This Mac is registered as a Runner/i)).toBeDefined();
    expect(screen.getByText(/Service is not running/i)).toBeDefined();
  });

  it("shows stale-service recovery CTA when phase.status is 'stale'", () => {
    setHook({
      isRegistered: false,
      localNodeId: null,
      phase: { kind: "idle", status: "stale" },
    });
    render(<RegisterLocalRunnerCard />);
    expect(screen.getByText(/Stale Runner service detected/i)).toBeDefined();
    expect(screen.getByRole("button", { name: /Re-register/i })).toBeDefined();
  });
});
