import { request, orgPath } from "./base";

// AutopilotController phase types
export type AutopilotPhase =
  | "initializing"
  | "running"
  | "paused"
  | "user_takeover"
  | "waiting_approval"
  | "completed"
  | "failed"
  | "stopped"
  | "max_iterations";

export type CircuitBreakerState = "closed" | "half_open" | "open";

// AutopilotController data interface
export interface AutopilotControllerData {
  id: number;
  autopilot_controller_key: string;
  pod_key: string;
  phase: AutopilotPhase;
  current_iteration: number;
  max_iterations: number;
  circuit_breaker: {
    state: CircuitBreakerState;
    reason?: string;
  };
  user_takeover: boolean;
  initial_prompt?: string;
  started_at?: string;
  last_iteration_at?: string;
  created_at: string;
}

// AutopilotIteration data interface
export interface AutopilotIterationData {
  id: number;
  autopilot_controller_id: number;
  iteration: number;
  phase: string;
  summary?: string;
  files_changed?: string[];
  duration_ms?: number;
  created_at: string;
}

// Create AutopilotController request
export interface CreateAutopilotControllerRequest {
  pod_key: string;
  initial_prompt?: string;
  max_iterations?: number;
  iteration_timeout_sec?: number;
  no_progress_threshold?: number;
  same_error_threshold?: number;
  approval_timeout_min?: number;
  control_agent_slug?: string;
  control_prompt_template?: string;
  mcp_config_json?: string;
}

// Approve request
export interface ApproveRequest {
  continue_execution?: boolean;
  additional_iterations?: number;
}

// AutopilotController API
export const autopilotApi = {
  // List all AutopilotControllers
  list: () =>
    request<AutopilotControllerData[]>(orgPath("/autopilot-controllers")),

  // Get a specific AutopilotController
  get: (key: string) =>
    request<AutopilotControllerData>(`${orgPath("/autopilot-controllers")}/${key}`),

  // Create a new AutopilotController
  create: (data: CreateAutopilotControllerRequest) =>
    request<AutopilotControllerData>(orgPath("/autopilot-controllers"), {
      method: "POST",
      body: data,
    }),

  // Pause AutopilotController
  pause: (key: string) =>
    request<{ status: string; action: string }>(
      `${orgPath("/autopilot-controllers")}/${key}/pause`,
      { method: "POST" }
    ),

  // Resume AutopilotController
  resume: (key: string) =>
    request<{ status: string; action: string }>(
      `${orgPath("/autopilot-controllers")}/${key}/resume`,
      { method: "POST" }
    ),

  // Stop AutopilotController
  stop: (key: string) =>
    request<{ status: string; action: string }>(
      `${orgPath("/autopilot-controllers")}/${key}/stop`,
      { method: "POST" }
    ),

  // Approve AutopilotController (after circuit breaker triggers)
  approve: (key: string, data?: ApproveRequest) =>
    request<{ status: string; action: string }>(
      `${orgPath("/autopilot-controllers")}/${key}/approve`,
      {
        method: "POST",
        body: data || { continue_execution: true },
      }
    ),

  // User takeover control
  takeover: (key: string) =>
    request<{ status: string; action: string }>(
      `${orgPath("/autopilot-controllers")}/${key}/takeover`,
      { method: "POST" }
    ),

  // Handback control to AutopilotController
  handback: (key: string) =>
    request<{ status: string; action: string }>(
      `${orgPath("/autopilot-controllers")}/${key}/handback`,
      { method: "POST" }
    ),

  // Get iteration history
  getIterations: (key: string) =>
    request<AutopilotIterationData[]>(
      `${orgPath("/autopilot-controllers")}/${key}/iterations`
    ),
};
