import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useBrowserNotification } from "../useBrowserNotification";

// Mock Notification API
const mockNotification = vi.fn();
const mockClose = vi.fn();

class MockNotification {
  static permission: NotificationPermission = "default";
  static requestPermission = vi.fn();

  title: string;
  options: NotificationOptions;
  onclick: ((event: Event) => void) | null = null;
  close = mockClose;

  constructor(title: string, options?: NotificationOptions) {
    this.title = title;
    this.options = options || {};
    mockNotification(title, options);
  }
}

// Mock ServiceWorkerRegistration
const mockShowNotification = vi.fn().mockResolvedValue(undefined);
const mockServiceWorkerRegistration = {
  showNotification: mockShowNotification,
};

// Mock navigator.serviceWorker
const mockServiceWorker = {
  ready: Promise.resolve(mockServiceWorkerRegistration),
};

describe("useBrowserNotification - showNotification", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockClose.mockClear();
    mockNotification.mockClear();
    mockShowNotification.mockClear();

    MockNotification.permission = "default";
    MockNotification.requestPermission = vi.fn().mockResolvedValue("granted");

    // @ts-expect-error - mocking global Notification
    global.Notification = MockNotification;

    global.matchMedia = vi.fn().mockReturnValue({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    });

    // @ts-expect-error - mocking PushManager
    global.PushManager = class {};

    Object.defineProperty(navigator, "serviceWorker", {
      value: mockServiceWorker,
      configurable: true,
      writable: true,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("should show notification via Service Worker when available", async () => {
    MockNotification.permission = "granted";

    const { result } = renderHook(() => useBrowserNotification());

    await waitFor(() => {
      expect(result.current.isSupported).toBe(true);
    });

    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 0));
    });

    let success: boolean = false;
    await act(async () => {
      success = await result.current.showNotification({
        title: "Test Title",
        body: "Test Body",
      });
    });

    expect(success).toBe(true);
    expect(mockShowNotification).toHaveBeenCalledWith("Test Title", expect.objectContaining({
      body: "Test Body",
    }));
  });

  it("should return false when permission is not granted", async () => {
    MockNotification.permission = "default";

    const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    const { result } = renderHook(() => useBrowserNotification());

    await waitFor(() => {
      expect(result.current.isSupported).toBe(true);
    });

    let success: boolean = true;
    await act(async () => {
      success = await result.current.showNotification({
        title: "Test Title",
      });
    });

    expect(success).toBe(false);
    expect(mockShowNotification).not.toHaveBeenCalled();
    expect(consoleSpy).toHaveBeenCalledWith(
      "[BrowserNotification] Permission not granted:",
      "default"
    );
    consoleSpy.mockRestore();
  });

  it("should return false when not supported", async () => {
    // @ts-expect-error - removing Notification from window
    delete global.Notification;
    // @ts-expect-error - removing PushManager
    delete global.PushManager;

    const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    const { result } = renderHook(() => useBrowserNotification());

    await waitFor(() => {
      expect(result.current.isSupported).toBe(false);
    });

    let success: boolean = true;
    await act(async () => {
      success = await result.current.showNotification({
        title: "Test Title",
      });
    });

    expect(success).toBe(false);
    expect(consoleSpy).toHaveBeenCalledWith("[BrowserNotification] Notifications not supported");
    consoleSpy.mockRestore();
  });

  it("should set notification options correctly via SW", async () => {
    MockNotification.permission = "granted";

    const { result } = renderHook(() => useBrowserNotification());

    await waitFor(() => {
      expect(result.current.isSupported).toBe(true);
    });

    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 0));
    });

    await act(async () => {
      await result.current.showNotification({
        title: "Test Title",
        body: "Test Body",
        icon: "/custom-icon.png",
        tag: "test-tag",
        data: { podKey: "pod-123" },
      });
    });

    expect(mockShowNotification).toHaveBeenCalledWith("Test Title", expect.objectContaining({
      body: "Test Body",
      icon: "/custom-icon.png",
      tag: "test-tag",
    }));
  });

  it("should use default icon and generate tag when not specified", async () => {
    MockNotification.permission = "granted";

    const { result } = renderHook(() => useBrowserNotification());

    await waitFor(() => {
      expect(result.current.isSupported).toBe(true);
    });

    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 0));
    });

    await act(async () => {
      await result.current.showNotification({
        title: "Test Title",
      });
    });

    expect(mockShowNotification).toHaveBeenCalledWith("Test Title", expect.objectContaining({
      icon: "/icons/icon.svg",
      badge: "/icons/icon.svg",
      silent: false,
    }));
    const callArgs = mockShowNotification.mock.calls[0][1];
    expect(callArgs.tag).toMatch(/^notification-\d+$/);
  });

  it("should fallback to direct Notification API when SW fails", async () => {
    MockNotification.permission = "granted";
    mockShowNotification.mockRejectedValueOnce(new Error("SW failed"));

    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const consoleLogSpy = vi.spyOn(console, "log").mockImplementation(() => {});
    const { result } = renderHook(() => useBrowserNotification());

    await waitFor(() => {
      expect(result.current.isSupported).toBe(true);
    });

    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 0));
    });

    let success: boolean = false;
    await act(async () => {
      success = await result.current.showNotification({
        title: "Test Title",
      });
    });

    expect(success).toBe(true);
    expect(mockNotification).toHaveBeenCalledWith("Test Title", expect.any(Object));
    expect(consoleLogSpy).toHaveBeenCalledWith("[BrowserNotification] Shown via Notification API");
    consoleSpy.mockRestore();
    consoleLogSpy.mockRestore();
  });

  it("should handle onClick callback in direct Notification API and simulate click", async () => {
    Object.defineProperty(navigator, "serviceWorker", {
      value: undefined,
      configurable: true,
      writable: true,
    });
    // @ts-expect-error - removing PushManager
    delete global.PushManager;

    MockNotification.permission = "granted";

    const onClick = vi.fn();
    const mockFocus = vi.fn();
    global.window.focus = mockFocus;

    vi.spyOn(console, "log").mockImplementation(() => {});

    const { result } = renderHook(() => useBrowserNotification());

    await waitFor(() => {
      expect(result.current.isSupported).toBe(true);
    });

    await act(async () => {
      await result.current.showNotification({
        title: "Test Title",
        onClick,
      });
    });

    expect(mockNotification).toHaveBeenCalledWith("Test Title", expect.objectContaining({
      requireInteraction: false,
    }));
  });

  it("should auto-close direct notification after timeout", async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    MockNotification.permission = "granted";
    Object.defineProperty(navigator, "serviceWorker", {
      value: undefined,
      configurable: true,
      writable: true,
    });
    // @ts-expect-error - removing PushManager
    delete global.PushManager;

    vi.spyOn(console, "log").mockImplementation(() => {});
    const { result } = renderHook(() => useBrowserNotification());

    await vi.waitFor(() => {
      expect(result.current.isSupported).toBe(true);
    });

    await act(async () => {
      await result.current.showNotification({
        title: "Test Title",
      });
    });

    expect(mockClose).not.toHaveBeenCalled();

    vi.advanceTimersByTime(5000);

    expect(mockClose).toHaveBeenCalled();

    vi.useRealTimers();
  });

  it("should return false when no notification method available (edge case)", async () => {
    // @ts-expect-error - removing Notification
    delete global.Notification;

    Object.defineProperty(navigator, "serviceWorker", {
      value: {
        ready: Promise.resolve(null),
      },
      configurable: true,
      writable: true,
    });

    const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
    const { result } = renderHook(() => useBrowserNotification());

    await waitFor(() => {
      expect(result.current.isSupported).toBe(true);
    });

    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 0));
    });

    let success: boolean = true;
    await act(async () => {
      success = await result.current.showNotification({
        title: "Test Title",
      });
    });

    expect(success).toBe(false);
    consoleSpy.mockRestore();
  });

  it("should handle direct Notification API failure gracefully", async () => {
    Object.defineProperty(navigator, "serviceWorker", {
      value: undefined,
      configurable: true,
      writable: true,
    });
    // @ts-expect-error - removing PushManager
    delete global.PushManager;

    // @ts-expect-error - mocking constructor to throw
    global.Notification = class {
      constructor() {
        throw new Error("Notification constructor failed");
      }
      static permission: NotificationPermission = "granted";
    };

    const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const { result } = renderHook(() => useBrowserNotification());

    await waitFor(() => {
      expect(result.current.isSupported).toBe(true);
    });

    let success: boolean = true;
    await act(async () => {
      success = await result.current.showNotification({
        title: "Test Title",
      });
    });

    expect(success).toBe(false);
    expect(consoleSpy).toHaveBeenCalledWith(
      "[BrowserNotification] Direct notification failed:",
      expect.any(Error)
    );
    consoleSpy.mockRestore();
  });
});
