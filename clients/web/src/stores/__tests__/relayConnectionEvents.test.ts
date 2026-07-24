import { beforeEach, describe, expect, it, vi } from "vitest";
import { markServiceReady, registerServiceProvider } from "@agentsmesh/service-runtime";

import { RelayConnectionEvents } from "../relayConnectionEvents";
import { RelayPodRegistry } from "../relayPodRegistry";

type StatusRaw = { status: string; runnerDisconnected: boolean };

const manager = {
  on_status_change: vi.fn().mockResolvedValue(undefined),
  on_acp_message: vi.fn().mockResolvedValue(undefined),
  on_pod_disconnected: vi.fn().mockResolvedValue(undefined),
};

function statusCallback(call = -1): (raw: StatusRaw) => void {
  return manager.on_status_change.mock.calls.at(call)![1] as (raw: StatusRaw) => void;
}

function acpCallback(call = -1): (msgType: number, payload: unknown) => void {
  return manager.on_acp_message.mock.calls.at(call)![1] as (
    msgType: number,
    payload: unknown,
  ) => void;
}

function disconnectCallback(): (podKey: string) => void {
  return manager.on_pod_disconnected.mock.calls[0]![0] as (podKey: string) => void;
}

describe("RelayConnectionEvents", () => {
  let pods: RelayPodRegistry;
  let flush: ReturnType<typeof vi.fn>;
  let events: RelayConnectionEvents;

  beforeEach(() => {
    vi.clearAllMocks();
    registerServiceProvider({ relayManager: manager as never });
    markServiceReady();
    pods = new RelayPodRegistry();
    flush = vi.fn();
    events = new RelayConnectionEvents(pods, flush);
  });

  it("installs one upstream and immediately emits the cached/default status", () => {
    const first = vi.fn();
    const second = vi.fn();

    const offFirst = events.onStatusChange("pod-1", first);
    const offSecond = events.onStatusChange("pod-1", second);

    expect(manager.on_pod_disconnected).toHaveBeenCalledTimes(1);
    expect(manager.on_status_change).toHaveBeenCalledTimes(1);
    expect(first).toHaveBeenCalledWith({ status: "none", runnerDisconnected: false });
    expect(second).toHaveBeenCalledWith({ status: "none", runnerDisconnected: false });

    statusCallback()({ status: "connecting", runnerDisconnected: false });
    expect(first).toHaveBeenLastCalledWith({ status: "connecting", runnerDisconnected: false });
    expect(second).toHaveBeenLastCalledWith({ status: "connecting", runnerDisconnected: false });

    offFirst();
    first.mockClear();
    statusCallback()({ status: "connected", runnerDisconnected: false });
    expect(first).not.toHaveBeenCalled();
    expect(second).toHaveBeenLastCalledWith({ status: "connected", runnerDisconnected: false });
    offSecond();
  });

  it("maps pre-subscription disconnected to none and preserves runner state", () => {
    const listener = vi.fn();
    events.onStatusChange("pod-1", listener);

    statusCallback()({ status: "disconnected", runnerDisconnected: true });

    expect(listener).toHaveBeenLastCalledWith({ status: "none", runnerDisconnected: true });
    expect(events.getStatus("pod-1")).toBe("none");
    expect(events.isConnected("pod-1")).toBe(false);
    expect(events.isRunnerDisconnected("pod-1")).toBe(true);
  });

  it("marks transport ready on connect and revokes readiness otherwise", () => {
    events.ensureStatusUpstream("pod-1");
    const session = pods.getOrCreate("pod-1");
    const token = session.beginSubscription("terminal");
    session.markManagerStarted(token);
    session.markSubscriptionReady(token);

    statusCallback()({ status: "connected", runnerDisconnected: false });
    expect(flush).toHaveBeenCalledExactlyOnceWith("pod-1", session);
    expect(session.scheduleResize({ cols: 80, rows: 24, force: false })).toEqual({
      cols: 80,
      rows: 24,
      force: false,
    });
    expect(events.getStatus("pod-1")).toBe("connected");
    expect(events.isConnected("pod-1")).toBe(true);

    statusCallback()({ status: "disconnected", runnerDisconnected: false });
    expect(events.getStatus("pod-1")).toBe("disconnected");
    expect(session.scheduleResize({ cols: 90, rows: 30, force: false })).toBeUndefined();
  });

  it("creates a session for a connected status received before subscription", () => {
    events.ensureStatusUpstream("pod-1");

    statusCallback()({ status: "connected", runnerDisconnected: false });

    const session = pods.get("pod-1");
    expect(session).toBeDefined();
    expect(flush).toHaveBeenCalledExactlyOnceWith("pod-1", session);
  });

  it("fans ACP messages out until each local listener is removed", () => {
    const first = vi.fn();
    const second = vi.fn();
    const offFirst = events.onAcpMessage("pod-1", first);
    const offSecond = events.onAcpMessage("pod-1", second);

    expect(manager.on_pod_disconnected).toHaveBeenCalledTimes(1);
    expect(manager.on_acp_message).toHaveBeenCalledTimes(1);
    acpCallback()(11, { type: "chunk" });
    expect(first).toHaveBeenCalledWith(11, { type: "chunk" });
    expect(second).toHaveBeenCalledWith(11, { type: "chunk" });

    offFirst();
    offSecond();
    first.mockClear();
    second.mockClear();
    acpCallback()(12, { type: "late" });
    expect(first).not.toHaveBeenCalled();
    expect(second).not.toHaveBeenCalled();
  });

  it("resets status and rebinds live status, ACP, and session owners after teardown", () => {
    const status = vi.fn();
    const acp = vi.fn();
    events.onStatusChange("pod-1", status);
    events.onAcpMessage("pod-1", acp);
    const session = pods.getOrCreate("pod-1");
    session.beginSubscription("next-generation");
    statusCallback()({ status: "connected", runnerDisconnected: false });
    status.mockClear();

    disconnectCallback()("pod-1");

    expect(status).toHaveBeenCalledExactlyOnceWith({
      status: "none",
      runnerDisconnected: false,
    });
    expect(events.getStatus("pod-1")).toBe("none");
    expect(pods.get("pod-1")).toBe(session);
    expect(manager.on_status_change).toHaveBeenCalledTimes(2);
    expect(manager.on_acp_message).toHaveBeenCalledTimes(2);
    expect(manager.on_pod_disconnected).toHaveBeenCalledTimes(1);

    statusCallback()({ status: "connected", runnerDisconnected: false });
    acpCallback()(7, "rebound");
    expect(status).toHaveBeenLastCalledWith({ status: "connected", runnerDisconnected: false });
    expect(acp).toHaveBeenCalledWith(7, "rebound");
  });

  it("rebinds an active session without local listeners", () => {
    events.ensureStatusUpstream("pod-1");
    pods.getOrCreate("pod-1").beginSubscription("terminal");

    disconnectCallback()("pod-1");

    expect(manager.on_status_change).toHaveBeenCalledTimes(2);
  });

  it("does not rebind an idle pod after all local listeners are gone", () => {
    const offStatus = events.onStatusChange("pod-1", vi.fn());
    const offAcp = events.onAcpMessage("pod-1", vi.fn());
    pods.getOrCreate("pod-1");
    offStatus();
    offAcp();

    disconnectCallback()("pod-1");

    expect(pods.get("pod-1")).toBeUndefined();
    expect(manager.on_status_change).toHaveBeenCalledTimes(1);
    expect(manager.on_acp_message).toHaveBeenCalledTimes(1);
  });

  it("returns safe values before a pod emits status", () => {
    expect(events.getStatus("unknown")).toBe("none");
    expect(events.isConnected("unknown")).toBe(false);
    expect(events.isRunnerDisconnected("unknown")).toBe(false);
  });
});
