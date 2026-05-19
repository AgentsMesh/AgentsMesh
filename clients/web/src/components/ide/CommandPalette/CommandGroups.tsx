"use client";

import { Command } from "cmdk";
import type { CommandItemData } from "./types";

interface CommandGroupsProps {
  navigationCommands: CommandItemData[];
  actionCommands: CommandItemData[];
  onSelect: (item: CommandItemData) => void;
  t: (key: string) => string;
}

export function CommandGroups({
  navigationCommands,
  actionCommands,
  onSelect,
  t,
}: CommandGroupsProps) {
  return (
    <>
      {/* Navigation */}
      <Command.Group heading={t("commandPalette.navigation")}>
        {navigationCommands.map((cmd) => (
          <Command.Item
            key={cmd.id}
            value={cmd.label}
            keywords={cmd.keywords}
            className="flex items-center gap-3 px-3 py-2 rounded cursor-pointer aria-selected:bg-muted"
            onSelect={() => onSelect(cmd)}
          >
            <span className="text-muted-foreground">{cmd.icon}</span>
            <div className="flex-1 min-w-0">
              <div className="text-sm">{cmd.label}</div>
              {cmd.description && (
                <div className="text-xs text-muted-foreground">{cmd.description}</div>
              )}
            </div>
          </Command.Item>
        ))}
      </Command.Group>

      {/* Actions */}
      <Command.Group heading={t("commandPalette.actions")}>
        {actionCommands.map((cmd) => (
          <Command.Item
            key={cmd.id}
            value={cmd.label}
            keywords={cmd.keywords}
            className="flex items-center gap-3 px-3 py-2 rounded cursor-pointer aria-selected:bg-muted"
            onSelect={() => onSelect(cmd)}
          >
            <span className="text-muted-foreground">{cmd.icon}</span>
            <div className="flex-1 min-w-0">
              <div className="text-sm">{cmd.label}</div>
              {cmd.description && (
                <div className="text-xs text-muted-foreground">{cmd.description}</div>
              )}
            </div>
          </Command.Item>
        ))}
      </Command.Group>
    </>
  );
}

export default CommandGroups;
