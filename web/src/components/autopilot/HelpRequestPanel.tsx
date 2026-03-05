"use client";

import * as React from "react";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useAutopilotStore } from "@/stores/autopilot";
import {
  AlertTriangle,
  CheckCircle,
  XCircle,
  MessageSquare,
  Terminal,
  HelpCircle,
} from "lucide-react";

interface HelpRequestPanelProps {
  autopilotControllerKey: string;
  className?: string;
  onApprove?: (continueExecution: boolean, additionalIterations?: number) => void;
  onCustomResponse?: () => void;
}

/**
 * HelpRequestPanel displays detailed help request information from the Control Agent
 * when it encounters a situation requiring human intervention.
 *
 * Features:
 * - Clear reason for help request
 * - Context about what was being attempted
 * - Terminal excerpt showing the relevant output
 * - Suggested actions as clickable buttons
 */
export function HelpRequestPanel({
  autopilotControllerKey,
  className,
  onApprove,
  onCustomResponse,
}: HelpRequestPanelProps) {
  const approveAutopilotController = useAutopilotStore((s) => s.approveAutopilotController);
  // Reactive thinking selector — re-renders when this controller's thinking changes
  const thinking = useAutopilotStore((s) => s.thinking[autopilotControllerKey] ?? null);

  // Only show when there's a help request
  if (!thinking?.help_request) {
    return null;
  }

  const helpRequest = thinking.help_request;

  const handleSuggestionClick = (suggestion: { action: string; label: string }) => {
    switch (suggestion.action) {
      case "approve":
        // Approve and continue execution
        if (onApprove) {
          onApprove(true);
        } else {
          approveAutopilotController(autopilotControllerKey, { continue_execution: true });
        }
        break;
      case "skip":
        // Approve but skip the current operation
        if (onApprove) {
          onApprove(true, 5); // Add extra iterations for recovery
        } else {
          approveAutopilotController(autopilotControllerKey, {
            continue_execution: true,
            additional_iterations: 5,
          });
        }
        break;
      case "stop":
        // Stop execution
        if (onApprove) {
          onApprove(false);
        } else {
          approveAutopilotController(autopilotControllerKey, { continue_execution: false });
        }
        break;
      case "custom":
        // Open custom response dialog
        onCustomResponse?.();
        break;
      default:
        // Default to approve and continue
        if (onApprove) {
          onApprove(true);
        }
    }
  };

  return (
    <div className={cn("rounded-lg border border-orange-500/50 bg-orange-500/5 p-4 shadow-sm", className)}>
      {/* Header */}
      <div className="flex items-center gap-2 mb-3">
        <div className="flex items-center justify-center w-8 h-8 rounded-full bg-orange-500/20">
          <AlertTriangle className="h-4 w-4 text-orange-500" />
        </div>
        <div>
          <h3 className="font-semibold text-sm text-orange-500">Help Requested</h3>
          <p className="text-xs text-muted-foreground">
            Control Agent needs human intervention
          </p>
        </div>
        <Badge variant="outline" className="ml-auto bg-orange-500 text-white">
          Iteration #{thinking.iteration}
        </Badge>
      </div>

      {/* Reason */}
      <div className="mb-3">
        <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-1">
          <HelpCircle className="h-3 w-3" />
          <span>Reason</span>
        </div>
        <p className="text-sm">{helpRequest.reason}</p>
      </div>

      {/* Context */}
      {helpRequest.context && (
        <div className="mb-3">
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-1">
            <MessageSquare className="h-3 w-3" />
            <span>Context</span>
          </div>
          <p className="text-sm text-muted-foreground">{helpRequest.context}</p>
        </div>
      )}

      {/* Terminal Excerpt */}
      {helpRequest.terminal_excerpt && (
        <div className="mb-4">
          <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-1">
            <Terminal className="h-3 w-3" />
            <span>Terminal Output</span>
          </div>
          <div className="rounded bg-zinc-900 p-3 overflow-x-auto">
            <pre className="text-xs text-zinc-100 font-mono whitespace-pre-wrap break-all">
              {helpRequest.terminal_excerpt}
            </pre>
          </div>
        </div>
      )}

      {/* Suggestions */}
      {helpRequest.suggestions && helpRequest.suggestions.length > 0 && (
        <div className="border-t border-orange-500/20 pt-3">
          <div className="text-xs text-muted-foreground mb-2">Suggested Actions</div>
          <div className="flex flex-wrap gap-2">
            {helpRequest.suggestions.map((suggestion, index) => (
              <Button
                key={index}
                size="sm"
                variant={suggestion.action === "approve" ? "default" : "outline"}
                onClick={() => handleSuggestionClick(suggestion)}
                className={cn(
                  suggestion.action === "approve" && "bg-green-600 hover:bg-green-700",
                  suggestion.action === "stop" && "text-red-500 border-red-500/50 hover:bg-red-500/10"
                )}
              >
                {suggestion.action === "approve" && <CheckCircle className="h-3 w-3 mr-1" />}
                {suggestion.action === "stop" && <XCircle className="h-3 w-3 mr-1" />}
                {suggestion.label}
              </Button>
            ))}
          </div>
        </div>
      )}

      {/* Default Actions (if no suggestions provided) */}
      {(!helpRequest.suggestions || helpRequest.suggestions.length === 0) && (
        <div className="border-t border-orange-500/20 pt-3">
          <div className="text-xs text-muted-foreground mb-2">Actions</div>
          <div className="flex flex-wrap gap-2">
            <Button
              size="sm"
              variant="default"
              className="bg-green-600 hover:bg-green-700"
              onClick={() => handleSuggestionClick({ action: "approve", label: "Continue" })}
            >
              <CheckCircle className="h-3 w-3 mr-1" />
              Approve & Continue
            </Button>
            <Button
              size="sm"
              variant="outline"
              onClick={() => handleSuggestionClick({ action: "skip", label: "Skip" })}
            >
              Skip This Step
            </Button>
            <Button
              size="sm"
              variant="outline"
              className="text-red-500 border-red-500/50 hover:bg-red-500/10"
              onClick={() => handleSuggestionClick({ action: "stop", label: "Stop" })}
            >
              <XCircle className="h-3 w-3 mr-1" />
              Stop Autopilot
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

export default HelpRequestPanel;
