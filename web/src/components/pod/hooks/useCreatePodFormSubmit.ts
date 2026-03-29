import { podApi, PodData } from "@/lib/api";

/**
 * Builds the API request payload and submits the pod creation request.
 *
 * All pod configuration (MODE, CONFIG, REPO, BRANCH, CREDENTIAL, PROMPT, etc.)
 * is conveyed through `podfileLayer` (PodFile SSOT).
 */
export async function submitCreatePod(params: {
  selectedAgent: string;
  selectedAgentSlug: string;
  selectedRepository: number | null;
  selectedBranch: string;
  selectedCredentialProfile: number;
  interactionMode: "pty" | "acp";
  prompt: string;
  alias: string;
  selectedRunnerId: number | null | undefined;
  pluginConfig: Record<string, unknown>;
  podfileLayer?: string;
  options?: { ticketSlug?: string; cols?: number; rows?: number };
}): Promise<PodData | null> {
  const { selectedAgent, alias, selectedRunnerId, podfileLayer, options } = params;

  const response = await podApi.create({
    agent_slug: selectedAgent,
    runner_id: selectedRunnerId || undefined,
    alias: alias.trim() || undefined,
    ticket_slug: options?.ticketSlug,
    cols: options?.cols,
    rows: options?.rows,
    // PodFile Layer — SSOT (PROMPT, MODE, CONFIG, REPO, BRANCH, CREDENTIAL)
    podfile_layer: podfileLayer || undefined,
  });

  return response.pod || null;
}
