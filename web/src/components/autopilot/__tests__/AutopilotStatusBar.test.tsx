import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { AutopilotStatusBar } from "../AutopilotStatusBar";
import type { AutopilotController, AutopilotThinking } from "@/stores/autopilot";

// Mock the autopilot store
const mockPauseAutopilotController = vi.fn();
const mockResumeAutopilotController = vi.fn();
const mockStopAutopilotController = vi.fn();
const mockTakeoverAutopilotController = vi.fn();
const mockHandbackAutopilotController = vi.fn();
let mockThinking: AutopilotThinking | null = null;

vi.mock("@/stores/autopilot", () => ({
  useAutopilotStore: (selector?: (s: Record<string, unknown>) => unknown) => {
    const state = {
      pauseAutopilotController: mockPauseAutopilotController,
      resumeAutopilotController: mockResumeAutopilotController,
      stopAutopilotController: mockStopAutopilotController,
      takeoverAutopilotController: mockTakeoverAutopilotController,
      handbackAutopilotController: mockHandbackAutopilotController,
      thinking: { "test-key-123": mockThinking } as Record<string, AutopilotThinking | null>,
    };
    return selector ? selector(state) : state;
  },
}));

// Helper to create mock controller
const createMockController = (
  phase: AutopilotController["phase"],
  overrides: Partial<AutopilotController> = {}
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
  prompt: "Test task",
  created_at: new Date().toISOString(),
  ...overrides,
});

// Helper to create mock thinking
const createMockThinking = (overrides: Partial<AutopilotThinking> = {}): AutopilotThinking => ({
  autopilot_controller_key: "test-key-123",
  iteration: 3,
  decision_type: "CONTINUE",
  reasoning: "Processing the task...",
  confidence: 0.85,
  action: undefined,
  progress: undefined,
  help_request: undefined,
  ...overrides,
});

