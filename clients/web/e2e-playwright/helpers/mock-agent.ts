import type { ApiFixture } from "../fixtures/api.fixture";
import { TEST_ORG_SLUG, getApiBaseUrl } from "./env";
import { pollUntil } from "./retry";

// Mirror of backend agentpod.IsPodStatusActive — the exact predicate the
// GetPodConnection gate uses (connect/pod/connection.go). A pod outside this
// set 400s with "pod is not active".
const CONNECTABLE_STATUSES = ["initializing", "running", "paused", "disconnected"];

// Slug of the built-in e2e-mock-agent AgentFile, owned by the
// universal-mock plan. See backend/migrations/000151_e2e_echo_dual_mode.
const E2E_AGENT_SLUG = "e2e-echo";

export type MockAgentMode = "pty" | "acp";

// Scenario names registered in //runner/internal/agents/mockagent/scenarios.go.
// Keep in sync with that file and backend/migrations/000151..000153.
export type MockAgentScenario =
  | "echo"
  | "autopilot"
  | "autopilot_fs"
  | "streaming_3"
  | "thinking_then_answer"
  | "tool_call_edit"
  | "permission_request_edit"
  | "config_change_plan"
  | "fail_after_1s"
  | "malformed_json"
  | "tool_call_failed"
  | "log_warnings"
  | "loopal_panels"
  | "permission_modes_loopal";

export interface CreateMockPodOptions {
  mode: MockAgentMode;
  scenario?: MockAgentScenario;
  prompt?: string;
  alias?: string;
}

export interface MockAgentPod {
  podKey: string;
  runnerId: bigint;
  cleanup: () => Promise<void>;
}

interface Runner { id: bigint }
interface Pod { podKey: string }

// createMockAgentPod spawns a pod backed by the e2e-mock-agent binary via
// Connect-RPC (PodService.CreatePod). Throws when no runner is online —
// the e2e suite contract is "dev env has at least one online runner", so
// returning null here would silently mask a missing prerequisite.
// The returned `cleanup` must be invoked from afterEach to avoid quota bleed.
export async function createMockAgentPod(
  api: ApiFixture,
  opts: CreateMockPodOptions,
): Promise<MockAgentPod> {
  const cc = await api.connect();
  const { items: runners } = await cc.runner.listAvailableRunners({ orgSlug: TEST_ORG_SLUG }) as { items?: Runner[] };
  if (!runners?.length) {
    throw new Error("createMockAgentPod: dev env must have at least one online runner");
  }
  const runnerId = runners[0].id;

  const input: Record<string, unknown> = {
    orgSlug: TEST_ORG_SLUG,
    runnerId,
    agentSlug: E2E_AGENT_SLUG,
    agentfileLayer: buildAgentfileLayer(opts),
  };
  if (opts.alias) input.alias = opts.alias;

  const resp = await cc.pod.createPod(input) as { pod?: Pod };
  const podKey = resp.pod?.podKey;
  if (!podKey) {
    throw new Error(`createMockAgentPod missing podKey: ${JSON.stringify(resp)}`);
  }

  // CreatePod returns before the runner spawns the pod process, so the pod sits
  // in a pre-active status. Navigating now would race the create→active
  // transition and GetPodConnection would 400 ("pod is not active"), surfacing
  // as an uncaught pageerror that the default-deny monitor flags. Gate on the
  // same predicate the backend connect gate uses so the returned pod is
  // connectable.
  await pollUntil(
    async () => {
      const { items } = (await cc.pod.listPods({ orgSlug: TEST_ORG_SLUG })) as {
        items?: Array<{ podKey?: string; status?: string }>;
      };
      const mine = items?.find((p) => p.podKey === podKey);
      return !!mine && CONNECTABLE_STATUSES.includes(mine.status ?? "");
    },
    { maxAttempts: 30, intervalMs: 1000, label: `pod-${podKey}-connectable` },
  );

  return {
    podKey,
    runnerId,
    cleanup: async () => {
      try {
        await cc.pod.terminatePod({ orgSlug: TEST_ORG_SLUG, podKey });
      } catch {
        // best-effort: tests should not fail because cleanup raced
      }
    },
  };
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
  // PROMPT travels through the AgentFile layer in the Connect-RPC create
  // path — CreatePodRequest does not expose a top-level prompt field.
  if (opts.prompt) {
    // Escape backslashes and double-quotes for the AgentFile single-line
    // string syntax (`PROMPT "..."`). Tests pass plain ASCII so this is
    // sufficient — no multi-line / unicode-escape handling needed.
    const safe = opts.prompt.replace(/\\/g, "\\\\").replace(/"/g, '\\"');
    lines.push(`PROMPT "${safe}"`);
  }
  return lines.length > 0 ? lines.join("\n") + "\n" : "";
}

// Returns the workspace URL for a given pod, which renders the
// AcpActivityStream / AcpPromptInput / AcpPermissionDialog stack.
// Pod selection travels via the `pod` query param — the workspace page
// reads it through useSearchParams and calls addPane(podKey) once the
// store hydrates.
export function workspaceUrlForPod(podKey: string): string {
  return `/${TEST_ORG_SLUG}/workspace?pod=${encodeURIComponent(podKey)}`;
}

// Returns the standalone Loopal control-console URL for a pod, which renders
// the bg-shell / cron / task / topology / mcp / goal panels and control bars
// (distinct from the workspace route — this is /[org]/loopal/[podKey]).
export function loopalConsoleUrlForPod(podKey: string): string {
  return `/${TEST_ORG_SLUG}/loopal/${encodeURIComponent(podKey)}`;
}

// getApiBaseUrl re-export for tests that need it without an extra import.
export { getApiBaseUrl };
