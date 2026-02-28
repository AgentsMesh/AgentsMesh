import { request, orgPath } from "./base";

// Pod interface matching the store
export interface PodData {
  id: number;
  pod_key: string;
  status: "initializing" | "running" | "paused" | "disconnected" | "orphaned" | "completed" | "terminated" | "error" | "failed";
  agent_status: string;
  initial_prompt?: string;
  branch_name?: string;
  sandbox_path?: string;
  started_at?: string;
  finished_at?: string;
  last_activity?: string;
  created_at: string;
  title?: string; // OSC 0/2 terminal title
  runner?: {
    id: number;
    node_id: string;
    status: string;
  };
  agent_type?: {
    id: number;
    name: string;
    slug: string;
  };
  repository?: {
    id: number;
    name: string;
    full_path: string;
    provider_type?: string; // github, gitlab, gitee
  };
  ticket?: {
    id: number;
    slug: string;
    title: string;
  };
  error_code?: string;
  error_message?: string;
  created_by?: {
    id: number;
    username: string;
    name?: string;
  };
}

// Pods API
export const podApi = {
  list: (filters?: { status?: string; runnerId?: number }) => {
    const params = new URLSearchParams();
    if (filters?.status) params.append("status", filters.status);
    if (filters?.runnerId) params.append("runner_id", String(filters.runnerId));
    const query = params.toString() ? `?${params.toString()}` : "";
    return request<{ pods: PodData[]; total: number }>(`${orgPath("/pods")}${query}`);
  },

  get: (key: string) =>
    request<{ pod: PodData }>(`${orgPath("/pods")}/${key}`),

  create: (data: {
    agent_type_id?: number; // Required unless resuming
    runner_id?: number;
    repository_id?: number;
    ticket_slug?: string;
    initial_prompt?: string;
    branch_name?: string;
    config_overrides?: Record<string, unknown>;
    credential_profile_id?: number; // User's credential profile ID (undefined = RunnerHost mode)
    cols?: number; // Terminal columns (from xterm.js)
    rows?: number; // Terminal rows (from xterm.js)
    // Resume mode fields
    source_pod_key?: string; // Pod key to resume from (enables resume mode)
    resume_agent_session?: boolean; // Whether to restore agent session (default: true when resuming)
  }) =>
    request<{ message: string; pod: PodData }>(
      orgPath("/pods"),
      {
        method: "POST",
        body: data,
      }
    ),

  terminate: (key: string) =>
    request<{ message: string }>(`${orgPath("/pods")}/${key}/terminate`, {
      method: "POST",
    }),

  // Get connection info for WebSocket terminal
  getConnectionInfo: (key: string) =>
    request<{ pod_key: string; ws_url: string; status: string }>(
      `${orgPath("/pods")}/${key}/connect`
    ),

  // Get terminal connection info via Relay
  // Returns Relay URL and token for WebSocket connection
  // Note: podKey is embedded in the token for channel routing
  getTerminalConnection: (key: string) =>
    request<{
      relay_url: string;
      token: string;
      pod_key: string;
    }>(`${orgPath("/pods")}/${key}/terminal/connect`),

  // Terminal control - observe terminal output
  observeTerminal: (key: string, lines?: number) => {
    const params = lines ? `?lines=${lines}` : "";
    return request<{
      pod_key: string;
      output: string;
      status: string;
      agent_status: string;
    }>(`${orgPath("/pods")}/${key}/terminal/observe${params}`);
  },

  // Terminal control - send input
  sendTerminalInput: (key: string, input: string) =>
    request<{ message: string }>(`${orgPath("/pods")}/${key}/terminal/input`, {
      method: "POST",
      body: { input },
    }),

  // Terminal control - resize terminal
  resizeTerminal: (key: string, cols: number, rows: number) =>
    request<{ message: string }>(`${orgPath("/pods")}/${key}/terminal/resize`, {
      method: "POST",
      body: { cols, rows },
    }),

  /**
   * Send a prompt to a pod's terminal.
   * Simulates typing the prompt text then pressing Enter:
   *   1. Send prompt text (raw characters)
   *   2. Send carriage return (\r) to submit
   *
   * This mirrors what happens when a user types a prompt in the terminal and presses Enter.
   */
  sendPrompt: async (key: string, prompt: string): Promise<{ message: string }> => {
    // Send prompt text first
    await request<{ message: string }>(`${orgPath("/pods")}/${key}/terminal/input`, {
      method: "POST",
      body: { input: prompt },
    });
    // Then send Enter (carriage return) to submit
    return request<{ message: string }>(`${orgPath("/pods")}/${key}/terminal/input`, {
      method: "POST",
      body: { input: "\r" },
    });
  },
};
