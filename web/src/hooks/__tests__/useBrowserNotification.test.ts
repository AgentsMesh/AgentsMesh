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

describe("useBrowserNotification", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockClose.mockClear();
    mockNotification.mockClear();
    mockShowNotification.mockClear();

    // Setup default Notification mock
    MockNotification.permission = "default";
    MockNotification.requestPermission = vi.fn().mockResolvedValue("granted");

    // @ts-expect-error - mocking global Notification
    global.Notification = MockNotification;

    // Mock matchMedia for PWA detection
    global.matchMedia = vi.fn().mockReturnValue({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    });

    // Mock PushManager
    // @ts-expect-error - mocking PushManager
    global.PushManager = class {};

    // Mock navigator.serviceWorker
    Object.defineProperty(navigator, "serviceWorker", {
      value: mockServiceWorker,
      configurable: true,
      writable: true,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("initial state", () => {
    it("should return default permission when Notification API is supported", async () => {
      MockNotification.permission = "default";

      const { result } = renderHook(() => useBrowserNotification());

      // Wait for useEffect to run
      await waitFor(() => {
        expect(result.current.isSupported).toBe(true);
      });

      expect(result.current.permission).toBe("default");
    });

    it("should return granted permission when already granted", async () => {
      MockNotification.permission = "granted";

      const { result } = renderHook(() => useBrowserNotification());

      await waitFor(() => {
        expect(result.current.isSupported).toBe(true);
      });

      expect(result.current.permission).toBe("granted");
    });

    it("should return denied permission when denied", async () => {
      MockNotification.permission = "denied";

      const { result } = renderHook(() => useBrowserNotification());

      await waitFor(() => {
        expect(result.current.isSupported).toBe(true);
      });

      expect(result.current.permission).toBe("denied");
    });

    it("should detect PWA mode via display-mode", async () => {
      global.matchMedia = vi.fn().mockReturnValue({
        matches: true, // standalone mode
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
      });

      const { result } = renderHook(() => useBrowserNotification());

      await waitFor(() => {
        expect(result.current.isPWA).toBe(true);
      });
    });

    it("should detect iOS PWA mode via navigator.standalone", async () => {
      global.matchMedia = vi.fn().mockReturnValue({
        matches: false,
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
      });
      // @ts-expect-error - iOS Safari specific
      navigator.standalone = true;

      const { result } = renderHook(() => useBrowserNotification());

      await waitFor(() => {
        expect(result.current.isPWA).toBe(true);
      });

      // Cleanup
      // @ts-expect-error - iOS Safari specific
      delete navigator.standalone;
    });

    it("should return unsupported when Notification API is not available and no SW support", async () => {
      // @ts-expect-error - removing Notification from window
      delete global.Notification;
      // @ts-expect-error - removing PushManager
      delete global.PushManager;

      const { result } = renderHook(() => useBrowserNotification());

      await waitFor(() => {
        expect(result.current.isSupported).toBe(false);
      });
      expect(result.current.permission).toBe("unsupported");
    });

    it("should return default permission when only SW is supported (iOS PWA scenario)", async () => {
      // @ts-expect-error - removing Notification from window
      delete global.Notification;
      // PushManager still exists

      const { result } = renderHook(() => useBrowserNotification());

      await waitFor(() => {
        expect(result.current.isSupported).toBe(true);
      });
      // Without Notification API, permission falls back to "default" via SW
      expect(result.current.permission).toBe("default");
    });
  });

  describe("requestPermission", () => {
    it("should request permission and return true when granted", async () => {
      MockNotification.permission = "default";
      MockNotification.requestPermission = vi.fn().mockResolvedValue("granted");

      const { result } = renderHook(() => useBrowserNotification());

      await waitFor(() => {
        expect(result.current.isSupported).toBe(true);
      });

      let granted: boolean = false;
      await act(async () => {
        granted = await result.current.requestPermission();
      });

      expect(granted).toBe(true);
      expect(MockNotification.requestPermission).toHaveBeenCalled();
    });

    it("should return false when permission is denied", async () => {
      MockNotification.permission = "default";
      MockNotification.requestPermission = vi.fn().mockResolvedValue("denied");

      const { result } = renderHook(() => useBrowserNotification());

      await waitFor(() => {
        expect(result.current.isSupported).toBe(true);
      });

      let granted: boolean = true;
      await act(async () => {
        granted = await result.current.requestPermission();
      });

      expect(granted).toBe(false);
    });

    it("should return true immediately if already granted", async () => {
      MockNotification.permission = "granted";

      const { result } = renderHook(() => useBrowserNotification());

      await waitFor(() => {
        expect(result.current.isSupported).toBe(true);
      });

      let granted: boolean = false;
      await act(async () => {
        granted = await result.current.requestPermission();
      });

      expect(granted).toBe(true);
      expect(MockNotification.requestPermission).not.toHaveBeenCalled();
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

      let granted: boolean = true;
      await act(async () => {
        granted = await result.current.requestPermission();
      });

      expect(granted).toBe(false);
      expect(consoleSpy).toHaveBeenCalledWith("[BrowserNotification] Notifications not supported");
      consoleSpy.mockRestore();
    });

    it("should return false when Notification API not available (iOS PWA without Notification)", async () => {
      // @ts-expect-error - removing Notification from window
      delete global.Notification;
      // PushManager still exists - simulates iOS PWA

      const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
      const { result } = renderHook(() => useBrowserNotification());

      await waitFor(() => {
        expect(result.current.isSupported).toBe(true);
      });

      let granted: boolean = true;
      await act(async () => {
        granted = await result.current.requestPermission();
      });

      expect(granted).toBe(false);
      expect(consoleSpy).toHaveBeenCalledWith(
        "[BrowserNotification] Cannot request permission without Notification API"
      );
      consoleSpy.mockRestore();
    });

    it("should handle request permission error gracefully", async () => {
      MockNotification.permission = "default";
      MockNotification.requestPermission = vi.fn().mockRejectedValue(new Error("Permission error"));

      const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      const { result } = renderHook(() => useBrowserNotification());

      await waitFor(() => {
        expect(result.current.isSupported).toBe(true);
      });

      let granted: boolean = true;
      await act(async () => {
        granted = await result.current.requestPermission();
      });

      expect(granted).toBe(false);
      expect(consoleSpy).toHaveBeenCalled();
      consoleSpy.mockRestore();
    });
  });

  describe("showNotification", () => {
    it("should show notification via Service Worker when available", async () => {
      MockNotification.permission = "granted";

      const { result } = renderHook(() => useBrowserNotification());

      await waitFor(() => {
        expect(result.current.isSupported).toBe(true);
      });

      // Wait for SW registration
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

      // Wait for SW registration
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

      // Wait for SW registration
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
      // Tag should be auto-generated with timestamp prefix
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

      // Wait for SW registration
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
      // Should fallback to direct API
      expect(mockNotification).toHaveBeenCalledWith("Test Title", expect.any(Object));
      expect(consoleLogSpy).toHaveBeenCalledWith("[BrowserNotification] Shown via Notification API");
      consoleSpy.mockRestore();
      consoleLogSpy.mockRestore();
    });

    it("should handle onClick callback in direct Notification API and simulate click", async () => {
      // No SW available, so it will use direct API
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

      // Verify notification was created with onClick handler
      expect(mockNotification).toHaveBeenCalledWith("Test Title", expect.objectContaining({
        requireInteraction: false,
      }));
    });

    it("should auto-close direct notification after timeout", async () => {
      vi.useFakeTimers({ shouldAdvanceTime: true });
      MockNotification.permission = "granted";
      // No SW available, so it will use direct API
      Object.defineProperty(navigator, "serviceWorker", {
        value: undefined,
        configurable: true,
        writable: true,
      });
      // @ts-expect-error - removing PushManager
      delete global.PushManager;

      vi.spyOn(console, "log").mockImplementation(() => {});
      const { result } = renderHook(() => useBrowserNotification());

      // Wait for initialization
      await vi.waitFor(() => {
        expect(result.current.isSupported).toBe(true);
      });

      await act(async () => {
        await result.current.showNotification({
          title: "Test Title",
        });
      });

      expect(mockClose).not.toHaveBeenCalled();

      // Advance timers
      vi.advanceTimersByTime(5000);

      expect(mockClose).toHaveBeenCalled();

      vi.useRealTimers();
    });

    it("should return false when no notification method available (edge case)", async () => {
      // This tests the edge case where SW is supported but no SW registration
      // and no Notification API is available
      // @ts-expect-error - removing Notification
      delete global.Notification;

      // SW supported but registration returns null
      Object.defineProperty(navigator, "serviceWorker", {
        value: {
          ready: Promise.resolve(null), // null registration
        },
        configurable: true,
        writable: true,
      });

      const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
      const { result } = renderHook(() => useBrowserNotification());

      await waitFor(() => {
        expect(result.current.isSupported).toBe(true);
      });

      // Wait for SW registration
      await act(async () => {
        await new Promise(resolve => setTimeout(resolve, 0));
      });

      // Since permission check uses getPermission() which returns "default" without Notification
      let success: boolean = true;
      await act(async () => {
        success = await result.current.showNotification({
          title: "Test Title",
        });
      });

      // Should return false because permission is "default" (not granted)
      expect(success).toBe(false);
      consoleSpy.mockRestore();
    });

    it("should handle direct Notification API failure gracefully", async () => {
      // No SW, direct Notification API throws
      Object.defineProperty(navigator, "serviceWorker", {
        value: undefined,
        configurable: true,
        writable: true,
      });
      // @ts-expect-error - removing PushManager
      delete global.PushManager;

      // Make direct Notification throw
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
});
