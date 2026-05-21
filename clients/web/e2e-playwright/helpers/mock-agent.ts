import type { ApiFixture } from "../fixtures/api.fixture";
import { TEST_ORG_SLUG, getApiBaseUrl } from "./env";

// Slug of the built-in e2e-mock-agent AgentFile, owned by the
// universal-mock plan. See backend/migrations/000151_e2e_echo_dual_mode.
const E2E_AGENT_SLUG = "e2e-echo";
const PODS_BASE = `/api/v1/orgs/${TEST_ORG_SLUG}/pods`;

export type MockAgentMode = "pty" | "acp";

// Scenario names registered in //runner/internal/agents/mockagent/scenarios.go.
// Keep in sync with that file and backend/migrations/000151..000153.
export type MockAgentScenario =
  | "echo"
  | "streaming_3"
  | "thinking_then_answer"
  | "tool_call_edit"
  | "permission_request_edit"
  | "config_change_plan"
  | "fail_after_1s"
  | "malformed_json"
  | "tool_call_failed"
  | "log_warnings";

export interface CreateMockPodOptions {
  mode: MockAgentMode;
  scenario?: MockAgentScenario;
  prompt?: string;
  alias?: string;
}

export interface MockAgentPod {
  podKey: string;
  runnerId: number;
  cleanup: () => Promise<void>;
}

// createMockAgentPod spawns a pod backed by the e2e-mock-agent binary.
// Returns null if no runner is online — caller should `test.skip()`. The
// returned `cleanup` must be invoked from afterEach to avoid quota bleed.
export async function createMockAgentPod(
  api: ApiFixture,
  opts: CreateMockPodOptions,
): Promise<MockAgentPod | null> {
  const runner = await pickAvailableRunner(api);
  if (!runner) return null;

  const body: Record<string, unknown> = {
    runner_id: runner.id,
    agent_slug: E2E_AGENT_SLUG,
    prompt: opts.prompt ?? "",
    agentfile_layer: buildAgentfileLayer(opts),
  };
  if (opts.alias) body.alias = opts.alias;

  const res = await api.post(PODS_BASE, body);
  if (![200, 201].includes(res.status)) {
    throw new Error(`createMockAgentPod failed: ${res.status} ${await res.text()}`);
  }
  const data = await res.json();
  const podKey = data.pod_key || data.pod?.pod_key;
  if (!podKey) {
    throw new Error(`createMockAgentPod missing podKey: ${JSON.stringify(data)}`);
  }

  return {
    podKey,
    runnerId: runner.id,
    cleanup: async () => {
      try {
        await api.post(`${PODS_BASE}/${podKey}/terminate`, {});
      } catch {
        // best-effort: tests should not fail because cleanup raced
      }
    },
  };
}

interface AvailableRunner { id: number }

async function pickAvailableRunner(api: ApiFixture): Promise<AvailableRunner | null> {
  const res = await api.get(`/api/v1/orgs/${TEST_ORG_SLUG}/runners/available`);
  if (!res.ok) return null;
  const { runners } = (await res.json()) as { runners?: AvailableRunner[] };
  return runners && runners.length > 0 ? runners[0] : null;
}

function buildAgentfileLayer(opts: CreateMockPodOptions): string {
  const lines: string[] = [];
  if (opts.mode === "acp") {
    // Selects MODE acp's resolved args declared in the base AgentFile.
    lines.push("MODE acp");
  }
  if (opts.scenario && opts.scenario !== "echo") {
    lines.push(`CONFIG scenario = "${opts.scenario}"`);
  }
  return lines.length > 0 ? lines.join("\n") + "\n" : "";
}

// Returns the workspace URL for a given pod, which renders the
// AcpActivityStream / AcpPromptInput / AcpPermissionDialog stack.
export function workspaceUrlForPod(podKey: string): string {
  return `/${TEST_ORG_SLUG}/workspace/${podKey}`;
}

// getApiBaseUrl re-export for tests that need it without an extra import.
export { getApiBaseUrl };
