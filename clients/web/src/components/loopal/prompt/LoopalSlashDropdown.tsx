"use client";

import type { SlashCommand } from "./loopalCommands";

export function LoopalSlashDropdown({
  commands,
  activeIndex,
  onSelect,
  visible,
}: {
  commands: SlashCommand[];
  activeIndex: number;
  onSelect: (cmd: SlashCommand) => void;
  visible: boolean;
}) {
  if (!visible || commands.length === 0) return null;
  return (
    <div
      data-testid="loopal-slash-dropdown"
      className="absolute bottom-full left-3 right-3 mb-2 max-h-64 overflow-auto rounded-md border border-border bg-popover p-1 shadow-md"
    >
      {commands.map((c, i) => (
        <button
          key={c.id}
          type="button"
          // preventDefault keeps the textarea focused through the click.
          onMouseDown={(e) => {
            e.preventDefault();
            onSelect(c);
          }}
          className={`flex w-full items-center justify-between gap-3 rounded px-2 py-1.5 text-left ${
            i === activeIndex ? "bg-muted" : "hover:bg-muted/50"
          }`}
        >
          <span className="shrink-0 font-mono text-xs text-foreground">{c.label}</span>
          <span className="truncate text-[10px] text-muted-foreground">{c.hint}</span>
        </button>
      ))}
    </div>
  );
}
