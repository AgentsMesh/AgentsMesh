export type EntryTarget = "bottom" | "unread";

export interface EntryState {
  userInterrupted: boolean;
  settled: boolean;
  lastAppliedKey: string | null;
  anchoredLastId: number | null;
  target: EntryTarget | null;
  quiesceTimer: ReturnType<typeof setTimeout> | null;
  resizeFrame: number | null;
  // scrollTop we expect after our own programmatic scrollIntoView; the next
  // scroll event matching it is ours, anything else is the user taking over.
  expectedScrollTop: number | null;
}

export function createEntryState(): EntryState {
  return {
    userInterrupted: false,
    settled: false,
    lastAppliedKey: null,
    anchoredLastId: null,
    target: null,
    quiesceTimer: null,
    resizeFrame: null,
    expectedScrollTop: null,
  };
}

export function clearEntryTimers(state: EntryState) {
  if (state.quiesceTimer) clearTimeout(state.quiesceTimer);
  if (state.resizeFrame != null) cancelAnimationFrame(state.resizeFrame);
  state.quiesceTimer = null;
  state.resizeFrame = null;
}

export function scrollToEntryAnchor(
  container: HTMLDivElement | null,
  bottom: HTMLDivElement | null,
  firstUnreadId: number | null | undefined,
  state: EntryState,
): EntryTarget | null {
  const before = container?.scrollTop ?? null;
  const anchor = firstUnreadId != null ? container?.querySelector("[data-unread-anchor]") : null;
  if (anchor) {
    anchor.scrollIntoView({ block: "start", behavior: "instant" as ScrollBehavior });
  } else if (bottom) {
    bottom.scrollIntoView({ behavior: "instant" as ScrollBehavior });
  } else {
    return null;
  }
  // Arm the "this scroll is ours" guard ONLY if the position actually moved: a
  // no-op scroll fires no scroll event, so a lingering expectedScrollTop would
  // otherwise swallow a later real user scroll that lands on that same offset.
  if (container) state.expectedScrollTop = container.scrollTop === before ? null : container.scrollTop;
  return anchor ? "unread" : "bottom";
}
