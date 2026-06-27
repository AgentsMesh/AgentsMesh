import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook } from "@testing-library/react";
import { useChannelEntryAnchor } from "../useChannelEntryAnchor";
import { getChannelState } from "@/lib/wasm-core";
import type { TransformedMessage } from "../types";

function msgs(...ids: number[]): TransformedMessage[] {
  return ids.map((id) => ({ id, body: "x", messageType: "text", createdAt: "2026-01-01T12:00:00Z" }));
}
const setCursor = (v: number) => vi.mocked(getChannelState().get_last_read_id).mockReturnValue(v);

describe("useChannelEntryAnchor", () => {
  beforeEach(() => vi.mocked(getChannelState().get_last_read_id).mockReset());

  it("no known cursor (-1) → no divider", () => {
    setCursor(-1);
    const { result } = renderHook(() => useChannelEntryAnchor(1, msgs(10, 11, 12)));
    expect(result.current).toBeNull();
  });

  it("genuine 0 cursor → divider anchors at the first (all-unread) message", () => {
    setCursor(0);
    const { result } = renderHook(() => useChannelEntryAnchor(2, msgs(10, 11, 12)));
    expect(result.current).toBe(10);
  });

  it("cursor at 11 → divider at the first message after it", () => {
    setCursor(11);
    const { result } = renderHook(() => useChannelEntryAnchor(3, msgs(10, 11, 12)));
    expect(result.current).toBe(12);
  });

  it("fully read (cursor at latest) → no divider", () => {
    setCursor(12);
    const { result } = renderHook(() => useChannelEntryAnchor(4, msgs(10, 11, 12)));
    expect(result.current).toBeNull();
  });
});
