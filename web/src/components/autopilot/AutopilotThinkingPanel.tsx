"use client";

import * as React from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import { useAutopilotStore } from "@/stores/autopilot";
import {
  Brain,
  CheckCircle,
  AlertTriangle,
  XCircle,
  ArrowRight,
  Clock,
  MessageSquare,
  Eye,
  Send,
} from "lucide-react";

// Stable empty array — avoids new reference on every selector call
const EMPTY_HISTORY: never[] = [];

interface AutopilotThinkingPanelProps {
  autopilotControllerKey: string;
  className?: string;
  showHistory?: boolean;
}

// Normalized decision types (lowercase only)
type NormalizedDecisionType = "continue" | "completed" | "need_help" | "give_up";

// Map backend decision types to frontend keys
// Backend uses: CONTINUE, TASK_COMPLETED, NEED_HUMAN_HELP, GIVE_UP
// Frontend expects: continue, completed, need_help, give_up
function normalizeDecisionType(backendType: string): NormalizedDecisionType {
  const mapping: Record<string, NormalizedDecisionType> = {
    "CONTINUE": "continue",
    "TASK_COMPLETED": "completed",
    "NEED_HUMAN_HELP": "need_help",
    "GIVE_UP": "give_up",
    "continue": "continue",
    "completed": "completed",
    "need_help": "need_help",
    "give_up": "give_up",
  };
  return mapping[backendType] || "continue";
}

// Decision type configuration
const decisionConfig: Record<
  NormalizedDecisionType,
  { label: string; color: string; icon: React.ReactNode }
> = {
  continue: {
    label: "Continue",
    color: "bg-blue-500",
    icon: <ArrowRight className="h-3 w-3" />,
  },
  completed: {
    label: "Completed",
    color: "bg-green-500",
    icon: <CheckCircle className="h-3 w-3" />,
  },
  need_help: {
    label: "Need Help",
    color: "bg-orange-500",
    icon: <AlertTriangle className="h-3 w-3" />,
  },
  give_up: {
    label: "Give Up",
    color: "bg-red-500",
    icon: <XCircle className="h-3 w-3" />,
  },
};

// Action type configuration
const actionConfig: Record<string, { label: string; icon: React.ReactNode }> = {
  observe: { label: "Observing", icon: <Eye className="h-3 w-3" /> },
  send_input: { label: "Sending Input", icon: <Send className="h-3 w-3" /> },
  wait: { label: "Waiting", icon: <Clock className="h-3 w-3" /> },
  none: { label: "No Action", icon: <MessageSquare className="h-3 w-3" /> },
};

