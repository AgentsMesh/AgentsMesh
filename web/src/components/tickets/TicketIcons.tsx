"use client";

import React from "react";
import {
  // Status icons
  CircleDashed,
  Circle,
  CircleDot,
  Loader2,
  CheckCircle2,
  XCircle,
  // Priority icons
  Minus,
  ChevronDown,
  ChevronUp,
  ChevronsUp,
  AlertTriangle,
  // Type icons
  CheckSquare,
  Bug,
  Sparkles,
  TrendingUp,
  Zap,
  CircleDot as SubtaskIcon,
  BookOpen,
} from "lucide-react";
import { TicketStatus, TicketPriority, TicketType } from "@/lib/api/client";
import { cn } from "@/lib/utils";

// Icon size presets
type IconSize = "xs" | "sm" | "md" | "lg";

const sizeClasses: Record<IconSize, string> = {
  xs: "h-3 w-3",
  sm: "h-3.5 w-3.5",
  md: "h-4 w-4",
  lg: "h-5 w-5",
};

// ============================================================================
// Status Icons
// ============================================================================

interface StatusIconProps {
  status: TicketStatus;
  size?: IconSize;
  className?: string;
}

const statusIconMap: Record<TicketStatus, React.ComponentType<{ className?: string }>> = {
  backlog: CircleDashed,
  todo: Circle,
  in_progress: Loader2,
  in_review: CircleDot,
  done: CheckCircle2,
  cancelled: XCircle,
};

const statusColorMap: Record<TicketStatus, string> = {
  backlog: "text-gray-500 dark:text-gray-400",
  todo: "text-blue-500 dark:text-blue-400",
  in_progress: "text-yellow-500 dark:text-yellow-400",
  in_review: "text-purple-500 dark:text-purple-400",
  done: "text-green-500 dark:text-green-400",
  cancelled: "text-red-500 dark:text-red-400",
};

export function StatusIcon({ status, size = "sm", className }: StatusIconProps) {
  const IconComponent = statusIconMap[status] || CircleDashed;
  const colorClass = statusColorMap[status] || statusColorMap.backlog;
  const isAnimated = status === "in_progress";

  return (
    <IconComponent
      className={cn(
        sizeClasses[size],
        colorClass,
        isAnimated && "animate-spin",
        className
      )}
    />
  );
}

// ============================================================================
// Priority Icons
// ============================================================================

interface PriorityIconProps {
  priority: TicketPriority;
  size?: IconSize;
  className?: string;
}

const priorityIconMap: Record<TicketPriority, React.ComponentType<{ className?: string }>> = {
  none: Minus,
  low: ChevronDown,
  medium: Minus,
  high: ChevronUp,
  urgent: AlertTriangle,
};

const priorityColorMap: Record<TicketPriority, string> = {
  none: "text-gray-400 dark:text-gray-500",
  low: "text-blue-500 dark:text-blue-400",
  medium: "text-yellow-500 dark:text-yellow-400",
  high: "text-orange-500 dark:text-orange-400",
  urgent: "text-red-500 dark:text-red-400",
};

export function PriorityIcon({ priority, size = "sm", className }: PriorityIconProps) {
  const IconComponent = priorityIconMap[priority] || Minus;
  const colorClass = priorityColorMap[priority] || priorityColorMap.none;

  return (
    <IconComponent
      className={cn(sizeClasses[size], colorClass, className)}
    />
  );
}

// ============================================================================
// Type Icons
// ============================================================================

interface TypeIconProps {
  type: TicketType;
  size?: IconSize;
  className?: string;
}

const typeIconMap: Record<TicketType, React.ComponentType<{ className?: string }>> = {
  task: CheckSquare,
  bug: Bug,
  feature: Sparkles,
  improvement: TrendingUp,
  epic: Zap,
  subtask: SubtaskIcon,
  story: BookOpen,
};

