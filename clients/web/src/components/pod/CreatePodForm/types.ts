import type { PodData } from "@/lib/api/pod";

export type PodCreationScenario = "workspace" | "ticket";

export interface TicketContext {
  id: number;
  slug: string;
  title: string;
  description?: string;
  repositoryId?: number;
}

export interface ScenarioContext {
  ticket?: TicketContext;
}

export type PromptGenerator = (context: ScenarioContext) => string;

export interface CreatePodFormConfig {
  scenario: PodCreationScenario;
  context?: ScenarioContext;
  promptGenerator?: PromptGenerator;
  promptPlaceholder?: string;
  onSuccess?: (pod: PodData) => void;
  onError?: (error: Error) => void;
  onCancel?: () => void;
}

export interface CreatePodFormProps {
  config: CreatePodFormConfig;
  enabled?: boolean;
  className?: string;
}