export function AutopilotThinkingPanel({
  autopilotControllerKey,
  className,
  showHistory = false,
}: AutopilotThinkingPanelProps) {
  // Reactive selectors — re-render when this controller's thinking/history changes
  const thinking = useAutopilotStore((s) => s.thinking[autopilotControllerKey] ?? null);
  const history = useAutopilotStore((s) =>
    showHistory ? s.thinkingHistory[autopilotControllerKey] ?? EMPTY_HISTORY : EMPTY_HISTORY
  );

  if (!thinking) {
    return (
      <div className={cn("rounded-lg border bg-muted/50 p-4", className)}>
        <div className="flex items-center justify-center text-muted-foreground gap-2">
          <Brain className="h-4 w-4" />
          <span className="text-sm">Waiting for Control Agent...</span>
        </div>
      </div>
    );
  }

  const normalizedDecisionType = normalizeDecisionType(thinking.decision_type);
  const decisionInfo = decisionConfig[normalizedDecisionType];
  const actionInfo = thinking.action ? actionConfig[thinking.action.type] : null;

  return (
    <div className={cn("space-y-3", className)}>
      {/* Current Thinking */}
      <div className="rounded-lg border bg-card p-4 shadow-sm">
        {/* Header */}
        <div className="flex items-center gap-2 text-sm font-medium mb-3">
          <Brain className="h-4 w-4 text-primary" />
          <span>Control Agent Thinking</span>
          <Badge
            variant="outline"
            className={cn("ml-auto", decisionInfo.color, "text-white")}
          >
            {decisionInfo.icon}
            <span className="ml-1">{decisionInfo.label}</span>
          </Badge>
        </div>

        {/* Content */}
        <div className="space-y-3">
          {/* Reasoning */}
          <div className="text-sm">
            <div className="text-muted-foreground mb-1 text-xs">Reasoning</div>
            <p className="text-foreground">{thinking.reasoning}</p>
          </div>

          {/* Confidence */}
          <div className="flex items-center gap-2 text-xs">
            <span className="text-muted-foreground">Confidence:</span>
            <div className="flex-1 max-w-[100px]">
              <Progress value={thinking.confidence * 100} className="h-1.5" />
            </div>
            <span className="font-medium">{Math.round(thinking.confidence * 100)}%</span>
          </div>

          {/* Action */}
          {actionInfo && thinking.action && (
            <div className="rounded-md bg-muted/50 p-2 text-sm">
              <div className="flex items-center gap-2 text-muted-foreground mb-1">
                {actionInfo.icon}
                <span className="text-xs">{actionInfo.label}</span>
              </div>
              {thinking.action.content && (
                <p className="text-xs font-mono bg-background/50 p-1 rounded">
                  {thinking.action.content}
                </p>
              )}
              {thinking.action.reason && (
                <p className="text-xs text-muted-foreground mt-1">
                  {thinking.action.reason}
                </p>
              )}
            </div>
          )}

          {/* Progress */}
          {thinking.progress && (
            <div className="space-y-2">
              <div className="flex items-center justify-between text-xs">
                <span className="text-muted-foreground">{thinking.progress.summary}</span>
                {thinking.progress.percent > 0 && (
                  <span className="font-medium">{thinking.progress.percent}%</span>
                )}
              </div>
              {thinking.progress.percent > 0 && (
                <Progress value={thinking.progress.percent} className="h-1.5" />
              )}

              {/* Completed Steps */}
              {thinking.progress.completed_steps.length > 0 && (
                <div className="text-xs space-y-1">
                  <span className="text-muted-foreground">Completed:</span>
                  <ul className="list-disc list-inside text-green-600">
                    {thinking.progress.completed_steps.map((step, i) => (
                      <li key={i}>{step}</li>
                    ))}
                  </ul>
                </div>
              )}

              {/* Remaining Steps */}
              {thinking.progress.remaining_steps.length > 0 && (
                <div className="text-xs space-y-1">
                  <span className="text-muted-foreground">Remaining:</span>
                  <ul className="list-disc list-inside text-muted-foreground">
                    {thinking.progress.remaining_steps.map((step, i) => (
                      <li key={i}>{step}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          )}

          {/* Help Request */}
          {thinking.help_request && (
            <div className="rounded-md bg-orange-500/10 border border-orange-500/30 p-2 text-sm">
              <div className="flex items-center gap-2 text-orange-500 mb-1">
                <AlertTriangle className="h-3 w-3" />
                <span className="font-medium text-xs">Help Needed</span>
              </div>
              <p className="text-xs">{thinking.help_request.reason}</p>
              {thinking.help_request.context && (
                <p className="text-xs text-muted-foreground mt-1">
                  Context: {thinking.help_request.context}
                </p>
              )}
              {thinking.help_request.terminal_excerpt && (
                <pre className="text-xs font-mono bg-background/50 p-1 rounded mt-1 overflow-x-auto">
                  {thinking.help_request.terminal_excerpt}
                </pre>
              )}
            </div>
          )}

          {/* Iteration */}
          <div className="text-xs text-muted-foreground text-right">
            Iteration #{thinking.iteration}
          </div>
        </div>
      </div>

      {/* History (if enabled) */}
      {showHistory && history.length > 1 && (
        <div className="rounded-lg border bg-card p-4 shadow-sm">
          <div className="text-sm font-medium mb-3">Decision History</div>
          <div className="space-y-2 max-h-[200px] overflow-y-auto">
            {history.slice(0, -1).reverse().map((item, index) => {
              const info = decisionConfig[normalizeDecisionType(item.decision_type)];
              return (
                <div
                  key={index}
                  className="flex items-start gap-2 text-xs p-2 rounded bg-muted/50"
                >
                  <Badge
                    variant="outline"
                    className={cn("shrink-0", info.color, "text-white")}
                  >
                    {info.icon}
                  </Badge>
                  <div className="flex-1 min-w-0">
                    <p className="truncate">{item.reasoning}</p>
                    <span className="text-muted-foreground">
                      #{item.iteration}
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}

export default AutopilotThinkingPanel;
