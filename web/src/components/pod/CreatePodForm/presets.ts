import { ScenarioContext, PromptGenerator, CreatePodFormConfig } from "./types";

/**
 * Ticket 场景的 Prompt 生成器
 * 根据 ticket 信息生成默认的初始 prompt
 */
export const ticketPromptGenerator: PromptGenerator = (context: ScenarioContext): string => {
  if (!context.ticket) return "";

  const { identifier, title } = context.ticket;
  return `Work on ticket ${identifier}: ${title}`;
};

/**
 * Workspace 场景的 Prompt 生成器
 * 返回空字符串，让用户自行输入
 */
export const workspacePromptGenerator: PromptGenerator = (): string => {
  return "";
};

/**
 * 获取场景预设配置
 */
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

/**
 * 合并用户配置和场景预设
 */
export function mergeConfig(
  config: CreatePodFormConfig
): CreatePodFormConfig {
  const preset = getScenarioPreset(config.scenario);
  return {
    ...preset,
    ...config,
    // 如果用户没有提供 promptGenerator，使用预设的
    promptGenerator: config.promptGenerator ?? preset.promptGenerator,
  };
}
