"use client";

import { useState, useCallback } from "react";
import { useTranslations } from "next-intl";
import { TicketStatus } from "@/stores/ticket";
import {
  Circle,
  CircleDot,
  CircleDashed,
  Loader2,
  CheckCircle2,
  XCircle,
  ChevronDown,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

interface StatusSelectProps {
  value: TicketStatus;
  onChange: (status: TicketStatus) => Promise<void>;
  disabled?: boolean;
  showLabel?: boolean;
  size?: "sm" | "md" | "lg";
}

// Linear-style status icons and colors
const statusConfig: Record<TicketStatus, {
  icon: React.ComponentType<{ className?: string }>;
  color: string;
  bgColor: string;
  label: string;
}> = {
  backlog: {
    icon: CircleDashed,
    color: "text-gray-500 dark:text-gray-400",
    bgColor: "bg-gray-100 dark:bg-gray-800",
    label: "Backlog",
  },
  todo: {
    icon: Circle,
    color: "text-blue-500 dark:text-blue-400",
    bgColor: "bg-blue-100 dark:bg-blue-900/30",
    label: "To Do",
  },
  in_progress: {
    icon: CircleDot,
    color: "text-yellow-500 dark:text-yellow-400",
    bgColor: "bg-yellow-100 dark:bg-yellow-900/30",
    label: "In Progress",
  },
  in_review: {
    icon: Loader2,
    color: "text-purple-500 dark:text-purple-400",
    bgColor: "bg-purple-100 dark:bg-purple-900/30",
    label: "In Review",
  },
  done: {
    icon: CheckCircle2,
    color: "text-green-500 dark:text-green-400",
    bgColor: "bg-green-100 dark:bg-green-900/30",
    label: "Done",
  },
  cancelled: {
    icon: XCircle,
    color: "text-red-500 dark:text-red-400",
    bgColor: "bg-red-100 dark:bg-red-900/30",
    label: "Cancelled",
  },
};

const statusOrder: TicketStatus[] = [
  "backlog",
  "todo",
  "in_progress",
  "in_review",
  "done",
  "cancelled",
];

const sizeClasses = {
  sm: "h-6 text-xs",
  md: "h-8 text-sm",
  lg: "h-10 text-base",
};

const iconSizeClasses = {
  sm: "h-3.5 w-3.5",
  md: "h-4 w-4",
  lg: "h-5 w-5",
};

export function StatusSelect({
  value,
  onChange,
  disabled = false,
  showLabel = true,
  size = "md",
}: StatusSelectProps) {
  const t = useTranslations();
  const [isOpen, setIsOpen] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);

  const currentConfig = statusConfig[value] || statusConfig.backlog;
  const StatusIcon = currentConfig.icon;

  const handleSelect = useCallback(async (status: TicketStatus) => {
    if (status === value || disabled) return;

    setIsUpdating(true);
    setIsOpen(false);

    try {
      await onChange(status);
    } catch (error) {
      console.error("Failed to update status:", error);
    } finally {
      setIsUpdating(false);
    }
  }, [value, onChange, disabled]);

  return (
    <DropdownMenu open={isOpen} onOpenChange={setIsOpen}>
      <DropdownMenuTrigger
        disabled={disabled || isUpdating}
        className={cn(
          "inline-flex items-center gap-1.5 px-2 rounded-md transition-all",
          "hover:bg-muted focus:outline-none focus:ring-2 focus:ring-primary/20",
          "disabled:opacity-50 disabled:cursor-not-allowed",
          sizeClasses[size]
        )}
      >
        {isUpdating ? (
          <Loader2 className={cn("animate-spin", iconSizeClasses[size], currentConfig.color)} />
        ) : (
          <StatusIcon className={cn(iconSizeClasses[size], currentConfig.color)} />
        )}
        {showLabel && (
          <span className={cn("font-medium", currentConfig.color)}>
            {t(`tickets.status.${value}`)}
          </span>
        )}
        <ChevronDown className={cn("h-3 w-3 text-muted-foreground", !showLabel && "ml-0.5")} />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-48">
        {statusOrder.map((status) => {
          const config = statusConfig[status];
          const Icon = config.icon;
          const isSelected = status === value;

          return (
            <DropdownMenuItem
              key={status}
              onClick={() => handleSelect(status)}
              className={cn(
                "flex items-center gap-2 cursor-pointer",
                isSelected && "bg-muted"
              )}
            >
              <Icon className={cn("h-4 w-4", config.color)} />
              <span className={isSelected ? "font-medium" : ""}>
                {t(`tickets.status.${status}`)}
              </span>
              {isSelected && (
                <CheckCircle2 className="h-3 w-3 ml-auto text-primary" />
              )}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export default StatusSelect;
