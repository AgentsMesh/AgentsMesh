import { podApi, PodData } from "@/lib/api";

/**
 * Builds the API request payload and submits the pod creation request.
 *
 * All pod configuration (MODE, CONFIG, REPO, BRANCH, CREDENTIAL, PROMPT, etc.)
 * is conveyed through `agentfileLayer` (AgentFile SSOT).
 */
export async function submitCreatePod(params: {
  selectedAgent: string;
  alias: string;
  selectedRunnerId: number | null | undefined;
  agentfileLayer?: string;
  options?: { ticketSlug?: string; cols?: number; rows?: number };
}): Promise<PodData | null> {
  const { selectedAgent, alias, selectedRunnerId, agentfileLayer, options } = params;

  const response = await podApi.create({
    agent_slug: selectedAgent,
    runner_id: selectedRunnerId || undefined,
    alias: alias.trim() || undefined,
    ticket_slug: options?.ticketSlug,
    cols: options?.cols,
    rows: options?.rows,
    // AgentFile Layer — SSOT (PROMPT, MODE, CONFIG, REPO, BRANCH, CREDENTIAL)
    agentfile_layer: agentfileLayer || undefined,
  });

  return response.pod || null;
}
