import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ControlButtons } from "../panel/ControlButtons";
import type { AutopilotController } from "@/stores/autopilot";

// Mock the autopilot store
const mockPauseAutopilotController = vi.fn();
const mockResumeAutopilotController = vi.fn();
const mockStopAutopilotController = vi.fn();
const mockTakeoverAutopilotController = vi.fn();
const mockHandbackAutopilotController = vi.fn();

vi.mock("@/stores/autopilot", () => ({
  useAutopilotStore: (selector?: (s: Record<string, unknown>) => unknown) => {
    const state = {
      pauseAutopilotController: mockPauseAutopilotController,
      resumeAutopilotController: mockResumeAutopilotController,
      stopAutopilotController: mockStopAutopilotController,
      takeoverAutopilotController: mockTakeoverAutopilotController,
      handbackAutopilotController: mockHandbackAutopilotController,
    };
    return selector ? selector(state) : state;
  },
}));

// Helper to create mock controller
const createMockController = (
  phase: AutopilotController["phase"]
): AutopilotController => ({
  id: 1,
  autopilot_controller_key: "test-key-123",
  pod_key: "pod-123",
  phase,
  current_iteration: 3,
  max_iterations: 10,
  user_takeover: phase === "user_takeover",
  circuit_breaker: {
    state: "closed",
    reason: undefined,
  },
  initial_prompt: "Test task",
  created_at: new Date().toISOString(),
});

