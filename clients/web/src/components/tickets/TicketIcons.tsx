"use client";

import React from "react";
import {
  CircleDashed,
  Circle,
  CircleDot,
  Timer,
  CheckCircle2,
  Minus,
  ChevronDown,
  ChevronUp,
  AlertTriangle,
} from "lucide-react";
import type { TicketStatus, TicketPriority } from "@/lib/api/ticketTypes";
import { cn } from "@/lib/utils";

type IconSize = "xs" | "sm" | "md" | "lg";

const sizeClasses: Record<IconSize, string> = {
  xs: "h-3 w-3",
  sm: "h-3.5 w-3.5",
  md: "h-4 w-4",
  lg: "h-5 w-5",
};

interface StatusIconProps {
  status: TicketStatus;
  size?: IconSize;
  className?: string;
}

const statusIconMap: Record<TicketStatus, React.ComponentType<{ className?: string }>> = {
  backlog: CircleDashed,
  todo: Circle,
  in_progress: Timer,
  in_review: CircleDot,
  done: CheckCircle2,
};

const statusColorMap: Record<TicketStatus, string> = {
  backlog: "text-gray-500 dark:text-gray-400",
  todo: "text-blue-500 dark:text-blue-400",
  in_progress: "text-yellow-500 dark:text-yellow-400",
  in_review: "text-purple-500 dark:text-purple-400",
  done: "text-green-500 dark:text-green-400",
};

export function StatusIcon({ status, size = "sm", className }: StatusIconProps) {
  const IconComponent = statusIconMap[status] || CircleDashed;
  const colorClass = statusColorMap[status] || statusColorMap.backlog;

  return (
    <IconComponent
      className={cn(
        sizeClasses[size],
        colorClass,
        className
      )}
    />
  );
}

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

type TranslateFn = (key: string) => string;

const statusFallbackLabels: Record<TicketStatus, string> = {
  backlog: "Backlog",
  todo: "To Do",
  in_progress: "In Progress",
  in_review: "In Review",
  done: "Done",
};

const statusBgColorMap: Record<TicketStatus, string> = {
  backlog: "bg-gray-100 dark:bg-gray-800",
  todo: "bg-blue-100 dark:bg-blue-900/30",
  in_progress: "bg-yellow-100 dark:bg-yellow-900/30",
  in_review: "bg-purple-100 dark:bg-purple-900/30",
  done: "bg-green-100 dark:bg-green-900/30",
};

export function getStatusDisplayInfo(status: TicketStatus, sizeOrT?: IconSize | TranslateFn, maybeSize?: IconSize): StatusInfo {
  let size: IconSize = "sm";
  let t: TranslateFn | undefined;

  if (typeof sizeOrT === "function") {
    t = sizeOrT;
    size = maybeSize || "sm";
  } else if (typeof sizeOrT === "string") {
    size = sizeOrT;
  }

  const label = t ? t(`tickets.status.${status}`) : (statusFallbackLabels[status] || status);

  return {
    label,
    color: statusColorMap[status] || statusColorMap.backlog,
    bgColor: statusBgColorMap[status] || statusBgColorMap.backlog,
    icon: <StatusIcon status={status} size={size} />,
  };
}

const priorityFallbackLabels: Record<TicketPriority, string> = {
  none: "None",
  low: "Low",
  medium: "Medium",
  high: "High",
  urgent: "Urgent",
};

export function getPriorityDisplayInfo(priority: TicketPriority, sizeOrT?: IconSize | TranslateFn, maybeSize?: IconSize): PriorityInfo {
  let size: IconSize = "sm";
  let t: TranslateFn | undefined;

  if (typeof sizeOrT === "function") {
    t = sizeOrT;
    size = maybeSize || "sm";
  } else if (typeof sizeOrT === "string") {
    size = sizeOrT;
  }

  const label = t ? t(`tickets.priority.${priority}`) : (priorityFallbackLabels[priority] || priority);

  return {
    label,
    color: priorityColorMap[priority] || priorityColorMap.none,
    icon: <PriorityIcon priority={priority} size={size} />,
  };
}

export { statusColorMap, priorityColorMap };
