import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { useChannelEntryAnchor } from "../useChannelEntryAnchor";
import { getChannelState } from "@/lib/wasm-core";
import type { TransformedMessage } from "../types";

function msgs(...ids: number[]): TransformedMessage[] {
  return ids.map((id) => ({ id, body: "x", messageType: "text", createdAt: "2026-01-01T12:00:00Z" }));
}
const setCursor = (v: number) => vi.mocked(getChannelState().get_last_read_id).mockReturnValue(v);

describe("useChannelEntryAnchor", () => {
  beforeEach(() => vi.mocked(getChannelState().get_last_read_id).mockReset());

  it("unknown cursor requests an unread summary before falling back", async () => {
    setCursor(-1);
    const { result, rerender } = renderHook(
      ({ attempted }) => useChannelEntryAnchor(1, msgs(10, 11, 12), attempted),
      { initialProps: { attempted: false } },
    );
    expect(result.current).toEqual({ firstUnreadId: null, resolved: false, needsUnreadSummary: true });

    rerender({ attempted: true });

    await waitFor(() => {
      expect(result.current).toEqual({ firstUnreadId: null, resolved: true, needsUnreadSummary: false });
    });
  });

  it("re-reads a now-known cursor once the summary fetch is attempted", async () => {
    setCursor(-1);
    const { result, rerender } = renderHook(
      ({ attempted }) => useChannelEntryAnchor(1, msgs(10, 11, 12), attempted),
      { initialProps: { attempted: false } },
    );
    expect(result.current.needsUnreadSummary).toBe(true);

    // fetchUnreadCounts wrote the Rust cursor before resolving; flipping
    // `attempted` re-runs the retry, which reads the now-known cursor.
    setCursor(11);
    rerender({ attempted: true });

    await waitFor(() => {
      expect(result.current).toEqual({ firstUnreadId: 12, resolved: true, needsUnreadSummary: false });
    });
  });

  it("genuine 0 cursor anchors at the first message", () => {
    setCursor(0);
    const { result } = renderHook(() => useChannelEntryAnchor(2, msgs(10, 11, 12)));
    expect(result.current).toEqual({ firstUnreadId: 10, resolved: true, needsUnreadSummary: false });
  });

  it("cursor at 11 anchors at the first message after it", () => {
    setCursor(11);
    const { result } = renderHook(() => useChannelEntryAnchor(3, msgs(10, 11, 12)));
    expect(result.current).toEqual({ firstUnreadId: 12, resolved: true, needsUnreadSummary: false });
  });

  it("fully read cursor has no divider", () => {
    setCursor(12);
    const { result } = renderHook(() => useChannelEntryAnchor(4, msgs(10, 11, 12)));
    expect(result.current).toEqual({ firstUnreadId: null, resolved: true, needsUnreadSummary: false });
  });

  it("freezes the cursor for the current entry", () => {
    setCursor(10);
    const { result, rerender } = renderHook(() => useChannelEntryAnchor(5, msgs(10, 11, 12)));
    expect(result.current.firstUnreadId).toBe(11);

    // A known cursor is frozen for the entry: a later mark-read advancing it
    // must not move the divider.
    setCursor(12);
    rerender();

    expect(result.current.firstUnreadId).toBe(11);
  });
});
