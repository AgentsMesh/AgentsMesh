"use client";

import { cn } from "@/lib/utils";

interface FilterOption {
  id: string;
  label: string;
  count: number;
  dotColor?: string;
}

interface MeshFilterSectionProps {
  title: string;
  options: FilterOption[];
  selected: Set<string>;
  onToggle: (id: string) => void;
}

export function MeshFilterSection({ title, options, selected, onToggle }: MeshFilterSectionProps) {
  if (options.length === 0) return null;

  return (
    <div className="border-t border-border px-3 py-3">
      <div className="mb-2 text-[10px] font-semibold uppercase tracking-[0.12em] text-muted-foreground/80">
        {title}
      </div>
      <ul className="space-y-0.5">
        {options.map((opt) => {
          const isOn = selected.has(opt.id);
          return (
            <li key={opt.id}>
              <label
                className={cn(
                  "flex cursor-pointer items-center gap-2 rounded-md px-2 py-1.5 transition-colors",
                  "hover:bg-muted",
                )}
              >
                <input
                  type="checkbox"
                  checked={isOn}
                  onChange={() => onToggle(opt.id)}
                  className="h-3.5 w-3.5 rounded border-border text-primary focus:ring-0"
                />
                {opt.dotColor && (
                  <span
                    className="h-2 w-2 flex-shrink-0 rounded-full"
                    style={{ backgroundColor: opt.dotColor }}
                  />
                )}
                <span className="min-w-0 flex-1 truncate text-[12px] text-foreground">
                  {opt.label}
                </span>
                <span className="font-mono text-[11px] text-muted-foreground">{opt.count}</span>
              </label>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
