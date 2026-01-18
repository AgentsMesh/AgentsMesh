"use client";

import { useState, useCallback } from "react";
import { useTranslations } from "@/lib/i18n/client";
import { TicketPriority } from "@/stores/ticket";
import {
  Minus,
  ChevronDown as ChevronDownIcon,
  ChevronUp,
  ChevronsUp,
  AlertTriangle,
  Loader2,
  Check,
} from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";

interface PrioritySelectProps {
  value: TicketPriority;
  onChange: (priority: TicketPriority) => Promise<void>;
  disabled?: boolean;
  showLabel?: boolean;
  size?: "sm" | "md" | "lg";
}

// Linear-style priority icons and colors
const priorityConfig: Record<TicketPriority, {
  icon: React.ComponentType<{ className?: string }>;
  color: string;
  label: string;
  shortLabel: string;
}> = {
  none: {
    icon: Minus,
    color: "text-gray-400 dark:text-gray-500",
    label: "No Priority",
    shortLabel: "None",
  },
  low: {
    icon: ChevronDownIcon,
    color: "text-green-500 dark:text-green-400",
    label: "Low Priority",
    shortLabel: "Low",
  },
  medium: {
    icon: Minus,
    color: "text-yellow-500 dark:text-yellow-400",
    label: "Medium Priority",
    shortLabel: "Medium",
  },
  high: {
    icon: ChevronUp,
    color: "text-orange-500 dark:text-orange-400",
    label: "High Priority",
    shortLabel: "High",
  },
  urgent: {
    icon: AlertTriangle,
    color: "text-red-500 dark:text-red-400",
    label: "Urgent",
    shortLabel: "Urgent",
  },
};

const priorityOrder: TicketPriority[] = [
  "urgent",
  "high",
  "medium",
  "low",
  "none",
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

export function PrioritySelect({
  value,
  onChange,
  disabled = false,
  showLabel = true,
  size = "md",
}: PrioritySelectProps) {
  const t = useTranslations();
  const [isOpen, setIsOpen] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);

  const currentConfig = priorityConfig[value] || priorityConfig.none;
  const PriorityIcon = currentConfig.icon;

  const handleSelect = useCallback(async (priority: TicketPriority) => {
    if (priority === value || disabled) return;

    setIsUpdating(true);
    setIsOpen(false);

    try {
      await onChange(priority);
    } catch (error) {
      console.error("Failed to update priority:", error);
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
          <PriorityIcon className={cn(iconSizeClasses[size], currentConfig.color)} />
        )}
        {showLabel && (
          <span className={cn("font-medium", currentConfig.color)}>
            {t(`tickets.priority.${value}`)}
          </span>
        )}
        <ChevronDownIcon className={cn("h-3 w-3 text-muted-foreground", !showLabel && "ml-0.5")} />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-44">
        {priorityOrder.map((priority) => {
          const config = priorityConfig[priority];
          const Icon = config.icon;
          const isSelected = priority === value;

          return (
            <DropdownMenuItem
              key={priority}
              onClick={() => handleSelect(priority)}
              className={cn(
                "flex items-center gap-2 cursor-pointer",
                isSelected && "bg-muted"
              )}
            >
              <Icon className={cn("h-4 w-4", config.color)} />
              <span className={isSelected ? "font-medium" : ""}>
                {t(`tickets.priority.${priority}`)}
              </span>
              {isSelected && (
                <Check className="h-3 w-3 ml-auto text-primary" />
              )}
            </DropdownMenuItem>
          );
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export default PrioritySelect;
