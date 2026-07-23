import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { MessageList } from "../MessageList";
import type { TransformedMessage } from "../types";

vi.mock("next-intl", () => ({ useTranslations: () => (key: string) => key }));
vi.mock("@/stores/pod", () => ({ usePods: () => [] }));

let resizeCallback: ResizeObserverCallback | null = null;

class TestResizeObserver {
  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn();
  constructor(callback: ResizeObserverCallback) {
    resizeCallback = callback;
  }
}

function msg(id: number): TransformedMessage {
  return {
    id,
    body: `message ${id}`,
    messageType: "text",
    createdAt: "2026-01-01T12:00:00Z",
    user: { id: 7, username: "dev" },
  };
}

function renderList(props: Partial<React.ComponentProps<typeof MessageList>> = {}) {
  return render(
    <MessageList
      messages={[msg(10), msg(11), msg(12)]}
      loading={false}
      channelId={1}
      {...props}
    />,
  );
}

let rafCallbacks: FrameRequestCallback[] = [];
function flushRaf() {
  const pending = rafCallbacks;
  rafCallbacks = [];
  pending.forEach((cb) => cb(performance.now()));
}

describe("MessageList entry scroll", () => {
  const originalRAF = global.requestAnimationFrame;
  const originalCAF = global.cancelAnimationFrame;
  const originalRO = global.ResizeObserver;
  const originalScrollIntoView = Element.prototype.scrollIntoView;

  beforeEach(() => {
    vi.useFakeTimers();
    resizeCallback = null;
    rafCallbacks = [];
    global.ResizeObserver = TestResizeObserver as unknown as typeof ResizeObserver;
    global.requestAnimationFrame = ((cb: FrameRequestCallback) => {
      rafCallbacks.push(cb);
      return rafCallbacks.length;
    }) as typeof requestAnimationFrame;
    global.cancelAnimationFrame = (() => {}) as typeof cancelAnimationFrame;
    Element.prototype.scrollIntoView = vi.fn();
  });

  afterEach(() => {
    vi.useRealTimers();
    // Restore every global clobbered above. The fake requestAnimationFrame never
    // invokes its callback, so leaking it into later test files in the same
    // worker thread hangs their animations/userEvent flows (5s timeouts).
    global.requestAnimationFrame = originalRAF;
    global.cancelAnimationFrame = originalCAF;
    global.ResizeObserver = originalRO;
    Element.prototype.scrollIntoView = originalScrollIntoView;
  });

  it("scrolls to bottom on read-channel entry", () => {
    renderList();

    expect(Element.prototype.scrollIntoView).toHaveBeenCalledWith({ behavior: "instant" });
  });

  it("scrolls the unread divider on unread entry", () => {
    renderList({ firstUnreadId: 11 });

    // getByRole("separator") is the UnreadDivider (the [data-unread-anchor]
    // element); asserting on its own scrollIntoView proves the divider — not
    // some other element — is what entry positioning scrolled to.
    const divider = screen.getByRole("separator");
    expect(divider.scrollIntoView).toHaveBeenCalledWith({ block: "start", behavior: "instant" });
  });

  it("waits for the anchor to resolve before the first scroll (no bottom→divider jump)", () => {
    // Unknown cursor: entryAnchorResolved false, firstUnreadId not yet known.
    const { rerender } = renderList({ firstUnreadId: null, entryAnchorResolved: false });
    expect(Element.prototype.scrollIntoView).not.toHaveBeenCalled();

    // Cursor resolves to an unread divider — now (and only now) it anchors,
    // straight to the divider rather than bottom-then-jump.
    rerender(
      <MessageList messages={[msg(10), msg(11), msg(12)]} loading={false} channelId={1} firstUnreadId={11} entryAnchorResolved />,
    );
    const divider = screen.getByRole("separator");
    expect(divider.scrollIntoView).toHaveBeenCalledWith({ block: "start", behavior: "instant" });
  });

  it("re-anchors when a fresh same-length message window arrives", () => {
    const { rerender } = renderList({ messages: [msg(1), msg(2), msg(3)] });
    vi.mocked(Element.prototype.scrollIntoView).mockClear();

    rerender(<MessageList messages={[msg(4), msg(5), msg(6)]} loading={false} channelId={1} />);

    expect(Element.prototype.scrollIntoView).toHaveBeenCalledWith({ behavior: "instant" });
  });

  it("keeps bottom alignment when content resizes before user intent", () => {
    renderList();
    vi.mocked(Element.prototype.scrollIntoView).mockClear();

    act(() => {
      resizeCallback?.([], {} as ResizeObserver);
      flushRaf();
    });

    expect(Element.prototype.scrollIntoView).toHaveBeenCalledWith({ behavior: "instant" });
  });

  it("does not re-anchor after the user scrolls (native scrollbar / any source)", () => {
    renderList();
    vi.mocked(Element.prototype.scrollIntoView).mockClear();

    // A scroll to a position we did not programmatically request = user intent,
    // regardless of gesture (native scrollbar, find-in-page, keyboard).
    const list = screen.getByTestId("channel-message-list");
    Object.defineProperty(list, "scrollTop", { value: 999, configurable: true });
    fireEvent.scroll(list);
    act(() => {
      resizeCallback?.([], {} as ResizeObserver);
      flushRaf();
    });

    expect(Element.prototype.scrollIntoView).not.toHaveBeenCalled();
  });

  it("preserves the previous first message after loading older messages", () => {
    const { rerender } = renderList({ messages: [msg(10), msg(11), msg(12)], loadingMore: false });

    rerender(<MessageList messages={[msg(10), msg(11), msg(12)]} loading={false} loadingMore channelId={1} />);
    vi.mocked(Element.prototype.scrollIntoView).mockClear();
    rerender(<MessageList messages={[msg(7), msg(8), msg(9), msg(10), msg(11), msg(12)]} loading={false} loadingMore={false} channelId={1} />);

    // Load-more restores the viewport to the previously-first message (top),
    // not the bottom.
    const previousFirst = document.querySelector('[data-message-id="10"]');
    expect(previousFirst?.scrollIntoView).toHaveBeenCalledWith({ block: "start" });
  });

  it("follows a tall new message to the bottom using the pre-append position", () => {
    const { rerender } = renderList({ messages: [msg(10), msg(11), msg(12)] });
    const list = screen.getByTestId("channel-message-list");
    // User sits exactly at the bottom before the new message arrives.
    Object.defineProperty(list, "scrollHeight", { value: 300, configurable: true });
    Object.defineProperty(list, "clientHeight", { value: 300, configurable: true });
    Object.defineProperty(list, "scrollTop", { value: 0, configurable: true });
    fireEvent.scroll(list);
    vi.mocked(Element.prototype.scrollIntoView).mockClear();

    // A tall message appends: post-append the raw gap (60px) exceeds the 50px
    // slack, so a live DOM read would conclude "not at bottom" and strand it —
    // the follow decision must instead use the position held before the append.
    Object.defineProperty(list, "scrollHeight", { value: 360, configurable: true });
    rerender(<MessageList messages={[msg(10), msg(11), msg(12), msg(13)]} loading={false} channelId={1} />);

    expect(Element.prototype.scrollIntoView).toHaveBeenCalled();
  });

  it("shows the new-message pill instead of following when the user has scrolled up", () => {
    const { rerender } = renderList({ messages: [msg(10), msg(11), msg(12)] });
    const list = screen.getByTestId("channel-message-list");
    Object.defineProperty(list, "scrollHeight", { value: 1000, configurable: true });
    Object.defineProperty(list, "clientHeight", { value: 300, configurable: true });
    Object.defineProperty(list, "scrollTop", { value: 100, configurable: true });
    fireEvent.scroll(list);
    vi.mocked(Element.prototype.scrollIntoView).mockClear();

    rerender(<MessageList messages={[msg(10), msg(11), msg(12), msg(13)]} loading={false} channelId={1} />);

    expect(Element.prototype.scrollIntoView).not.toHaveBeenCalled();
    expect(screen.getByText("1")).toBeInTheDocument();
  });

  it("does not re-anchor the unread divider when a message streams in after entry", () => {
    const { rerender } = renderList({ firstUnreadId: 11, messages: [msg(10), msg(11), msg(12)] });
    expect(Element.prototype.scrollIntoView).toHaveBeenCalledWith({ block: "start", behavior: "instant" });
    vi.mocked(Element.prototype.scrollIntoView).mockClear();

    rerender(
      <MessageList messages={[msg(10), msg(11), msg(12), msg(13)]} loading={false} channelId={1} firstUnreadId={11} entryAnchorResolved />,
    );

    // Entry positioning settled on the append; the streamed message must not drag
    // the viewport back to the divider (block:start is the divider-anchor signature).
    expect(Element.prototype.scrollIntoView).not.toHaveBeenCalledWith({ block: "start", behavior: "instant" });
  });

  it("does not follow a message that streams in before the unread cursor resolves", async () => {
    // Unknown cursor: entryAnchorResolved false, firstUnreadId not yet known.
    const { rerender } = renderList({ messages: [msg(10), msg(11), msg(12)], firstUnreadId: null, entryAnchorResolved: false });
    // Flush the channel-switch seed microtask (setIsAtBottom(false) for an
    // unknown cursor). In the real app this microtask always runs before the
    // next macrotask — i.e. before any streamed message can arrive.
    await act(async () => { await Promise.resolve(); });
    vi.mocked(Element.prototype.scrollIntoView).mockClear();

    // A message arrives during the fetchUnreadCounts window. It must NOT auto-
    // follow to the bottom — that would trip userInterrupted and lose the divider.
    rerender(<MessageList messages={[msg(10), msg(11), msg(12), msg(13)]} loading={false} channelId={1} firstUnreadId={null} entryAnchorResolved={false} />);
    expect(Element.prototype.scrollIntoView).not.toHaveBeenCalled();

    // Cursor resolves to a divider — entry now anchors straight to it.
    rerender(<MessageList messages={[msg(10), msg(11), msg(12), msg(13)]} loading={false} channelId={1} firstUnreadId={11} entryAnchorResolved />);
    const divider = screen.getByRole("separator");
    expect(divider.scrollIntoView).toHaveBeenCalledWith({ block: "start", behavior: "instant" });
  });

  it("does not re-anchor entry positioning after a load-more prepend", () => {
    const { rerender } = renderList({ messages: [msg(10), msg(11), msg(12)], loadingMore: false });
    rerender(<MessageList messages={[msg(10), msg(11), msg(12)]} loading={false} loadingMore channelId={1} />);
    vi.mocked(Element.prototype.scrollIntoView).mockClear();

    // Older page prepends: firstId changes (→ anchorKey) but the tail id is
    // unchanged. The load-more restore scrolls the previous first message to the
    // top; entry positioning must NOT also re-anchor to the bottom and fight it.
    rerender(<MessageList messages={[msg(7), msg(8), msg(9), msg(10), msg(11), msg(12)]} loading={false} loadingMore={false} channelId={1} />);

    expect(Element.prototype.scrollIntoView).not.toHaveBeenCalledWith({ behavior: "instant" });
    expect(document.querySelector('[data-message-id="10"]')?.scrollIntoView).toHaveBeenCalledWith({ block: "start" });
  });
});
