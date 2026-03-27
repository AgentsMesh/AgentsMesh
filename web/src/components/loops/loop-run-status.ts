import {
  CheckCircle2, XCircle, Clock, Ban, SkipForward, Loader2, AlertTriangle,
} from "lucide-react";

interface StatusConfig {
  icon: React.ElementType;
  color: string;
  bg: string;
  labelKey: string;
}

export const STATUS_CONFIG: Record<string, StatusConfig> = {
  completed: { icon: CheckCircle2, color: "text-emerald-600 dark:text-emerald-400", bg: "bg-emerald-500/10", labelKey: "loops.statusCompleted" },
  failed: { icon: XCircle, color: "text-red-600 dark:text-red-400", bg: "bg-red-500/10", labelKey: "loops.statusFailed" },
  timeout: { icon: AlertTriangle, color: "text-amber-600 dark:text-amber-400", bg: "bg-amber-500/10", labelKey: "loops.statusTimeout" },
  cancelled: { icon: Ban, color: "text-gray-500", bg: "bg-gray-500/10", labelKey: "loops.statusCancelled" },
  skipped: { icon: SkipForward, color: "text-gray-400", bg: "bg-gray-500/10", labelKey: "loops.statusSkipped" },
  running: { icon: Loader2, color: "text-blue-600 dark:text-blue-400", bg: "bg-blue-500/10", labelKey: "loops.statusRunning" },
  pending: { icon: Clock, color: "text-yellow-600 dark:text-yellow-400", bg: "bg-yellow-500/10", labelKey: "loops.statusPending" },
};
