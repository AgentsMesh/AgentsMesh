import { ScenarioContext, PromptGenerator, CreatePodFormConfig } from "./types";

export const ticketPromptGenerator: PromptGenerator = (context: ScenarioContext): string => {
  if (!context.ticket) return "";

  const { slug, title, description } = context.ticket;

  let prompt = `Work on ticket ${slug}: ${title}`;

  if (description) {
    const truncated =
      description.length > 500
        ? description.substring(0, 500) + "..."
        : description;
    prompt += `\n\nTicket Description:\n${truncated}`;
  }

  return prompt;
};

export const workspacePromptGenerator: PromptGenerator = (): string => {
  return "";
};

export function getScenarioPreset(
  scenario: CreatePodFormConfig["scenario"]
): Partial<CreatePodFormConfig> {
  switch (scenario) {
    case "ticket":
      return {
        scenario: "ticket",
        promptGenerator: ticketPromptGenerator,
      };
    case "workspace":
    default:
      return {
        scenario: "workspace",
        promptGenerator: workspacePromptGenerator,
      };
  }
}

export function mergeConfig(
  config: CreatePodFormConfig
): CreatePodFormConfig {
  const preset = getScenarioPreset(config.scenario);
  return {
    ...preset,
    ...config,
    promptGenerator: config.promptGenerator ?? preset.promptGenerator,
  };
}
