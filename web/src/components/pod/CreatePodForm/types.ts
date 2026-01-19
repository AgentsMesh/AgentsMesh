import { PodData } from "@/lib/api";

/**
 * Pod 创建场景枚举
 */
export type PodCreationScenario = "workspace" | "ticket";

/**
 * Ticket 上下文信息
 */
export interface TicketContext {
  id: number;
  identifier: string;
  title: string;
  description?: string;
}

/**
 * 场景上下文 - 不同场景传入的额外信息
 */
export interface ScenarioContext {
  ticket?: TicketContext;
}

/**
 * Prompt 生成器 - 根据场景上下文生成默认 prompt
 */
export type PromptGenerator = (context: ScenarioContext) => string;

/**
 * CreatePodForm 配置
 */
export interface CreatePodFormConfig {
  /** 创建场景 */
  scenario: PodCreationScenario;

  /** 场景上下文（如 ticket 信息） */
  context?: ScenarioContext;

  /** 自定义 Prompt 生成器 */
  promptGenerator?: PromptGenerator;

  /** Prompt 输入框占位符 */
  promptPlaceholder?: string;

  /** 创建成功回调 */
  onSuccess?: (pod: PodData) => void;

  /** 创建失败回调 */
  onError?: (error: Error) => void;

  /** 取消回调 */
  onCancel?: () => void;
}

/**
 * CreatePodForm Props
 */
export interface CreatePodFormProps {
  /** 表单配置 */
  config: CreatePodFormConfig;

  /** 是否启用（控制数据加载） */
  enabled?: boolean;

  /** 自定义样式类名 */
  className?: string;
}
