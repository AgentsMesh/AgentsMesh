import type { Runner, RunnerStatus } from "./runner";

export const getRunnerStatusInfo = (status: RunnerStatus) => {
  const statusMap: Record<
    RunnerStatus,
    { label: string; color: string; dotColor: string }
  > = {
    online: {
      label: "Online",
      color: "text-green-600 dark:text-green-400",
      dotColor: "bg-green-500",
    },
    offline: {
      label: "Offline",
      color: "text-gray-500 dark:text-gray-400",
      dotColor: "bg-gray-400",
    },
    maintenance: {
      label: "Maintenance",
      color: "text-yellow-600 dark:text-yellow-400",
      dotColor: "bg-yellow-500",
    },
    busy: {
      label: "Busy",
      color: "text-orange-600 dark:text-orange-400",
      dotColor: "bg-orange-500",
    },
  };
  return statusMap[status];
};

export const canAcceptPods = (runner: Runner): boolean => {
  return (
    runner.status === "online" &&
    runner.current_pods < runner.max_concurrent_pods
  );
};

export const formatHostInfo = (hostInfo?: Runner["host_info"]) => {
  if (!hostInfo) return "Unknown";

  const parts: string[] = [];
  if (hostInfo.os) parts.push(hostInfo.os);
  if (hostInfo.arch) parts.push(hostInfo.arch);
  if (hostInfo.cpu_cores) parts.push(`${hostInfo.cpu_cores} cores`);
  if (hostInfo.memory) {
    const memoryGB = (hostInfo.memory / 1024 / 1024 / 1024).toFixed(1);
    parts.push(`${memoryGB}GB RAM`);
  }

  return parts.length > 0 ? parts.join(" / ") : "Unknown";
};
