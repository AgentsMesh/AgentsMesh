import { renderHook, act, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import type { RunnerPodData } from "@/lib/api/runnerTypes";

// vi.mock factories are hoisted to the top of the file, before any
// const declarations are executed. To safely reference our shared mock
// fns from inside the factory, we hoist them with vi.hoisted so they
// exist by the time the factory runs.
const h = vi.hoisted(() => ({
  push: vi.fn(),
  createPod: vi.fn(),
  fetchRunner: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: h.push }),
  useParams: () => ({ id: "1", org: "test-org" }),
}));

vi.mock("sonner", () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}));

// Override setup.ts's global wasm-core mock locally. Plain (non-vi.fn)
// factory functions are immune to vi.clearAllMocks(), so the hook always
// sees a working getRunnerService()/getPodService().
vi.mock("@/lib/wasm-core", () => ({
  initWasmCore: vi.fn().mockResolvedValue(undefined),
  isWasmReady: () => true,
  getRunnerService: () => ({ fetch_runner: h.fetchRunner }),
  getPodService: () => ({ create_pod: h.createPod }),
}));

import { useRunnerDetail } from "../useRunnerDetail";

const baseRunner = { id: 42, status: "online", is_enabled: true, relay_connections: [] };

function makePod(overrides: Partial<RunnerPodData> = {}): RunnerPodData {
  return {
    id: 1, pod_key: "pod-source-abc", organization_id: 1, runner_id: 42,
    status: "terminated", agent_status: "idle",
    ...overrides,
  };
}

async function renderResumed(pod: RunnerPodData) {
  const { result } = renderHook(() => useRunnerDetail((k: string) => k));
  await waitFor(
    () => expect(result.current.runner?.id).toBe(42),
    { timeout: 5000 }
  );
  act(() => { result.current.setResumingPod(pod); });
  await act(async () => { await result.current.handleConfirmResume(); });
  return result;
}

describe("useRunnerDetail.handleConfirmResume — resume payload contract", () => {
  beforeEach(() => {
    h.push.mockClear();
    h.fetchRunner.mockReset();
    h.createPod.mockReset();

    h.fetchRunner.mockResolvedValue(JSON.stringify(baseRunner));
    h.createPod.mockResolvedValue(
      JSON.stringify({ pod: { pod_key: "pod-resume-xyz" } })
    );
  });

  it("includes agent_slug from source pod in payload", async () => {
    await renderResumed(makePod({ agent_slug: "claude-code" }));

    expect(h.createPod).toHaveBeenCalledTimes(1);
    const payload = JSON.parse(h.createPod.mock.calls[0][0]);
    expect(payload.agent_slug).toBe("claude-code");
  });

  it("falls back to empty string when source pod has no agent_slug", async () => {
    await renderResumed(makePod({ agent_slug: undefined }));

    const payload = JSON.parse(h.createPod.mock.calls[0][0]);
    expect(payload).toHaveProperty("agent_slug");
    expect(payload.agent_slug).toBe("");
  });

  it("sends complete resume payload shape (PR #340 regression guard)", async () => {
    await renderResumed(makePod({ agent_slug: "aider", pod_key: "pod-source-abc" }));

    const payload = JSON.parse(h.createPod.mock.calls[0][0]);
    expect(payload).toMatchObject({
      agent_slug: "aider",
      runner_id: 42,
      source_pod_key: "pod-source-abc",
      resume_agent_session: true,
    });
    expect(typeof payload.cols).toBe("number");
    expect(typeof payload.rows).toBe("number");
  });

  it("navigates to new pod workspace on success", async () => {
    await renderResumed(makePod({ agent_slug: "claude-code" }));

    expect(h.push).toHaveBeenCalledWith("/test-org/workspace?pod=pod-resume-xyz");
  });
});
