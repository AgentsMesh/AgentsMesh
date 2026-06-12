import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { dispatchLoopalRelayEvent } from "@/stores/loopalDispatcher";
import { MsgType } from "@/stores/relayProtocol";
import { useLoopalConsoleStore } from "@/stores/loopalConsole";

describe("dispatchLoopalRelayEvent", () => {
  const dispatchEvent = vi.fn();
  const dispatchSnapshot = vi.fn();

  beforeEach(() => {
    dispatchEvent.mockClear();
    dispatchSnapshot.mockClear();
    vi.spyOn(useLoopalConsoleStore, "getState").mockReturnValue({
      dispatchEvent,
      dispatchSnapshot,
    } as unknown as ReturnType<typeof useLoopalConsoleStore.getState>);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("routes loopal.* incremental events to dispatchEvent", () => {
    const handled = dispatchLoopalRelayEvent("pod1", MsgType.AcpEvent, {
      type: "loopal.bgTask.spawned",
      id: "bg1",
    });
    expect(handled).toBe(true);
    expect(dispatchEvent).toHaveBeenCalledWith("pod1", "loopal.bgTask.spawned", expect.anything());
    expect(dispatchSnapshot).not.toHaveBeenCalled();
  });

  it("routes loopal.snapshot to dispatchSnapshot", () => {
    const handled = dispatchLoopalRelayEvent("pod1", MsgType.AcpEvent, {
      type: "loopal.snapshot",
      bg_tasks: [],
    });
    expect(handled).toBe(true);
    expect(dispatchSnapshot).toHaveBeenCalled();
    expect(dispatchEvent).not.toHaveBeenCalled();
  });

  it("ignores standard ACP events (non-loopal type)", () => {
    const handled = dispatchLoopalRelayEvent("pod1", MsgType.AcpEvent, {
      type: "contentChunk",
      text: "hi",
    });
    expect(handled).toBe(false);
    expect(dispatchEvent).not.toHaveBeenCalled();
  });

  it("ignores non-AcpEvent message types", () => {
    const handled = dispatchLoopalRelayEvent("pod1", MsgType.Output, {
      type: "loopal.crons",
    });
    expect(handled).toBe(false);
  });

  it("ignores payloads without a string type", () => {
    expect(dispatchLoopalRelayEvent("pod1", MsgType.AcpEvent, {})).toBe(false);
    expect(dispatchLoopalRelayEvent("pod1", MsgType.AcpEvent, null)).toBe(false);
  });
});
