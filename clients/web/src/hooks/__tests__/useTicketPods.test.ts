import { renderHook, waitFor, act } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { create, toBinary } from "@bufbuild/protobuf";
import { ReplaceCachedPodsRequestSchema } from "@proto/pod_state/v1/pod_state_pb";
import { PodSchema } from "@proto/pod/v1/pod_pb";

import {
  useTicketPods,
  invalidateTicketPods,
  __resetTicketPodsCacheForTests,
} from "../useTicketPods";

const podsBySlug = new Map<string, { pod_key: string; status?: string }[]>();
const stateMirror = new Map<string, string>();
let pendingFetch: ((_: void) => void) | null = null;
const getTicketPodsMock = vi.fn(async (slug: string) => {
  if (pendingFetch) {
    await new Promise<void>((resolve) => {
      pendingFetch = resolve;
    });
  }
  return JSON.stringify({ pods: podsBySlug.get(slug) ?? [] });
});

vi.mock("@/lib/wasm-core", async () => {
  const actual = await vi.importActual<typeof import("@/lib/wasm-core")>("@/lib/wasm-core");
  return {
    ...actual,
    getTicketService: () => ({
      get_ticket_pods: getTicketPodsMock,
    }),
    // ticket→pods cache now lives on runtime.state (WasmTicketState): the
    // fetch mirrors via set_ticket_pods, the sync read goes through ticket_pods_bytes.
    getTicketState: () => ({
      ticket_pods_bytes: (slug: string) => {
        const pods = JSON.parse(stateMirror.get(slug) ?? "[]") as { pod_key: string; status?: string }[];
        return toBinary(ReplaceCachedPodsRequestSchema, create(ReplaceCachedPodsRequestSchema, {
          pods: pods.map((p) => create(PodSchema, { podKey: p.pod_key, status: p.status ?? "" })),
        }));
      },
      set_ticket_pods: (slug: string, podsJson: string) => {
        stateMirror.set(slug, podsJson);
      },
    }),
  };
});

function seed(slug: string, pods: { pod_key: string; status?: string }[]) {
  podsBySlug.set(slug, pods);
}

describe("useTicketPods", () => {
  beforeEach(() => {
    podsBySlug.clear();
    stateMirror.clear();
    pendingFetch = null;
    getTicketPodsMock.mockClear();
    __resetTicketPodsCacheForTests();
  });

  afterEach(() => {
    __resetTicketPodsCacheForTests();
    podsBySlug.clear();
  });

  it("fetches once and shares the result across subscribers", async () => {
    seed("T-1", [{ pod_key: "a", status: "running" }]);

    const a = renderHook(() => useTicketPods("T-1"));
    const b = renderHook(() => useTicketPods("T-1"));

    await waitFor(() => {
      expect(a.result.current.pods).toHaveLength(1);
      expect(b.result.current.pods).toHaveLength(1);
    });

    expect(getTicketPodsMock).toHaveBeenCalledTimes(1);
    expect(getTicketPodsMock).toHaveBeenCalledWith("T-1", true);
  });

  it("deduplicates in-flight calls when the hook is mounted twice rapidly", async () => {
    seed("T-2", [{ pod_key: "x", status: "running" }]);
    pendingFetch = () => undefined;

    renderHook(() => useTicketPods("T-2"));
    renderHook(() => useTicketPods("T-2"));

    expect(getTicketPodsMock).toHaveBeenCalledTimes(1);
    const release = pendingFetch;
    pendingFetch = null;
    release?.();
  });

  it("re-reads Rust state after invalidate + refresh", async () => {
    seed("T-4", [{ pod_key: "a" }]);

    const { result } = renderHook(() => useTicketPods("T-4"));
    await waitFor(() => expect(result.current.pods).toHaveLength(1));

    seed("T-4", [{ pod_key: "a" }, { pod_key: "b" }]);
    invalidateTicketPods("T-4");
    await act(async () => {
      await result.current.refresh();
    });
    await waitFor(() => expect(result.current.pods).toHaveLength(2));
    expect(getTicketPodsMock).toHaveBeenCalledTimes(2);
  });

  it("returns empty list when ticketSlug is null", () => {
    const { result } = renderHook(() => useTicketPods(null));
    expect(result.current.pods).toEqual([]);
    expect(getTicketPodsMock).not.toHaveBeenCalled();
  });
});