const typeColorMap: Record<TicketType, string> = {
  task: "text-blue-500 dark:text-blue-400",
  bug: "text-red-500 dark:text-red-400",
  feature: "text-green-500 dark:text-green-400",
  improvement: "text-cyan-500 dark:text-cyan-400",
  epic: "text-purple-500 dark:text-purple-400",
  subtask: "text-gray-500 dark:text-gray-400",
  story: "text-teal-500 dark:text-teal-400",
};

export function TypeIcon({ type, size = "sm", className }: TypeIconProps) {
  const IconComponent = typeIconMap[type] || CheckSquare;
  const colorClass = typeColorMap[type] || typeColorMap.task;

  return (
    <IconComponent
      className={cn(sizeClasses[size], colorClass, className)}
    />
  );
}

// ============================================================================
// Helper functions for getting display info with React nodes
// ============================================================================

export interface StatusInfo {
  label: string;
  color: string;
  bgColor: string;
  icon: React.ReactNode;
}

export interface PriorityInfo {
  label: string;
  color: string;
  icon: React.ReactNode;
}

export interface TypeInfo {
  label: string;
  color: string;
  bgColor: string;
  icon: React.ReactNode;
}

export function getStatusDisplayInfo(status: TicketStatus, size: IconSize = "sm"): StatusInfo {
  const bgColorMap: Record<TicketStatus, string> = {
    backlog: "bg-gray-100 dark:bg-gray-800",
    todo: "bg-blue-100 dark:bg-blue-900/30",
    in_progress: "bg-yellow-100 dark:bg-yellow-900/30",
    in_review: "bg-purple-100 dark:bg-purple-900/30",
    done: "bg-green-100 dark:bg-green-900/30",
    cancelled: "bg-red-100 dark:bg-red-900/30",
  };

  const labelMap: Record<TicketStatus, string> = {
    backlog: "Backlog",
    todo: "To Do",
    in_progress: "In Progress",
    in_review: "In Review",
    done: "Done",
    cancelled: "Cancelled",
  };

  return {
    label: labelMap[status] || status,
    color: statusColorMap[status] || statusColorMap.backlog,
    bgColor: bgColorMap[status] || bgColorMap.backlog,
    icon: <StatusIcon status={status} size={size} />,
  };
}

export function getPriorityDisplayInfo(priority: TicketPriority, size: IconSize = "sm"): PriorityInfo {
  const labelMap: Record<TicketPriority, string> = {
    none: "None",
    low: "Low",
    medium: "Medium",
    high: "High",
    urgent: "Urgent",
  };

  return {
    label: labelMap[priority] || priority,
    color: priorityColorMap[priority] || priorityColorMap.none,
    icon: <PriorityIcon priority={priority} size={size} />,
  };
}

export function getTypeDisplayInfo(type: TicketType, size: IconSize = "sm"): TypeInfo {
  const labelMap: Record<TicketType, string> = {
    task: "Task",
    bug: "Bug",
    feature: "Feature",
    improvement: "Improvement",
    epic: "Epic",
    subtask: "Subtask",
    story: "Story",
  };

  const bgColorMap: Record<TicketType, string> = {
    task: "bg-blue-100 dark:bg-blue-900/30",
    bug: "bg-red-100 dark:bg-red-900/30",
    feature: "bg-purple-100 dark:bg-purple-900/30",
    improvement: "bg-green-100 dark:bg-green-900/30",
    epic: "bg-indigo-100 dark:bg-indigo-900/30",
    subtask: "bg-gray-100 dark:bg-gray-800",
    story: "bg-cyan-100 dark:bg-cyan-900/30",
  };

  return {
    label: labelMap[type] || type,
    color: typeColorMap[type] || typeColorMap.task,
    bgColor: bgColorMap[type] || bgColorMap.task,
    icon: <TypeIcon type={type} size={size} />,
  };
}

// Export color maps for external use
export { statusColorMap, priorityColorMap, typeColorMap };
