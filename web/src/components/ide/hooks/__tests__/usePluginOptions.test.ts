import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { usePluginOptions } from "../usePluginOptions";
import { runnerApi } from "@/lib/api/client";
import type { PluginCapability } from "@/lib/api/runner";

// Mock the client API
vi.mock("@/lib/api/client", () => ({
  runnerApi: {
    getPluginOptions: vi.fn(),
  },
}));

describe("usePluginOptions", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("initial state", () => {
    it("should return empty state when runner is null", () => {
      const { result } = renderHook(() => usePluginOptions(null, ""));

      expect(result.current.plugins).toEqual([]);
      expect(result.current.loading).toBe(false);
      expect(result.current.config).toEqual({});
    });

    it("should return empty state when agent is empty", () => {
      const { result } = renderHook(() => usePluginOptions(1, ""));

      expect(result.current.plugins).toEqual([]);
      expect(result.current.loading).toBe(false);
      expect(result.current.config).toEqual({});
    });
  });

  describe("loading plugins", () => {
    it("should fetch plugins when runner and agent are provided", async () => {
      const mockPlugins: PluginCapability[] = [
        {
          name: "claude-code",
          version: "1.0.0",
          description: "Claude Code Plugin",
          supported_agents: ["claude-code"],
          ui: {
            configurable: true,
            fields: [
              { name: "mcp_enabled", type: "boolean", label: "Enable MCP", default: true },
            ],
          },
        },
      ];

      vi.mocked(runnerApi.getPluginOptions).mockResolvedValue({
        plugins: mockPlugins,
      });

      const { result } = renderHook(() => usePluginOptions(1, "claude-code"));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
        expect(result.current.plugins).toEqual(mockPlugins);
      });

      expect(runnerApi.getPluginOptions).toHaveBeenCalledWith(1, "claude-code");
    });

    it("should initialize config with default values", async () => {
      const mockPlugins: PluginCapability[] = [
        {
          name: "test-plugin",
          version: "1.0.0",
          description: "Test Plugin",
          supported_agents: ["test-agent"],
          ui: {
            configurable: true,
            fields: [
              { name: "enabled", type: "boolean", label: "Enable", default: true },
              { name: "count", type: "number", label: "Count", default: 10 },
            ],
          },
        },
      ];

      vi.mocked(runnerApi.getPluginOptions).mockResolvedValue({
        plugins: mockPlugins,
      });

      const { result } = renderHook(() => usePluginOptions(1, "test-agent"));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
        expect(result.current.config).toEqual({
          "test-plugin.enabled": true,
          "test-plugin.count": 10,
        });
      });
    });

    it("should handle API error gracefully", async () => {
      const consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      vi.mocked(runnerApi.getPluginOptions).mockRejectedValue(
        new Error("Network error")
      );

      const { result } = renderHook(() => usePluginOptions(1, "test-agent"));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      expect(result.current.plugins).toEqual([]);
      consoleSpy.mockRestore();
    });
  });

  describe("updateConfig", () => {
    it("should update config value", async () => {
      const mockPlugins: PluginCapability[] = [
        {
          name: "test-plugin",
          version: "1.0.0",
          description: "Test Plugin",
          supported_agents: ["test-agent"],
          ui: {
            configurable: true,
            fields: [
              { name: "enabled", type: "boolean", label: "Enable", default: true },
            ],
          },
        },
      ];

      vi.mocked(runnerApi.getPluginOptions).mockResolvedValue({
        plugins: mockPlugins,
      });

      const { result } = renderHook(() => usePluginOptions(1, "test-agent"));

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      // Update the config
      act(() => {
        result.current.updateConfig("test-plugin", "enabled", false);
      });

      expect(result.current.config["test-plugin.enabled"]).toBe(false);
    });
  });

  describe("resetConfig", () => {
    it("should reset config and plugins to empty", async () => {
      const mockPlugins: PluginCapability[] = [
        {
          name: "test-plugin",
          version: "1.0.0",
          description: "Test Plugin",
          supported_agents: ["test-agent"],
          ui: {
            configurable: true,
            fields: [
              { name: "enabled", type: "boolean", label: "Enable", default: true },
            ],
          },
        },
      ];

      vi.mocked(runnerApi.getPluginOptions).mockResolvedValue({
        plugins: mockPlugins,
      });

      const { result } = renderHook(() => usePluginOptions(1, "test-agent"));

      await waitFor(() => {
        expect(result.current.plugins.length).toBeGreaterThan(0);
      });

      // Reset
      act(() => {
        result.current.resetConfig();
      });

      expect(result.current.config).toEqual({});
      expect(result.current.plugins).toEqual([]);
    });
  });

  describe("dependency changes", () => {
    it("should refetch when runner changes", async () => {
      const mockPlugins: PluginCapability[] = [
        {
          name: "test-plugin",
          version: "1.0.0",
          description: "Test Plugin",
          supported_agents: ["test-agent"],
          ui: { configurable: true, fields: [] },
        },
      ];

      vi.mocked(runnerApi.getPluginOptions).mockResolvedValue({
        plugins: mockPlugins,
      });

      const { result, rerender } = renderHook(
        ({ runner, agent }: { runner: number | null; agent: string }) => usePluginOptions(runner, agent),
        { initialProps: { runner: 1, agent: "test-agent" } }
      );

      await waitFor(() => {
        expect(result.current.loading).toBe(false);
      });

      const callsBefore = vi.mocked(runnerApi.getPluginOptions).mock.calls.length;

      // Change runner
      rerender({ runner: 2, agent: "test-agent" });

      await waitFor(() => {
        expect(vi.mocked(runnerApi.getPluginOptions).mock.calls.length).toBeGreaterThan(callsBefore);
      });
    });
  });
});
