import type { PodData } from "@/lib/api";
import { getPodService } from "@/lib/wasm-core";

/**
 * Builds the API request payload and submits the pod creation request.
 *
 * All pod configuration (MODE, CONFIG, REPO, BRANCH, CREDENTIAL, PROMPT, etc.)
 * is conveyed through `agentfileLayer` (AgentFile SSOT).
 */
export async function submitCreatePod(params: {
  selectedAgent: string;
  alias: string;
  perpetual?: boolean;
  selectedRunnerId: number | null | undefined;
  agentfileLayer?: string;
  options?: { ticketSlug?: string; cols?: number; rows?: number };
}): Promise<PodData | null> {
  const { selectedAgent, alias, perpetual, selectedRunnerId, agentfileLayer, options } = params;

  const pod: PodData = JSON.parse(await getPodService().create_pod(JSON.stringify({
    agent_slug: selectedAgent,
    runner_id: selectedRunnerId || undefined,
    alias: alias.trim() || undefined,
    ticket_slug: options?.ticketSlug,
    cols: options?.cols,
    rows: options?.rows,
    agentfile_layer: agentfileLayer || undefined,
    perpetual: perpetual || undefined,
  })));

  return pod || null;
}
