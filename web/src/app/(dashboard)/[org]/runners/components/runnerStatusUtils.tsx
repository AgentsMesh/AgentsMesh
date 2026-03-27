import {
  CheckCircle,
  PowerOff,
  Activity,
  AlertCircle,
  Clock,
} from "lucide-react";
import { type RunnerData } from "@/lib/api";

export function getStatusIcon(status: RunnerData["status"]) {
  switch (status) {
    case "online":
      return <CheckCircle className="w-4 h-4 text-green-500 dark:text-green-400" />;
    case "offline":
      return <PowerOff className="w-4 h-4 text-gray-500 dark:text-gray-400" />;
    case "busy":
      return <Activity className="w-4 h-4 text-yellow-500 dark:text-yellow-400" />;
    case "maintenance":
      return <AlertCircle className="w-4 h-4 text-orange-500 dark:text-orange-400" />;
    default:
      return <Clock className="w-4 h-4 text-gray-400 dark:text-gray-500" />;
  }
}

export function getStatusColor(status: RunnerData["status"]) {
  switch (status) {
    case "online":
      return "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400";
    case "offline":
      return "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400";
    case "busy":
      return "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400";
    case "maintenance":
      return "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400";
    default:
      return "bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-400";
  }
}
