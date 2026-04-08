import { describe, it, expect, vi, beforeEach } from "vitest";
import { handleNotificationEvent } from "../notificationHandler";
import type { RealtimeEvent } from "@/lib/realtime";

vi.mock("sonner", () => ({
  toast: {
    info: vi.fn(),
    warning: vi.fn(),
  },
}));

import { toast } from "sonner";

function makeEvent(channels: { toast?: boolean; browser?: boolean }, priority = "normal", link?: string): RealtimeEvent {
  return {
    type: "notification",
    category: "notification",
    organization_id: 1,
    data: { source: "test", title: "Test", body: "Body", priority, channels, link },
    timestamp: Date.now(),
  } as RealtimeEvent;
}

function makeOpts() {
  return {
    router: { push: vi.fn() },
    showBrowserNotification: vi.fn(),
  };
}

describe("handleNotificationEvent", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("ignores non-notification events", () => {
    const opts = makeOpts();
    handleNotificationEvent({ type: "pod:created", data: {}, timestamp: 0, category: "entity", organization_id: 1 } as RealtimeEvent, opts);
    expect(toast.info).not.toHaveBeenCalled();
    expect(opts.showBrowserNotification).not.toHaveBeenCalled();
  });

  it("shows toast when tab visible and both channels enabled", () => {
    Object.defineProperty(document, "visibilityState", { value: "visible", configurable: true });
    const opts = makeOpts();
    handleNotificationEvent(makeEvent({ toast: true, browser: true }), opts);
    expect(toast.info).toHaveBeenCalledTimes(1);
    expect(opts.showBrowserNotification).not.toHaveBeenCalled();
  });

  it("shows browser notification when tab hidden and both channels enabled", () => {
    Object.defineProperty(document, "visibilityState", { value: "hidden", configurable: true });
    const opts = makeOpts();
    handleNotificationEvent(makeEvent({ toast: true, browser: true }), opts);
    expect(toast.info).not.toHaveBeenCalled();
    expect(opts.showBrowserNotification).toHaveBeenCalledTimes(1);
  });

  it("shows toast only when browser channel disabled", () => {
    Object.defineProperty(document, "visibilityState", { value: "hidden", configurable: true });
    const opts = makeOpts();
    handleNotificationEvent(makeEvent({ toast: true, browser: false }), opts);
    expect(toast.info).toHaveBeenCalledTimes(1);
    expect(opts.showBrowserNotification).not.toHaveBeenCalled();
  });

  it("shows browser only when toast channel disabled", () => {
    Object.defineProperty(document, "visibilityState", { value: "visible", configurable: true });
    const opts = makeOpts();
    handleNotificationEvent(makeEvent({ toast: false, browser: true }), opts);
    expect(toast.info).not.toHaveBeenCalled();
    expect(opts.showBrowserNotification).toHaveBeenCalledTimes(1);
  });

  it("uses toast.warning for high priority", () => {
    Object.defineProperty(document, "visibilityState", { value: "visible", configurable: true });
    const opts = makeOpts();
    handleNotificationEvent(makeEvent({ toast: true, browser: false }, "high"), opts);
    expect(toast.warning).toHaveBeenCalledTimes(1);
    expect(toast.info).not.toHaveBeenCalled();
  });

  it("shows nothing when both channels disabled", () => {
    const opts = makeOpts();
    handleNotificationEvent(makeEvent({ toast: false, browser: false }), opts);
    expect(toast.info).not.toHaveBeenCalled();
    expect(toast.warning).not.toHaveBeenCalled();
    expect(opts.showBrowserNotification).not.toHaveBeenCalled();
  });
});
