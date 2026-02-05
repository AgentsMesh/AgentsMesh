import { cn } from "@/lib/utils";

interface SpinnerProps {
  size?: "sm" | "md" | "lg";
  className?: string;
}

const sizeClasses = {
  sm: "h-4 w-4",
  md: "h-8 w-8",
  lg: "h-12 w-12",
};

/**
 * Spinner - Unified loading spinner component
 */
export function Spinner({ size = "md", className }: SpinnerProps) {
  return (
    <div
      className={cn(
        "animate-spin rounded-full border-b-2 border-primary",
        sizeClasses[size],
        className
      )}
    />
  );
}

/**
 * CenteredSpinner - Spinner centered in its container
 */
export function CenteredSpinner({ size = "md", className }: SpinnerProps) {
  return (
    <div className={cn("flex items-center justify-center h-full", className)}>
      <Spinner size={size} />
    </div>
  );
}
