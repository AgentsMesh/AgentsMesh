"use client";

import { X } from "lucide-react";
import { cn } from "@/lib/utils";

type AlertType = "error" | "success" | "warning" | "info";

interface AlertMessageProps {
  type: AlertType;
  message: string;
  onDismiss?: () => void;
  className?: string;
}

const alertStyles: Record<AlertType, { container: string; text: string }> = {
  error: {
    container: "bg-destructive/15 border-destructive/30",
    text: "text-destructive",
  },
  success: {
    container: "bg-green-500/15 border-green-500/30",
    text: "text-green-600 dark:text-green-400",
  },
  warning: {
    container: "bg-yellow-500/15 border-yellow-500/30",
    text: "text-yellow-600 dark:text-yellow-400",
  },
  info: {
    container: "bg-blue-500/15 border-blue-500/30",
    text: "text-blue-600 dark:text-blue-400",
  },
};

export function AlertMessage({ type, message, onDismiss, className }: AlertMessageProps) {
  const styles = alertStyles[type];

  return (
    <div
      role="alert"
      aria-live={type === "error" ? "assertive" : "polite"}
      className={cn(
        "flex items-start justify-between gap-2 p-3 rounded-md border text-sm",
        styles.container,
        className
      )}
    >
      <p className={styles.text}>{message}</p>
      {onDismiss && (
        <button
          type="button"
          onClick={onDismiss}
          className={cn(
            "flex-shrink-0 p-0.5 rounded hover:bg-black/10 dark:hover:bg-white/10 transition-colors",
            styles.text
          )}
          aria-label="Dismiss"
        >
          <X className="w-4 h-4" />
        </button>
      )}
    </div>
  );
}

export default AlertMessage;
