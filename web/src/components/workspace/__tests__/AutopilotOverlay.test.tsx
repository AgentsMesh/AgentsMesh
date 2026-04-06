import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";
import { AutopilotOverlay } from "../AutopilotOverlay";
import type { AutopilotController, AutopilotThinking } from "@/stores/autopilot";

// Mock stores
const mockSetBottomPanelOpen = vi.fn();
const mockSetBottomPanelTab = vi.fn();
let mockController: AutopilotController | undefined;
let mockThinkingMap: Record<string, AutopilotThinking | null> = {};

vi.mock("@/stores/autopilot", () => ({
  useAutopilotStore: (selector?: (s: Record<string, unknown>) => unknown) => {
    const state = {
      getAutopilotControllerByPodKey: (podKey: string) =>
        mockController?.pod_key === podKey ? mockController : undefined,
      thinking: mockThinkingMap,
    };
    return selector ? selector(state) : state;
  },
}));

vi.mock("@/stores/ide", () => ({
  useIDEStore: (selector?: (s: Record<string, unknown>) => unknown) => {
    const state = {
      setBottomPanelOpen: mockSetBottomPanelOpen,
      setBottomPanelTab: mockSetBottomPanelTab,
    };
    return selector ? selector(state) : state;
  },
}));

// Mock autopilot child components
vi.mock("@/components/autopilot", () => ({
  TakeoverBanner: ({ autopilotController, className }: { autopilotController: AutopilotController; className?: string }) => (
    <div data-testid="takeover-banner" data-class={className}>
      {autopilotController.phase}
    </div>
  ),
  CircuitBreakerAlert: ({ autopilotController, className }: { autopilotController: AutopilotController; className?: string }) => (
    <div data-testid="circuit-breaker" data-class={className}>
      {autopilotController.circuit_breaker?.state}
    </div>
  ),
  AutopilotStatusBar: ({ onTogglePanel }: { autopilotController: AutopilotController; onTogglePanel?: () => void }) => (
    <div data-testid="status-bar">
      <button data-testid="toggle-panel" onClick={onTogglePanel}>Toggle</button>
    </div>
  ),
}));

const createController = (
  phase: AutopilotController["phase"],
  overrides: Partial<AutopilotController> = {},
): AutopilotController => ({
  id: 1,
  autopilot_controller_key: "ctrl-key-1",
  pod_key: "pod-1",
  phase,
  current_iteration: 1,
  max_iterations: 10,
  user_takeover: false,
  circuit_breaker: { state: "closed", reason: undefined },
  prompt: "test",
  created_at: new Date().toISOString(),
  ...overrides,
});

describe("AutopilotOverlay", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockController = undefined;
    mockThinkingMap = {};
  });

  it("renders nothing when no autopilot controller exists", () => {
    const { container } = render(<AutopilotOverlay podKey="pod-1" />);
    expect(container.innerHTML).toBe("");
  });

  it("renders TakeoverBanner, CircuitBreakerAlert, and AutopilotStatusBar when controller exists", () => {
    mockController = createController("running");
    render(<AutopilotOverlay podKey="pod-1" />);

    expect(screen.getByTestId("takeover-banner")).toBeInTheDocument();
    expect(screen.getByTestId("circuit-breaker")).toBeInTheDocument();
    expect(screen.getByTestId("status-bar")).toBeInTheDocument();
  });

  it("passes correct className to TakeoverBanner", () => {
    mockController = createController("running");
    render(<AutopilotOverlay podKey="pod-1" />);

    expect(screen.getByTestId("takeover-banner")).toHaveAttribute("data-class", "rounded-none");
  });

  it("passes correct className to CircuitBreakerAlert", () => {
    mockController = createController("running");
    render(<AutopilotOverlay podKey="pod-1" />);

    expect(screen.getByTestId("circuit-breaker")).toHaveAttribute("data-class", "mx-2 mt-2 rounded-md");
  });

  it("opens bottom panel when toggle button is clicked", () => {
    mockController = createController("running");
    render(<AutopilotOverlay podKey="pod-1" />);

    screen.getByTestId("toggle-panel").click();

    expect(mockSetBottomPanelTab).toHaveBeenCalledWith("autopilot");
    expect(mockSetBottomPanelOpen).toHaveBeenCalledWith(true);
  });

  describe("auto-open BottomPanel effect", () => {
    it("opens panel when thinking decision_type is need_help", () => {
      mockController = createController("running");
      mockThinkingMap = {
        "ctrl-key-1": {
          autopilot_controller_key: "ctrl-key-1",
          iteration: 1,
          decision_type: "need_help",
          reasoning: "stuck",
        } as AutopilotThinking,
      };

      render(<AutopilotOverlay podKey="pod-1" />);

      expect(mockSetBottomPanelTab).toHaveBeenCalledWith("autopilot");
      expect(mockSetBottomPanelOpen).toHaveBeenCalledWith(true);
    });

    it("opens panel when thinking decision_type is NEED_HUMAN_HELP", () => {
      mockController = createController("running");
      mockThinkingMap = {
        "ctrl-key-1": {
          autopilot_controller_key: "ctrl-key-1",
          iteration: 1,
          decision_type: "NEED_HUMAN_HELP",
          reasoning: "stuck",
        } as AutopilotThinking,
      };

      render(<AutopilotOverlay podKey="pod-1" />);

      expect(mockSetBottomPanelTab).toHaveBeenCalledWith("autopilot");
      expect(mockSetBottomPanelOpen).toHaveBeenCalledWith(true);
    });

    it("opens panel when phase is waiting_approval", () => {
      mockController = createController("waiting_approval");

      render(<AutopilotOverlay podKey="pod-1" />);

      expect(mockSetBottomPanelTab).toHaveBeenCalledWith("autopilot");
      expect(mockSetBottomPanelOpen).toHaveBeenCalledWith(true);
    });

    it("does not open panel for normal running state", () => {
      mockController = createController("running");

      render(<AutopilotOverlay podKey="pod-1" />);

      expect(mockSetBottomPanelTab).not.toHaveBeenCalled();
      expect(mockSetBottomPanelOpen).not.toHaveBeenCalled();
    });
  });
});
