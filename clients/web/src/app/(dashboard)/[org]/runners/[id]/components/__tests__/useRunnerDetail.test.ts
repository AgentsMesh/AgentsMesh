import { renderHook, act, waitFor } from "@testing-library/react";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { getPodService, getRunnerService } from "@/lib/wasm-core";
import type { RunnerPodData } from "@/lib/api/runnerTypes";

const mockPush = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
  useParams: () => ({ id: "1", org: "test-org" }),
}));

vi.mock("sonner", () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}));

import { useRunnerDetail } from "../useRunnerDetail";

const mockCreatePod = vi.fn();
const mockFetchRunner = vi.fn();

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
    vi.clearAllMocks();

    vi.mocked(getRunnerService).mockReturnValue({
      fetch_runner: mockFetchRunner,
    } as unknown as ReturnType<typeof getRunnerService>);

    vi.mocked(getPodService).mockReturnValue({
      create_pod: mockCreatePod,
    } as unknown as ReturnType<typeof getPodService>);

    mockFetchRunner.mockResolvedValue(JSON.stringify(baseRunner));
    mockCreatePod.mockResolvedValue(
      JSON.stringify({ pod: { pod_key: "pod-resume-xyz" } })
    );
  });

  it("includes agent_slug from source pod in payload", async () => {
    await renderResumed(makePod({ agent_slug: "claude-code" }));

    expect(mockCreatePod).toHaveBeenCalledTimes(1);
    const payload = JSON.parse(mockCreatePod.mock.calls[0][0]);
    expect(payload.agent_slug).toBe("claude-code");
  });

  it("falls back to empty string when source pod has no agent_slug", async () => {
    await renderResumed(makePod({ agent_slug: undefined }));

    const payload = JSON.parse(mockCreatePod.mock.calls[0][0]);
    expect(payload).toHaveProperty("agent_slug");
    expect(payload.agent_slug).toBe("");
  });

  it("sends complete resume payload shape (PR #340 regression guard)", async () => {
    await renderResumed(makePod({ agent_slug: "aider", pod_key: "pod-source-abc" }));

    const payload = JSON.parse(mockCreatePod.mock.calls[0][0]);
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

    expect(mockPush).toHaveBeenCalledWith("/test-org/workspace?pod=pod-resume-xyz");
  });
});
