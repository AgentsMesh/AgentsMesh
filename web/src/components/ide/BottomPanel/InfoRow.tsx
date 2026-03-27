import React from "react";
import Link from "next/link";
import { cn } from "@/lib/utils";

interface InfoRowProps {
  icon: React.ReactNode;
  label: string;
  value: React.ReactNode;
  mono?: boolean;
  href?: string;
  className?: string;
  valueClassName?: string;
}

export function InfoRow({
  icon,
  label,
  value,
  mono,
  href,
  className,
  valueClassName,
}: InfoRowProps) {
  const valueContent = href ? (
    <Link
      href={href}
      className={cn(
        "text-xs truncate hover:underline text-primary",
        mono && "font-mono",
        valueClassName
      )}
      title={typeof value === "string" ? value : undefined}
    >
      {value}
    </Link>
  ) : (
    <span
      className={cn(
        "text-xs truncate",
        mono && "font-mono",
        valueClassName
      )}
      title={typeof value === "string" ? value : undefined}
    >
      {value}
    </span>
  );

  return (
    <div className={cn("flex items-start gap-1.5 min-w-0", className)}>
      <span className="text-muted-foreground mt-0.5 flex-shrink-0">{icon}</span>
      <span className="text-[10px] text-muted-foreground whitespace-nowrap flex-shrink-0">
        {label}:
      </span>
      {valueContent}
    </div>
  );
}
