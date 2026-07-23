import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { useChannelEntryMarkRead } from "../useChannelEntryMarkRead";

describe("useChannelEntryMarkRead", () => {
  beforeEach(() => vi.useFakeTimers());
  afterEach(() => vi.useRealTimers());

  it("marks read after the debounce once the anchor is resolved", () => {
    const markRead = vi.fn().mockResolvedValue(undefined);
    renderHook(() =>
      useChannelEntryMarkRead({ channelId: 1, lastMessageId: 99, entryAnchorResolved: true, markRead }),
    );

    expect(markRead).not.toHaveBeenCalled();
    vi.advanceTimersByTime(300);
    expect(markRead).toHaveBeenCalledWith(1, 99);
  });

  it("flushes on unmount even if the anchor never resolved (unknown cursor, quick leave)", () => {
    const markRead = vi.fn().mockResolvedValue(undefined);
    const { unmount } = renderHook(() =>
      useChannelEntryMarkRead({ channelId: 1, lastMessageId: 99, entryAnchorResolved: false, markRead }),
    );

    // Left before fetchUnreadCounts resolved: the debounce has not even fired,
    // but the pending mark-read must still flush so the channel is marked read.
    unmount();
    expect(markRead).toHaveBeenCalledWith(1, 99);
  });

  it("defers the cursor advance until the anchor resolves, then flushes", () => {
    const markRead = vi.fn().mockResolvedValue(undefined);
    const { rerender } = renderHook(
      ({ resolved }) =>
        useChannelEntryMarkRead({ channelId: 1, lastMessageId: 99, entryAnchorResolved: resolved, markRead }),
      { initialProps: { resolved: false } },
    );

    // Debounce elapses while still unresolved → must NOT advance the cursor yet.
    vi.advanceTimersByTime(300);
    expect(markRead).not.toHaveBeenCalled();

    // Anchor resolves → the parked pending flushes immediately.
    rerender({ resolved: true });
    expect(markRead).toHaveBeenCalledWith(1, 99);
  });

  it("does not double-mark when it resolves before the debounce fires", () => {
    const markRead = vi.fn().mockResolvedValue(undefined);
    const { rerender } = renderHook(
      ({ resolved }) =>
        useChannelEntryMarkRead({ channelId: 1, lastMessageId: 99, entryAnchorResolved: resolved, markRead }),
      { initialProps: { resolved: false } },
    );

    // Resolves mid-debounce (before 300ms): the resolve effect must not flush
    // early, and the timer must flush exactly once.
    rerender({ resolved: true });
    expect(markRead).not.toHaveBeenCalled();
    vi.advanceTimersByTime(300);
    expect(markRead).toHaveBeenCalledTimes(1);
    expect(markRead).toHaveBeenCalledWith(1, 99);
  });
});
