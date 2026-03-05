import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { HistoryTab } from "../panel/HistoryTab";
import type { AutopilotIteration } from "@/stores/autopilot";

// Mock the autopilot store
const mockFetchIterations = vi.fn();
const mockIterations: Record<string, AutopilotIteration[]> = {};

vi.mock("@/stores/autopilot", () => ({
  useAutopilotStore: (selector?: (s: Record<string, unknown>) => unknown) => {
    const state = {
      iterations: mockIterations,
      fetchIterations: mockFetchIterations,
    };
    return selector ? selector(state) : state;
  },
}));

// Helper to create mock iteration
const createMockIteration = (overrides: Partial<AutopilotIteration> = {}): AutopilotIteration => ({
  id: 1,
  autopilot_controller_id: 1,
  iteration: 1,
  phase: "completed",
  summary: undefined,
  files_changed: undefined,
  duration_ms: undefined,
  created_at: new Date().toISOString(),
  ...overrides,
});

describe("HistoryTab", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Clear mock iterations
    Object.keys(mockIterations).forEach((key) => delete mockIterations[key]);
  });

  describe("empty state", () => {
    it("should show no iterations message when empty", () => {
      render(<HistoryTab autopilotControllerKey="test-key" />);
      expect(screen.getByText("No iterations yet")).toBeInTheDocument();
    });

    it("should call fetchIterations on mount", () => {
      render(<HistoryTab autopilotControllerKey="test-key" />);
      expect(mockFetchIterations).toHaveBeenCalledWith("test-key");
    });
  });

  describe("iteration list", () => {
    it("should display iterations in reverse order (most recent first)", () => {
      mockIterations["test-key"] = [
        createMockIteration({ id: 1, iteration: 1, phase: "completed" }),
        createMockIteration({ id: 2, iteration: 2, phase: "completed" }),
        createMockIteration({ id: 3, iteration: 3, phase: "control_running" }),
      ];

      render(<HistoryTab autopilotControllerKey="test-key" />);

      const iterationNumbers = screen.getAllByText(/#\d+/);
      expect(iterationNumbers[0]).toHaveTextContent("#3");
      expect(iterationNumbers[1]).toHaveTextContent("#2");
      expect(iterationNumbers[2]).toHaveTextContent("#1");
    });

    it("should display iteration phase badge", () => {
      mockIterations["test-key"] = [
        createMockIteration({ phase: "completed" }),
      ];

      render(<HistoryTab autopilotControllerKey="test-key" />);
      expect(screen.getByText("Done")).toBeInTheDocument();
    });

    it("should display different phase labels", () => {
      mockIterations["test-key"] = [
        createMockIteration({ id: 1, iteration: 1, phase: "initial_prompt" }),
        createMockIteration({ id: 2, iteration: 2, phase: "control_running" }),
        createMockIteration({ id: 3, iteration: 3, phase: "error" }),
      ];

      render(<HistoryTab autopilotControllerKey="test-key" />);
      expect(screen.getByText("Initial")).toBeInTheDocument();
      expect(screen.getByText("Running")).toBeInTheDocument();
      expect(screen.getByText("Error")).toBeInTheDocument();
    });

    it("should display duration when available", () => {
      mockIterations["test-key"] = [
        createMockIteration({ duration_ms: 2500 }),
      ];

      render(<HistoryTab autopilotControllerKey="test-key" />);
      expect(screen.getByText("2.5s")).toBeInTheDocument();
    });

    it("should not display duration when undefined", () => {
      mockIterations["test-key"] = [
        createMockIteration({ duration_ms: undefined }),
      ];

      render(<HistoryTab autopilotControllerKey="test-key" />);
      expect(screen.queryByText(/\d+\.\d+s/)).not.toBeInTheDocument();
    });
  });

  describe("expandable details", () => {
    it("should expand to show summary when clicked", () => {
      mockIterations["test-key"] = [
        createMockIteration({
          summary: "Created hello.py file",
          files_changed: undefined,
        }),
      ];

      render(<HistoryTab autopilotControllerKey="test-key" />);

      // Summary should not be visible initially
      expect(screen.queryByText("Created hello.py file")).not.toBeInTheDocument();

      // Click to expand
      const expandableRow = screen.getByText("Done").closest("div")?.parentElement;
      if (expandableRow) {
        fireEvent.click(expandableRow);
      }

      // Summary should now be visible
      expect(screen.getByText("Created hello.py file")).toBeInTheDocument();
    });

    it("should expand when only files_changed is present (no summary)", () => {
      mockIterations["test-key"] = [
        createMockIteration({
          summary: undefined,
          files_changed: ["src/file.ts"],
        }),
      ];

      render(<HistoryTab autopilotControllerKey="test-key" />);

      // Click to expand
      const expandableRow = screen.getByText("Done").closest("div")?.parentElement;
      if (expandableRow) {
        fireEvent.click(expandableRow);
      }

      expect(screen.getByText("Files:")).toBeInTheDocument();
      expect(screen.getByText("src/file.ts")).toBeInTheDocument();
    });

    it("should handle empty files_changed array", () => {
      mockIterations["test-key"] = [
        createMockIteration({
          summary: "Test summary",
          files_changed: [],
        }),
      ];

      render(<HistoryTab autopilotControllerKey="test-key" />);

      // Click to expand
      const expandableRow = screen.getByText("Done").closest("div")?.parentElement;
      if (expandableRow) {
        fireEvent.click(expandableRow);
      }

      // Summary should be visible but not Files section (empty array)
      expect(screen.getByText("Test summary")).toBeInTheDocument();
      expect(screen.queryByText("Files:")).not.toBeInTheDocument();
    });

    it("should expand to show files changed", () => {
      mockIterations["test-key"] = [
        createMockIteration({
          summary: "Modified files",
          files_changed: ["src/index.ts", "package.json"],
        }),
      ];

      render(<HistoryTab autopilotControllerKey="test-key" />);

      // Click to expand
      const expandableRow = screen.getByText("Done").closest("div")?.parentElement;
      if (expandableRow) {
        fireEvent.click(expandableRow);
      }

      expect(screen.getByText("Files:")).toBeInTheDocument();
      expect(screen.getByText("src/index.ts, package.json")).toBeInTheDocument();
    });

    it("should collapse when clicked again", () => {
      mockIterations["test-key"] = [
        createMockIteration({
          summary: "Test summary",
          files_changed: undefined,
        }),
      ];

      render(<HistoryTab autopilotControllerKey="test-key" />);

      const expandableRow = screen.getByText("Done").closest("div")?.parentElement;
      if (expandableRow) {
        // Expand
        fireEvent.click(expandableRow);
        expect(screen.getByText("Test summary")).toBeInTheDocument();

        // Collapse
        fireEvent.click(expandableRow);
        expect(screen.queryByText("Test summary")).not.toBeInTheDocument();
      }
    });

    it("should not be expandable when no details available", () => {
      mockIterations["test-key"] = [
        createMockIteration({
          summary: undefined,
          files_changed: undefined,
        }),
      ];

      render(<HistoryTab autopilotControllerKey="test-key" />);

      // Should not have chevron icons for non-expandable items
      const chevronRight = document.querySelector(".lucide-chevron-right");
      const chevronDown = document.querySelector(".lucide-chevron-down");
      expect(chevronRight).not.toBeInTheDocument();
      expect(chevronDown).not.toBeInTheDocument();
    });
  });

  describe("unknown phase handling", () => {
    it("should handle unknown phase gracefully", () => {
      mockIterations["test-key"] = [
        createMockIteration({ phase: "unknown_phase" as AutopilotIteration["phase"] }),
      ];

      render(<HistoryTab autopilotControllerKey="test-key" />);
      // Should display the phase name as-is for unknown phases
      expect(screen.getByText("unknown_phase")).toBeInTheDocument();
    });
  });

  describe("refetch on key change", () => {
    it("should fetch iterations when autopilotControllerKey changes", () => {
      const { rerender } = render(<HistoryTab autopilotControllerKey="key-1" />);
      expect(mockFetchIterations).toHaveBeenCalledWith("key-1");

      rerender(<HistoryTab autopilotControllerKey="key-2" />);
      expect(mockFetchIterations).toHaveBeenCalledWith("key-2");
    });

    it("should not fetch when autopilotControllerKey is empty string", () => {
      render(<HistoryTab autopilotControllerKey="" />);
      expect(mockFetchIterations).not.toHaveBeenCalled();
    });
  });

  describe("iteration key handling", () => {
    it("should use iteration.iteration as key when id is undefined", () => {
      mockIterations["test-key"] = [
        createMockIteration({ id: undefined, iteration: 1, phase: "completed" }),
        createMockIteration({ id: undefined, iteration: 2, phase: "completed" }),
      ];

      render(<HistoryTab autopilotControllerKey="test-key" />);

      // Both iterations should be rendered (no key collision)
      const iterationNumbers = screen.getAllByText(/#\d+/);
      expect(iterationNumbers).toHaveLength(2);
    });
  });
});
