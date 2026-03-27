import { podApi, PodData } from "@/lib/api";

/**
 * Builds the API request payload and submits the pod creation request.
 * Returns the created PodData or null.
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
  options?: { ticketSlug?: string; initialPrompt?: string; cols?: number; rows?: number };
}): Promise<PodData | null> {
  const {
    selectedAgent,
    selectedAgentSlug,
    selectedRepository,
    selectedBranch,
    selectedCredentialProfile,
    interactionMode,
    prompt,
    alias,
    selectedRunnerId,
    pluginConfig,
    options,
  } = params;

  // Build plugin config for API
  const config: Record<string, unknown> = {
    agent: selectedAgentSlug,
    ...pluginConfig,
  };

  // Use provided initialPrompt (from options) or form prompt
  const finalPrompt = options?.initialPrompt ?? prompt;

  const response = await podApi.create({
    agent_slug: selectedAgent,
    runner_id: selectedRunnerId || undefined, // omit when not manually selected
    repository_id: selectedRepository || undefined,
    branch_name: selectedBranch || undefined,
    initial_prompt: finalPrompt,
    alias: alias.trim() || undefined,
    config_overrides: config,
    credential_profile_id: selectedCredentialProfile,
    ticket_slug: options?.ticketSlug,
    cols: options?.cols,
    rows: options?.rows,
    interaction_mode: interactionMode !== "pty" ? interactionMode : undefined,
  });

  return response.pod || null;
}