describe("ControlButtons", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("running phase", () => {
    it("should show Pause button", () => {
      render(<ControlButtons autopilotController={createMockController("running")} />);
      expect(screen.getByText("Pause")).toBeInTheDocument();
    });

    it("should show Takeover button", () => {
      render(<ControlButtons autopilotController={createMockController("running")} />);
      expect(screen.getByText("Takeover")).toBeInTheDocument();
    });

    it("should show Stop button", () => {
      render(<ControlButtons autopilotController={createMockController("running")} />);
      expect(screen.getByText("Stop")).toBeInTheDocument();
    });

    it("should not show Resume button", () => {
      render(<ControlButtons autopilotController={createMockController("running")} />);
      expect(screen.queryByText("Resume")).not.toBeInTheDocument();
    });

    it("should call pauseAutopilotController when Pause clicked", () => {
      render(<ControlButtons autopilotController={createMockController("running")} />);
      fireEvent.click(screen.getByText("Pause"));
      expect(mockPauseAutopilotController).toHaveBeenCalledWith("test-key-123");
    });

    it("should call takeoverAutopilotController when Takeover clicked", () => {
      render(<ControlButtons autopilotController={createMockController("running")} />);
      fireEvent.click(screen.getByText("Takeover"));
      expect(mockTakeoverAutopilotController).toHaveBeenCalledWith("test-key-123");
    });

    it("should call stopAutopilotController when Stop clicked", () => {
      render(<ControlButtons autopilotController={createMockController("running")} />);
      fireEvent.click(screen.getByText("Stop"));
      expect(mockStopAutopilotController).toHaveBeenCalledWith("test-key-123");
    });
  });

  describe("paused phase", () => {
    it("should show Resume button", () => {
      render(<ControlButtons autopilotController={createMockController("paused")} />);
      expect(screen.getByText("Resume")).toBeInTheDocument();
    });

    it("should show Stop button", () => {
      render(<ControlButtons autopilotController={createMockController("paused")} />);
      expect(screen.getByText("Stop")).toBeInTheDocument();
    });

    it("should not show Pause button", () => {
      render(<ControlButtons autopilotController={createMockController("paused")} />);
      expect(screen.queryByText("Pause")).not.toBeInTheDocument();
    });

    it("should not show Takeover button", () => {
      render(<ControlButtons autopilotController={createMockController("paused")} />);
      expect(screen.queryByText("Takeover")).not.toBeInTheDocument();
    });

    it("should call resumeAutopilotController when Resume clicked", () => {
      render(<ControlButtons autopilotController={createMockController("paused")} />);
      fireEvent.click(screen.getByText("Resume"));
      expect(mockResumeAutopilotController).toHaveBeenCalledWith("test-key-123");
    });
  });

  describe("user_takeover phase", () => {
    it("should show Handback button", () => {
      render(<ControlButtons autopilotController={createMockController("user_takeover")} />);
      expect(screen.getByText("Handback")).toBeInTheDocument();
    });

    it("should show Stop button", () => {
      render(<ControlButtons autopilotController={createMockController("user_takeover")} />);
      expect(screen.getByText("Stop")).toBeInTheDocument();
    });

    it("should not show Pause or Takeover buttons", () => {
      render(<ControlButtons autopilotController={createMockController("user_takeover")} />);
      expect(screen.queryByText("Pause")).not.toBeInTheDocument();
      expect(screen.queryByText("Takeover")).not.toBeInTheDocument();
    });

    it("should call handbackAutopilotController when Handback clicked", () => {
      render(<ControlButtons autopilotController={createMockController("user_takeover")} />);
      fireEvent.click(screen.getByText("Handback"));
      expect(mockHandbackAutopilotController).toHaveBeenCalledWith("test-key-123");
    });
  });

  describe("initializing phase", () => {
    it("should show Stop button", () => {
      render(<ControlButtons autopilotController={createMockController("initializing")} />);
      expect(screen.getByText("Stop")).toBeInTheDocument();
    });

    it("should not show Pause, Resume, Takeover, or Handback buttons", () => {
      render(<ControlButtons autopilotController={createMockController("initializing")} />);
      expect(screen.queryByText("Pause")).not.toBeInTheDocument();
      expect(screen.queryByText("Resume")).not.toBeInTheDocument();
      expect(screen.queryByText("Takeover")).not.toBeInTheDocument();
      expect(screen.queryByText("Handback")).not.toBeInTheDocument();
    });
  });

  describe("waiting_approval phase", () => {
    it("should not show Stop button", () => {
      render(<ControlButtons autopilotController={createMockController("waiting_approval")} />);
      expect(screen.queryByText("Stop")).not.toBeInTheDocument();
    });

    it("should not show any control buttons", () => {
      render(<ControlButtons autopilotController={createMockController("waiting_approval")} />);
      expect(screen.queryByText("Pause")).not.toBeInTheDocument();
      expect(screen.queryByText("Resume")).not.toBeInTheDocument();
      expect(screen.queryByText("Takeover")).not.toBeInTheDocument();
      expect(screen.queryByText("Handback")).not.toBeInTheDocument();
    });
  });

  describe("completed phase", () => {
    it("should not show any buttons", () => {
      render(<ControlButtons autopilotController={createMockController("completed")} />);
      expect(screen.queryByText("Pause")).not.toBeInTheDocument();
      expect(screen.queryByText("Resume")).not.toBeInTheDocument();
      expect(screen.queryByText("Takeover")).not.toBeInTheDocument();
      expect(screen.queryByText("Handback")).not.toBeInTheDocument();
      expect(screen.queryByText("Stop")).not.toBeInTheDocument();
    });
  });

  describe("failed phase", () => {
    it("should not show any buttons", () => {
      render(<ControlButtons autopilotController={createMockController("failed")} />);
      expect(screen.queryByText("Pause")).not.toBeInTheDocument();
      expect(screen.queryByText("Stop")).not.toBeInTheDocument();
    });
  });

  describe("stopped phase", () => {
    it("should not show any buttons", () => {
      render(<ControlButtons autopilotController={createMockController("stopped")} />);
      expect(screen.queryByText("Pause")).not.toBeInTheDocument();
      expect(screen.queryByText("Stop")).not.toBeInTheDocument();
    });
  });

  describe("max_iterations phase", () => {
    it("should not show any buttons", () => {
      render(<ControlButtons autopilotController={createMockController("max_iterations")} />);
      expect(screen.queryByText("Pause")).not.toBeInTheDocument();
      expect(screen.queryByText("Stop")).not.toBeInTheDocument();
    });
  });
});
