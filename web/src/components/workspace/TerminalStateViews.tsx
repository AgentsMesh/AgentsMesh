"use client";

import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import {
  X,
  Loader2,
  AlertCircle,
  CheckCircle2,
  RefreshCw,
  Clock,
} from "lucide-react";

interface InitProgress {
  progress: number;
  phase: string;
  message: string;
}

interface TerminalLoadingStateProps {
  podStatus: string;
  initProgress?: InitProgress;
  onClose?: () => void;
}

/** Threshold in seconds after which we show a "taking longer than expected" warning */
const SLOW_INIT_THRESHOLD_SEC = 120;

function useElapsedSeconds() {
  const [elapsed, setElapsed] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      setElapsed((prev) => prev + 1);
    }, 1000);
    return () => clearInterval(interval);
  }, []);

  return elapsed;
}

function formatElapsed(totalSeconds: number): string {
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  if (minutes === 0) return `${seconds}s`;
  return `${minutes}m ${seconds.toString().padStart(2, "0")}s`;
}

/**
 * Loading/Waiting state view for TerminalPane
 */
export function TerminalLoadingState({
  podStatus,
  initProgress,
  onClose,
}: TerminalLoadingStateProps) {
  const isCompleted = podStatus === "completed";
  const elapsed = useElapsedSeconds();
  const isSlowInit = !isCompleted && elapsed >= SLOW_INIT_THRESHOLD_SEC;

  return (
    <div className="flex-1 flex items-center justify-center bg-terminal-bg">
      <div className="text-center p-4 max-w-sm">
        {isCompleted ? (
          <CheckCircle2 className="w-12 h-12 text-green-500 dark:text-green-400 mx-auto mb-3" />
        ) : (
          <Loader2 className="w-12 h-12 text-primary animate-spin mx-auto mb-3" />
        )}
        <p className="text-terminal-text font-medium mb-1">
          {isCompleted
            ? "Pod completed"
            : initProgress?.message || "Waiting for Pod to be ready..."}
        </p>
        {initProgress ? (
          <div className="mt-3 space-y-2">
            <Progress value={initProgress.progress} className="h-2" />
            <p className="text-xs text-terminal-text-muted">
              {initProgress.phase} - {initProgress.progress}%
            </p>
          </div>
        ) : (
          <p className="text-sm text-terminal-text-muted">
            {isCompleted ? (
              <>Status: <span className="text-green-500 dark:text-green-400">{podStatus}</span></>
            ) : (
              <>Status: <span className="text-yellow-500 dark:text-yellow-400">{podStatus}</span></>
            )}
          </p>
        )}
        {/* Elapsed time indicator for non-completed states */}
        {!isCompleted && (
          <p className="text-xs text-terminal-text-muted mt-2 flex items-center justify-center gap-1">
            <Clock className="w-3 h-3" />
            {formatElapsed(elapsed)}
          </p>
        )}
        {/* Slow initialization warning */}
        {isSlowInit && (
          <div className="mt-3 p-2 rounded bg-yellow-500/10 border border-yellow-500/30">
            <p className="text-xs text-yellow-500 dark:text-yellow-400">
              Taking longer than expected. The runner may be cloning a large repository or experiencing connectivity issues.
            </p>
          </div>
        )}
        {/* Show close button when status is unknown, completed, or init is slow */}
        {(podStatus === "unknown" || isCompleted || isSlowInit) && onClose && (
          <Button
            variant="outline"
            size="sm"
            className="mt-4 text-red-500 dark:text-red-400 border-red-500/50 hover:bg-red-500/10"
            onClick={onClose}
          >
            <X className="w-4 h-4 mr-2" />
            Close Terminal
          </Button>
        )}
      </div>
    </div>
  );
}

interface TerminalReconnectingStateProps {
  onClose?: () => void;
}

/**
 * Reconnecting state view for TerminalPane - shown when pod is orphaned
 * (Runner is restarting and the session will resume automatically)
 */
export function TerminalReconnectingState({
  onClose,
}: TerminalReconnectingStateProps) {
  return (
    <div className="flex-1 flex items-center justify-center bg-terminal-bg">
      <div className="text-center p-4 max-w-sm">
        <RefreshCw className="w-12 h-12 text-amber-500 dark:text-amber-400 mx-auto mb-3 animate-spin" />
        <p className="text-terminal-text font-medium mb-1">
          Runner is restarting...
        </p>
        <p className="text-sm text-terminal-text-muted mb-4">
          Session will resume automatically. Your work is preserved.
        </p>
        {onClose && (
          <Button
            variant="outline"
            size="sm"
            className="text-muted-foreground border-border hover:bg-muted"
            onClick={onClose}
          >
            <X className="w-4 h-4 mr-2" />
            Close Terminal
          </Button>
        )}
      </div>
    </div>
  );
}

interface TerminalErrorStateProps {
  error: string;
  onClose?: () => void;
}

/**
 * Error state view for TerminalPane
 */
export function TerminalErrorState({
  error,
  onClose,
}: TerminalErrorStateProps) {
  return (
    <div className="flex-1 flex items-center justify-center bg-terminal-bg">
      <div className="text-center p-4">
        <AlertCircle className="w-12 h-12 text-red-500 dark:text-red-400 mx-auto mb-3" />
        <p className="text-terminal-text font-medium mb-1">{error}</p>
        <p className="text-sm text-terminal-text-muted mb-4">
          The pod cannot be connected. Please check the pod status or create a new one.
        </p>
        {onClose && (
          <Button
            variant="outline"
            size="sm"
            className="text-red-500 dark:text-red-400 border-red-500/50 hover:bg-red-500/10"
            onClick={onClose}
          >
            <X className="w-4 h-4 mr-2" />
            Close Terminal
          </Button>
        )}
      </div>
    </div>
  );
}

// Named exports only - no default export needed
