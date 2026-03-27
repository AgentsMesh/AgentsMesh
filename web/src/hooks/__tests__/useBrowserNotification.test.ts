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
});
