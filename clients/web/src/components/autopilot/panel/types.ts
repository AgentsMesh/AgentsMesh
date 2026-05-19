import * as React from "react";

export type NormalizedDecisionType = "continue" | "completed" | "need_help" | "give_up";

export function normalizeDecisionType(backendType: string): NormalizedDecisionType {
  const mapping: Record<string, NormalizedDecisionType> = {
    CONTINUE: "continue",
    TASK_COMPLETED: "completed",
    NEED_HUMAN_HELP: "need_help",
    GIVE_UP: "give_up",
    continue: "continue",
    completed: "completed",
    need_help: "need_help",
    give_up: "give_up",
  };
  return mapping[backendType] || "continue";
}

export interface DecisionTypeConfig {
  label: string;
  bgColor: string;
  textColor: string;
  icon: React.ReactNode;
}

export interface ActionTypeConfig {
  label: string;
  icon: React.ReactNode;
}

export interface IterationPhaseConfig {
  label: string;
  color: string;
  icon: React.ReactNode;
}
