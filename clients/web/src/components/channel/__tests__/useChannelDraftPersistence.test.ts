import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useChannelDraftPersistence } from "../useChannelDraftPersistence";
import { getChannelDraft, setChannelDraft, clearChannelDraft } from "@/stores/useChannelDraft";

describe("useChannelDraftPersistence", () => {
  beforeEach(() => {
    [1, 2, 3, 10, 20].forEach(clearChannelDraft);
    vi.useRealTimers();
  });

  it("restores the channel's draft into the composer on mount", () => {
    setChannelDraft(1, "saved draft");
    const setContent = vi.fn();
    renderHook(() => useChannelDraftPersistence(1, "", setContent));
    expect(setContent).toHaveBeenCalledWith("saved draft");
  });

  // Regression: React StrictMode's mount→cleanup→mount ran the unmount-save with
  // a stale empty contentRef, deleting the draft. Unmount must never delete.
  it("does NOT delete a saved draft when unmounting with empty content", () => {
    setChannelDraft(1, "saved draft");
    const { unmount } = renderHook(() => useChannelDraftPersistence(1, "", vi.fn()));
    unmount();
    expect(getChannelDraft(1)).toBe("saved draft");
  });

  it("persists a non-empty draft on unmount", () => {
    const { unmount } = renderHook(() => useChannelDraftPersistence(2, "typed but unsent", vi.fn()));
    unmount();
    expect(getChannelDraft(2)).toBe("typed but unsent");
  });

  it("debounce-saves the typed text for its channel", () => {
    vi.useFakeTimers();
    renderHook(() => useChannelDraftPersistence(3, "hello", vi.fn()));
    act(() => {
      vi.advanceTimersByTime(400);
    });
    expect(getChannelDraft(3)).toBe("hello");
  });

  // Regression: on a fast switch, the debounce paired the wrong channel with the
  // wrong text — one channel's draft bled into another. The outgoing channel's
  // draft must be saved and the incoming channel's restored, never crossed.
  it("on channel switch saves the outgoing draft and restores the incoming (no bleed)", () => {
    setChannelDraft(20, "channel-20 draft");
    const setContent = vi.fn();
    const { rerender } = renderHook(
      ({ id, c }) => useChannelDraftPersistence(id, c, setContent),
      { initialProps: { id: 10, c: "channel-10 draft" } },
    );
    rerender({ id: 20, c: "channel-10 draft" }); // switch; outgoing content still present
    expect(getChannelDraft(10)).toBe("channel-10 draft"); // outgoing saved
    expect(setContent).toHaveBeenLastCalledWith("channel-20 draft"); // incoming restored, not bled
  });
});
