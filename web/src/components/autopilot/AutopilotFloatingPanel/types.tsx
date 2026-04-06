import * as React from "react";
import {
  ArrowRight,
  CheckCircle,
  AlertTriangle,
  XCircle,
  Clock,
  Eye,
  Send,
  MessageSquare,
  Play,
  Loader2,
} from "lucide-react";
import type { AutopilotController } from "@/stores/autopilot";

/**
 * Props for AutopilotFloatingPanel component
 */
export interface AutopilotFloatingPanelProps {
  autopilotController: AutopilotController;
  className?: string;
  onClose?: () => void;
}

/**
 * Normalized decision types (lowercase only)
 */
export type NormalizedDecisionType = "continue" | "completed" | "need_help" | "give_up";

/**
 * Decision type configuration
 */
export const decisionConfig: Record<
  NormalizedDecisionType,
  { label: string; bgColor: string; textColor: string; icon: React.ReactNode }
> = {
  continue: {
    label: "Continue",
    bgColor: "bg-blue-500",
    textColor: "text-blue-500",
    icon: <ArrowRight className="h-3 w-3" />,
  },
  completed: {
    label: "Completed",
    bgColor: "bg-green-500",
    textColor: "text-green-500",
    icon: <CheckCircle className="h-3 w-3" />,
  },
  need_help: {
    label: "Need Help",
    bgColor: "bg-orange-500",
    textColor: "text-orange-500",
    icon: <AlertTriangle className="h-3 w-3" />,
  },
  give_up: {
    label: "Give Up",
    bgColor: "bg-red-500",
    textColor: "text-red-500",
    icon: <XCircle className="h-3 w-3" />,
  },
};

/**
 * Action type configuration
 */
export const actionConfig: Record<string, { label: string; icon: React.ReactNode }> = {
  observe: { label: "Observing", icon: <Eye className="h-3 w-3" /> },
  send_input: { label: "Sending Input", icon: <Send className="h-3 w-3" /> },
  wait: { label: "Waiting", icon: <Clock className="h-3 w-3" /> },
  none: { label: "No Action", icon: <MessageSquare className="h-3 w-3" /> },
};

/**
 * Iteration phase display configuration
 */
export const iterationPhaseConfig: Record<
  string,
  { label: string; color: string; icon: React.ReactNode }
> = {
  prompt: {
    label: "Initial",
    color: "bg-blue-500",
    icon: <Send className="h-3 w-3" />,
  },
  started: {
    label: "Started",
    color: "bg-blue-400",
    icon: <Play className="h-3 w-3" />,
  },
  control_running: {
    label: "Running",
    color: "bg-yellow-500",
    icon: <Loader2 className="h-3 w-3 animate-spin" />,
  },
  action_sent: {
    label: "Sent",
    color: "bg-green-500",
    icon: <Send className="h-3 w-3" />,
  },
  completed: {
    label: "Done",
    color: "bg-green-600",
    icon: <CheckCircle className="h-3 w-3" />,
  },
  error: {
    label: "Error",
    color: "bg-red-500",
    icon: <XCircle className="h-3 w-3" />,
  },
};

/**
 * Map backend decision types to frontend keys
 * Backend uses: CONTINUE, TASK_COMPLETED, NEED_HUMAN_HELP, GIVE_UP
 * Frontend expects: continue, completed, need_help, give_up
 */
export function normalizeDecisionType(backendType: string): NormalizedDecisionType {
  const mapping: Record<string, NormalizedDecisionType> = {
    "CONTINUE": "continue",
    "TASK_COMPLETED": "completed",
    "NEED_HUMAN_HELP": "need_help",
    "GIVE_UP": "give_up",
    // Also support lowercase (in case backend is updated)
    "continue": "continue",
    "completed": "completed",
    "need_help": "need_help",
    "give_up": "give_up",
  };
  return mapping[backendType] || "continue";
}