describe("AutopilotStatusBar", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockThinking = null;
  });

  describe("phase display", () => {
    it("should display Running for running phase", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("running")} />);
      expect(screen.getByText("Running")).toBeInTheDocument();
    });

    it("should display Paused for paused phase", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("paused")} />);
      expect(screen.getByText("Paused")).toBeInTheDocument();
    });

    it("should display Initializing for initializing phase", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("initializing")} />);
      expect(screen.getByText("Initializing")).toBeInTheDocument();
    });

    it("should display User Control for user_takeover phase", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("user_takeover")} />);
      expect(screen.getByText("User Control")).toBeInTheDocument();
    });

    it("should display Completed for completed phase", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("completed")} />);
      expect(screen.getByText("Completed")).toBeInTheDocument();
    });

    it("should display Failed for failed phase", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("failed")} />);
      expect(screen.getByText("Failed")).toBeInTheDocument();
    });

    it("should display Stopped for stopped phase", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("stopped")} />);
      expect(screen.getByText("Stopped")).toBeInTheDocument();
    });

    it("should display Max Iterations for max_iterations phase", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("max_iterations")} />);
      expect(screen.getByText("Max Iterations")).toBeInTheDocument();
    });
  });

  describe("progress display", () => {
    it("should display iteration progress", () => {
      render(
        <AutopilotStatusBar
          autopilotController={createMockController("running", {
            current_iteration: 5,
            max_iterations: 15,
          })}
        />
      );
      expect(screen.getByText("5/15")).toBeInTheDocument();
    });

    it("should show progress bar", () => {
      const { container } = render(
        <AutopilotStatusBar autopilotController={createMockController("running")} />
      );
      expect(container.querySelector('[role="progressbar"]')).toBeInTheDocument();
    });
  });

  describe("reasoning display", () => {
    it("should display waiting message when no thinking", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("running")} />);
      expect(screen.getByText("Waiting for Control Agent...")).toBeInTheDocument();
    });

    it("should display reasoning text when thinking available", () => {
      mockThinking = createMockThinking({ reasoning: "Processing the task..." });
      render(<AutopilotStatusBar autopilotController={createMockController("running")} />);
      expect(screen.getByText("Processing the task...")).toBeInTheDocument();
    });

    it("should truncate long reasoning text", () => {
      const longReasoning = "A".repeat(100);
      mockThinking = createMockThinking({ reasoning: longReasoning });
      render(<AutopilotStatusBar autopilotController={createMockController("running")} />);
      // Should be truncated to 60 chars + "..."
      expect(screen.getByText("A".repeat(60) + "...")).toBeInTheDocument();
    });
  });

  describe("need help state", () => {
    it("should display Need Help when decision_type is need_help", () => {
      mockThinking = createMockThinking({ decision_type: "need_help" });
      render(<AutopilotStatusBar autopilotController={createMockController("running")} />);
      expect(screen.getByText("Need Help")).toBeInTheDocument();
    });

    it("should display Need Help when decision_type is NEED_HUMAN_HELP", () => {
      mockThinking = createMockThinking({ decision_type: "NEED_HUMAN_HELP" });
      render(<AutopilotStatusBar autopilotController={createMockController("running")} />);
      expect(screen.getByText("Need Help")).toBeInTheDocument();
    });

    it("should display Need Help when phase is waiting_approval", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("waiting_approval")} />);
      expect(screen.getByText("Need Help")).toBeInTheDocument();
    });
  });

  describe("control buttons", () => {
    it("should call onTogglePanel when View Details clicked", () => {
      const onTogglePanel = vi.fn();
      render(
        <AutopilotStatusBar
          autopilotController={createMockController("running")}
          onTogglePanel={onTogglePanel}
        />
      );
      const viewDetailsButton = screen.getByTitle("View Details");
      fireEvent.click(viewDetailsButton);
      expect(onTogglePanel).toHaveBeenCalled();
    });

    it("should show Pause button in running phase", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("running")} />);
      expect(screen.getByTitle("Pause")).toBeInTheDocument();
    });

    it("should call pauseAutopilotController when Pause clicked", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("running")} />);
      fireEvent.click(screen.getByTitle("Pause"));
      expect(mockPauseAutopilotController).toHaveBeenCalledWith("test-key-123");
    });

    it("should show Resume button in paused phase", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("paused")} />);
      expect(screen.getByTitle("Resume")).toBeInTheDocument();
    });

    it("should call resumeAutopilotController when Resume clicked", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("paused")} />);
      fireEvent.click(screen.getByTitle("Resume"));
      expect(mockResumeAutopilotController).toHaveBeenCalledWith("test-key-123");
    });

    it("should show Takeover button in running phase", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("running")} />);
      expect(screen.getByTitle("Takeover Control")).toBeInTheDocument();
    });

    it("should call takeoverAutopilotController when Takeover clicked", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("running")} />);
      fireEvent.click(screen.getByTitle("Takeover Control"));
      expect(mockTakeoverAutopilotController).toHaveBeenCalledWith("test-key-123");
    });

    it("should show Handback button in user_takeover phase", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("user_takeover")} />);
      expect(screen.getByTitle("Handback to Autopilot")).toBeInTheDocument();
    });

    it("should call handbackAutopilotController when Handback clicked", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("user_takeover")} />);
      fireEvent.click(screen.getByTitle("Handback to Autopilot"));
      expect(mockHandbackAutopilotController).toHaveBeenCalledWith("test-key-123");
    });

    it("should show Stop button in active phases", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("running")} />);
      expect(screen.getByTitle("Stop Autopilot")).toBeInTheDocument();
    });

    it("should call stopAutopilotController when Stop clicked", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("running")} />);
      fireEvent.click(screen.getByTitle("Stop Autopilot"));
      expect(mockStopAutopilotController).toHaveBeenCalledWith("test-key-123");
    });

    it("should not show Stop button in waiting_approval phase", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("waiting_approval")} />);
      expect(screen.queryByTitle("Stop Autopilot")).not.toBeInTheDocument();
    });

    it("should not show Stop button in completed phase", () => {
      render(<AutopilotStatusBar autopilotController={createMockController("completed")} />);
      expect(screen.queryByTitle("Stop Autopilot")).not.toBeInTheDocument();
    });
  });

  describe("className prop", () => {
    it("should apply custom className", () => {
      const { container } = render(
        <AutopilotStatusBar
          autopilotController={createMockController("running")}
          className="custom-class"
        />
      );
      expect(container.firstChild).toHaveClass("custom-class");
    });
  });
});
