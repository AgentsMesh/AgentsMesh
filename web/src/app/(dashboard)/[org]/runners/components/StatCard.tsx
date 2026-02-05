import { cn } from "@/lib/utils";

interface StatCardProps {
  title: string;
  value: number;
  icon: React.ReactNode;
  variant?: "success" | "warning" | "error";
}

/**
 * StatCard - Compact statistics card for displaying metrics
 */
export function StatCard({ title, value, icon, variant }: StatCardProps) {
  return (
    <div className="p-3 md:p-4 border border-border rounded-lg bg-card">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-xs md:text-sm text-muted-foreground">{title}</p>
          <p className="text-xl md:text-2xl font-bold">{value}</p>
        </div>
        <div
          className={cn(
            "w-8 h-8 md:w-10 md:h-10 rounded-lg flex items-center justify-center",
            variant === "success"
              ? "bg-green-500/10 text-green-500 dark:text-green-400"
              : variant === "warning"
                ? "bg-yellow-500/10 text-yellow-500 dark:text-yellow-400"
                : variant === "error"
                  ? "bg-red-500/10 text-red-500 dark:text-red-400"
                  : "bg-primary/10 text-primary"
          )}
        >
          {icon}
        </div>
      </div>
    </div>
  );
}
