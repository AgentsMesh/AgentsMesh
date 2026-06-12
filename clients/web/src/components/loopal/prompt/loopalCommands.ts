export type SlashAction =
  | { kind: "send"; subtype: string; payload?: Record<string, unknown> }
  | { kind: "popover"; popover: "model" | "rewind" | "resume" }
  | { kind: "goal-create" };

export interface SlashCommand {
  id: string;
  label: string;
  hint: string;
  action: SlashAction;
  hasArg?: boolean;
}

// Command list mirrors loopal TUI's builtin slash commands. Goal sub-commands
// only appear when a goal exists (matches loopal's multi-state /goal). hint is
// resolved through the caller's translator so the dropdown is localized.
export function buildLoopalCommands({
  hasGoal,
  t,
}: {
  hasGoal: boolean;
  t: (k: string) => string;
}): SlashCommand[] {
  const cmds: SlashCommand[] = [
    { id: "act", label: "/act", hint: t("commands.act"), action: { kind: "send", subtype: "loopal.mode", payload: { mode: "act" } } },
    { id: "plan", label: "/plan", hint: t("commands.plan"), action: { kind: "send", subtype: "loopal.mode", payload: { mode: "plan" } } },
    { id: "model", label: "/model", hint: t("commands.model"), action: { kind: "popover", popover: "model" } },
    { id: "thinking", label: "/thinking", hint: t("commands.thinking"), action: { kind: "popover", popover: "model" } },
    { id: "compact", label: "/compact", hint: t("commands.compact"), action: { kind: "send", subtype: "loopal.compact" } },
    { id: "clear", label: "/clear", hint: t("commands.clear"), action: { kind: "send", subtype: "loopal.clear" } },
    { id: "rewind", label: "/rewind", hint: t("commands.rewind"), action: { kind: "popover", popover: "rewind" } },
    { id: "resume", label: "/resume", hint: t("commands.resume"), action: { kind: "popover", popover: "resume" } },
    { id: "suspend", label: "/suspend", hint: t("commands.suspend"), action: { kind: "send", subtype: "loopal.suspend" } },
    { id: "unsuspend", label: "/unsuspend", hint: t("commands.unsuspend"), action: { kind: "send", subtype: "loopal.unsuspend" } },
    { id: "goal", label: "/goal", hint: t("commands.goal"), action: { kind: "goal-create" }, hasArg: true },
    { id: "mcp-refresh", label: "/mcp-refresh", hint: t("commands.mcpRefresh"), action: { kind: "send", subtype: "loopal.mcpStatus" } },
  ];
  if (hasGoal) {
    cmds.push(
      { id: "goal-pause", label: "/goal-pause", hint: t("commands.goalPause"), action: { kind: "send", subtype: "loopal.goalPause" } },
      { id: "goal-resume", label: "/goal-resume", hint: t("commands.goalResume"), action: { kind: "send", subtype: "loopal.goalResume" } },
      { id: "goal-complete", label: "/goal-complete", hint: t("commands.goalComplete"), action: { kind: "send", subtype: "loopal.goalComplete" } },
      { id: "goal-reopen", label: "/goal-reopen", hint: t("commands.goalReopen"), action: { kind: "send", subtype: "loopal.goalReopen" } },
      { id: "goal-clear", label: "/goal-clear", hint: t("commands.goalClear"), action: { kind: "send", subtype: "loopal.goalClear" } },
    );
  }
  return cmds;
}

// Resolve a fully-typed "/cmd arg..." line to its command + argument.
export function parseSlashInput(
  text: string,
  commands: SlashCommand[],
): { command: SlashCommand; arg: string } | null {
  if (!text.startsWith("/")) return null;
  const sp = text.indexOf(" ");
  const name = (sp < 0 ? text.slice(1) : text.slice(1, sp)).toLowerCase();
  const arg = sp < 0 ? "" : text.slice(sp + 1).trim();
  const command = commands.find((c) => c.id === name);
  return command ? { command, arg } : null;
}

export function filterCommands(commands: SlashCommand[], query: string): SlashCommand[] {
  const q = query.toLowerCase();
  if (!q) return commands;
  return commands.filter((c) => c.id.startsWith(q));
}
