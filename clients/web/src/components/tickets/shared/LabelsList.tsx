"use client";

import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

interface Label {
  id: number;
  name: string;
  color: string;
}

interface LabelsListProps {
  labels: Label[];
  compact?: boolean;
  className?: string;
}

export function LabelsList({
  labels,
  compact = false,
  className,
}: LabelsListProps) {
  if (!labels || labels.length === 0) return null;

  if (compact) {
    return (
      <div className={cn("flex flex-wrap gap-1.5", className)}>
        {labels.map((label) => (
          <Badge
            key={label.id}
            style={{
              backgroundColor: `${label.color}15`,
              color: label.color,
            }}
            className="text-xs font-normal border-0"
          >
            {label.name}
          </Badge>
        ))}
      </div>
    );
  }

  return (
    <div className={cn("flex flex-wrap gap-2 mb-6", className)}>
      {labels.map((label) => (
        <span
          key={label.id}
          className="px-2.5 py-1 rounded-md text-sm font-medium"
          style={{
            backgroundColor: `${label.color}15`,
            color: label.color,
          }}
        >
          {label.name}
        </span>
      ))}
    </div>
  );
}

export default LabelsList;
