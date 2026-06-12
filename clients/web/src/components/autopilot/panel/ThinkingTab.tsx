"use client";

import * as React from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Brain, AlertTriangle } from "lucide-react";
import type { AutopilotThinking } from "@/stores/autopilot";
import { normalizeDecisionType } from "./types";
import { decisionConfig, actionConfig } from "./config";

interface ThinkingTabProps {
  thinking: AutopilotThinking | null;
}

export function ThinkingTab({ thinking }: ThinkingTabProps) {
  if (!thinking) {
    return (
      <div className="flex flex-col items-center justify-center py-6 text-muted-foreground">
        <Brain className="h-6 w-6 mb-2 opacity-50" />
        <span className="text-xs">Waiting for Control Agent...</span>
      </div>
    );
  }

  const normalizedDecisionType = normalizeDecisionType(thinking.decision_type);
  const decisionInfo = decisionConfig[normalizedDecisionType];
  const actionInfo = thinking.action ? actionConfig[thinking.action.type] : null;

  return (
    <div className="space-y-3 p-3">
      {/* Decision Type Badge */}
      <div className="flex items-center gap-2">
        <Badge
          variant="outline"
          className={cn("flex items-center gap-1", decisionInfo.bgColor, "text-white")}
        >
          {decisionInfo.icon}
          <span>{decisionInfo.label}</span>
        </Badge>
        <span className="text-xs text-muted-foreground">Iteration #{thinking.iteration}</span>
        {(thinking.confidence ?? 0) > 0 && (
          <span className="text-xs text-muted-foreground ml-auto">
            Confidence: {Math.round((thinking.confidence ?? 0) * 100)}%
          </span>
        )}
      </div>

      {/* Reasoning */}
      <div>
        <div className="text-xs text-muted-foreground mb-1">Reasoning</div>
        <p className="text-sm leading-relaxed">{thinking.reasoning}</p>
      </div>

      {/* Action */}
      {actionInfo && thinking.action && (
        <div className="rounded-md bg-muted/50 p-2">
          <div className="flex items-center gap-2 text-muted-foreground mb-1">
            {actionInfo.icon}
            <span className="text-xs font-medium">{actionInfo.label}</span>
          </div>
          {thinking.action.content && (
            <p className="text-xs font-mono bg-background/50 p-1.5 rounded break-all">
              {thinking.action.content}
            </p>
          )}
          {thinking.action.reason && (
            <p className="text-xs text-muted-foreground mt-1">{thinking.action.reason}</p>
          )}
        </div>
      )}

      {/* Help Request */}
      {thinking.help_request && (
        <div className="rounded-md bg-orange-500/10 border border-orange-500/30 p-2">
          <div className="flex items-center gap-2 text-orange-500 mb-1">
            <AlertTriangle className="h-3 w-3" />
            <span className="font-medium text-xs">Help Needed</span>
          </div>
          <p className="text-xs mb-1">{thinking.help_request.reason}</p>
          {thinking.help_request.context && (
            <p className="text-xs text-muted-foreground">
              Context: {thinking.help_request.context}
            </p>
          )}
          {thinking.help_request.terminal_excerpt && (
            <pre className="text-xs font-mono bg-zinc-900 text-zinc-100 p-2 rounded mt-2 overflow-x-auto whitespace-pre-wrap break-all max-h-20">
              {thinking.help_request.terminal_excerpt}
            </pre>
          )}
        </div>
      )}
    </div>
  );
}

export default ThinkingTab;
