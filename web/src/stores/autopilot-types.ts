import type {
  AutopilotControllerData,
  AutopilotIterationData,
  CreateAutopilotControllerRequest,
  ApproveRequest,
} from "@/lib/api/autopilot";
import type { AutopilotThinkingData } from "@/lib/realtime/types";

// Re-export types for component use
export type AutopilotController = AutopilotControllerData;
export type AutopilotIteration = AutopilotIterationData;
export type AutopilotThinking = AutopilotThinkingData;

// Re-export request types for the store
export type { CreateAutopilotControllerRequest, ApproveRequest };

export interface AutopilotState {
  // State
  autopilotControllers: AutopilotController[];
  currentAutopilotController: AutopilotController | null;
  iterations: Record<string, AutopilotIteration[]>; // keyed by autopilot_controller_key
  thinking: Record<string, AutopilotThinking | null>; // Latest thinking event per autopilot
  thinkingHistory: Record<string, AutopilotThinking[]>; // All thinking events per autopilot
  loading: boolean;
  error: string | null;

  // Actions
  fetchAutopilotControllers: () => Promise<void>;
  fetchAutopilotController: (key: string) => Promise<void>;
  createAutopilotController: (data: CreateAutopilotControllerRequest) => Promise<AutopilotController>;
  pauseAutopilotController: (key: string) => Promise<void>;
  resumeAutopilotController: (key: string) => Promise<void>;
  stopAutopilotController: (key: string) => Promise<void>;
  approveAutopilotController: (key: string, data?: ApproveRequest) => Promise<void>;
  takeoverAutopilotController: (key: string) => Promise<void>;
  handbackAutopilotController: (key: string) => Promise<void>;
  fetchIterations: (key: string) => Promise<void>;

  // Real-time updates (called from RealtimeProvider)
  updateAutopilotControllerStatus: (
    key: string,
    phase: string,
    currentIteration: number,
    maxIterations: number,
    circuitBreakerState: string,
    circuitBreakerReason?: string
  ) => void;
  addIteration: (key: string, iteration: AutopilotIteration) => void;
  updateThinking: (key: string, thinking: AutopilotThinking) => void;
  setCurrentAutopilotController: (controller: AutopilotController | null) => void;
  removeAutopilotController: (key: string) => void;

  // Error handling
  clearError: () => void;

  // Selectors
  getAutopilotControllerByPodKey: (podKey: string) => AutopilotController | undefined;
  getThinking: (key: string) => AutopilotThinking | null;
  getThinkingHistory: (key: string) => AutopilotThinking[];
}
