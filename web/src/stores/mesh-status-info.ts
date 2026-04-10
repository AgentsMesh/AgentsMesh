import { Play, Hourglass, Pause, type LucideIcon } from "lucide-react";

export const getPodStatusInfo = (status: string) => {
  const statusMap: Record<
    string,
    { label: string; color: string; bgColor: string }
  > = {
    initializing: {
      label: "Initializing",
      color: "text-blue-600 dark:text-blue-400",
      bgColor: "bg-blue-100 dark:bg-blue-900/30",
    },
    running: {
      label: "Running",
      color: "text-green-600 dark:text-green-400",
      bgColor: "bg-green-100 dark:bg-green-900/30",
    },
    paused: {
      label: "Paused",
      color: "text-yellow-600 dark:text-yellow-400",
      bgColor: "bg-yellow-100 dark:bg-yellow-900/30",
    },
    terminated: {
      label: "Terminated",
      color: "text-gray-600 dark:text-gray-400",
      bgColor: "bg-gray-100 dark:bg-gray-800",
    },
    failed: {
      label: "Failed",
      color: "text-red-600 dark:text-red-400",
      bgColor: "bg-red-100 dark:bg-red-900/30",
    },
  };
  return statusMap[status] || statusMap.terminated;
};

export const getAgentStatusInfo = (agentStatus: string): {
  label: string;
  color: string;
  dotColor: string;
  bgColor: string;
  icon: LucideIcon;
} => {
  const statusMap: Record<string, {
    label: string;
    color: string;
    dotColor: string;
    bgColor: string;
    icon: LucideIcon;
  }> = {
    executing: {
      label: "Executing",
      color: "text-green-600 dark:text-green-400",
      dotColor: "bg-green-500",
      bgColor: "bg-green-500/10",
      icon: Play,
    },
    waiting: {
      label: "Waiting for Input",
      color: "text-amber-600 dark:text-amber-400",
      dotColor: "bg-amber-500",
      bgColor: "bg-amber-500/10",
      icon: Hourglass,
    },
    idle: {
      label: "Idle",
      color: "text-gray-500 dark:text-gray-400",
      dotColor: "bg-gray-400",
      bgColor: "bg-gray-400/10",
      icon: Pause,
    },
  };
  return statusMap[agentStatus] || statusMap.idle;
};

export const getBindingStatusInfo = (status: string) => {
  const statusMap: Record<
    string,
    { label: string; color: string }
  > = {
    active: { label: "Active", color: "stroke-green-500" },
    pending: { label: "Pending", color: "stroke-yellow-500" },
    revoked: { label: "Revoked", color: "stroke-red-500" },
    expired: { label: "Expired", color: "stroke-gray-500" },
  };
  return statusMap[status] || statusMap.active;
};
