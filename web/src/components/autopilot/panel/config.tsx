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
  Pause,
  Square,
  Hand,
  Loader2,
} from "lucide-react";
import type {
  NormalizedDecisionType,
  DecisionTypeConfig,
  ActionTypeConfig,
  IterationPhaseConfig,
} from "./types";
import type { AutopilotController } from "@/stores/autopilot";

// Decision type configuration
export const decisionConfig: Record<NormalizedDecisionType, DecisionTypeConfig> = {
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

// Action type configuration
export const actionConfig: Record<string, ActionTypeConfig> = {
  observe: { label: "Observing", icon: <Eye className="h-3 w-3" /> },
  send_input: { label: "Sending Input", icon: <Send className="h-3 w-3" /> },
  wait: { label: "Waiting", icon: <Clock className="h-3 w-3" /> },
  none: { label: "No Action", icon: <MessageSquare className="h-3 w-3" /> },
};

// Iteration phase display configuration
export const iterationPhaseConfig: Record<string, IterationPhaseConfig> = {
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

// Phase display configuration with colors
export const phaseConfig: Record<
  AutopilotController["phase"],
  { label: string; color: string; icon: React.ReactNode }
> = {
  initializing: {
    label: "Initializing",
    color: "text-blue-500",
    icon: <Loader2 className="h-3.5 w-3.5 animate-spin" />,
  },
  running: {
    label: "Running",
    color: "text-green-500",
    icon: <Play className="h-3.5 w-3.5" />,
  },
  paused: {
    label: "Paused",
    color: "text-yellow-500",
    icon: <Pause className="h-3.5 w-3.5" />,
  },
  user_takeover: {
    label: "User Control",
    color: "text-purple-500",
    icon: <Hand className="h-3.5 w-3.5" />,
  },
  waiting_approval: {
    label: "Waiting Approval",
    color: "text-orange-500",
    icon: <AlertTriangle className="h-3.5 w-3.5" />,
  },
  completed: {
    label: "Completed",
    color: "text-green-600",
    icon: <CheckCircle className="h-3.5 w-3.5" />,
  },
  failed: {
    label: "Failed",
    color: "text-red-500",
    icon: <XCircle className="h-3.5 w-3.5" />,
  },
  stopped: {
    label: "Stopped",
    color: "text-gray-500",
    icon: <Square className="h-3.5 w-3.5" />,
  },
  max_iterations: {
    label: "Max Iterations",
    color: "text-orange-600",
    icon: <Clock className="h-3.5 w-3.5" />,
  },
};
