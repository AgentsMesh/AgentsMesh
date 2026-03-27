import type { TicketStatus, TicketPriority } from "./ticket-types";

// Helper function to get status display info
export const getStatusInfo = (status: TicketStatus) => {
  const statusMap: Record<
    TicketStatus,
    { label: string; color: string; bgColor: string }
  > = {
    backlog: { label: "Backlog", color: "text-gray-600 dark:text-gray-400", bgColor: "bg-gray-100 dark:bg-gray-800" },
    todo: { label: "To Do", color: "text-blue-600 dark:text-blue-400", bgColor: "bg-blue-100 dark:bg-blue-900/30" },
    in_progress: { label: "In Progress", color: "text-yellow-600 dark:text-yellow-400", bgColor: "bg-yellow-100 dark:bg-yellow-900/30" },
    in_review: { label: "In Review", color: "text-purple-600 dark:text-purple-400", bgColor: "bg-purple-100 dark:bg-purple-900/30" },
    done: { label: "Done", color: "text-green-600 dark:text-green-400", bgColor: "bg-green-100 dark:bg-green-900/30" },
  };
  // Return default if status not found
  return statusMap[status] || { label: status || "Unknown", color: "text-gray-500 dark:text-gray-400", bgColor: "bg-gray-100 dark:bg-gray-800" };
};

// Helper function to get priority display info
export const getPriorityInfo = (priority: TicketPriority) => {
  const priorityMap: Record<
    TicketPriority,
    { label: string; color: string; icon: string }
  > = {
    none: { label: "None", color: "text-gray-400 dark:text-gray-500", icon: "—" },
    low: { label: "Low", color: "text-green-500 dark:text-green-400", icon: "↓" },
    medium: { label: "Medium", color: "text-yellow-500 dark:text-yellow-400", icon: "→" },
    high: { label: "High", color: "text-orange-500 dark:text-orange-400", icon: "↑" },
    urgent: { label: "Urgent", color: "text-red-500 dark:text-red-400", icon: "⚡" },
  };
  // Return default if priority not found
  return priorityMap[priority] || { label: priority || "Unknown", color: "text-gray-400 dark:text-gray-500", icon: "?" };
};
