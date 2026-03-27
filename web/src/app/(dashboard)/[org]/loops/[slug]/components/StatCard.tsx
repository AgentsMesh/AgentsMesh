import { cn } from "@/lib/utils";

interface StatCardProps {
  icon: React.ElementType;
  iconColor?: string;
  label: string;
  value: string;
  suffix?: string;
  note?: string;
}

export function StatCard({
  icon: Icon,
  iconColor,
  label,
  value,
  suffix,
  note,
}: StatCardProps) {
  return (
    <div className="border rounded-xl p-4 bg-card">
      <div className="flex items-center gap-1.5 text-xs text-muted-foreground mb-2">
        <Icon className={cn("w-3.5 h-3.5", iconColor)} />
        {label}
      </div>
      <div className="flex items-baseline gap-1.5">
        <span className="text-2xl font-bold tabular-nums tracking-tight">{value}</span>
        {suffix && (
          <span className="text-sm font-medium text-muted-foreground">{suffix}</span>
        )}
      </div>
      {note && (
        <p className="text-[10px] text-muted-foreground/60 mt-1">{note}</p>
      )}
    </div>
  );
}
