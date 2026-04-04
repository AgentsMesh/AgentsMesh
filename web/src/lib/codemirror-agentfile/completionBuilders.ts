/**
 * Completion builders for AgentFile autocomplete.
 * Converts domain data (agents, repos, credentials, config) into CodeMirror completions.
 */
import type { Completion } from "@codemirror/autocomplete";
import type { ConfigField } from "@/lib/api/agent";
import type { AgentfileCompletionContext } from "./autocomplete";

// ---------------------------------------------------------------------------
// Static value completions
// ---------------------------------------------------------------------------

export const MODE_VALUES: Completion[] = [
  { label: "pty", type: "constant", detail: "Terminal mode" },
  { label: "acp", type: "constant", detail: "Conversational mode" },
];

export const GIT_CREDENTIAL_VALUES: Completion[] = [
  { label: "http", type: "constant", detail: "HTTP credentials" },
  { label: "ssh", type: "constant", detail: "SSH key" },
  { label: "token", type: "constant", detail: "Personal access token" },
];

export const PROMPT_POSITION_VALUES: Completion[] = [
  { label: "prepend", type: "constant", detail: "Prepend to launch args" },
  { label: "append", type: "constant", detail: "Append to launch args" },
  { label: "none", type: "constant", detail: "Do not inject prompt" },
];

// ---------------------------------------------------------------------------
// Dynamic builders
// ---------------------------------------------------------------------------

export function buildFieldCompletions(fields: ConfigField[]): Completion[] {
  return fields.map((f) => ({
    label: f.name,
    type: "property",
    detail: f.type + (f.required ? " (required)" : ""),
    apply: `${f.name} = `,
  }));
}

export function buildValueCompletions(field: ConfigField): Completion[] {
  if (field.options && field.options.length > 0) {
    return field.options.map((opt) => ({
      label: `"${opt.value}"`,
      type: "constant",
      detail: field.name,
    }));
  }
  if (field.type === "boolean") {
    return [
      { label: "true", type: "constant" },
      { label: "false", type: "constant" },
    ];
  }
  if (field.type === "model_list") {
    return [
      { label: '"sonnet"', type: "constant", detail: "Claude Sonnet" },
      { label: '"opus"', type: "constant", detail: "Claude Opus" },
      { label: '"haiku"', type: "constant", detail: "Claude Haiku" },
    ];
  }
  return [];
}

export function buildAgentCompletions(
  agents: AgentfileCompletionContext["agents"]
): Completion[] {
  if (!agents?.length) return [];
  return agents.map((a) => ({
    label: a.slug,
    type: "constant",
    detail: a.name,
  }));
}

export function buildRepoCompletions(
  repos: AgentfileCompletionContext["repositories"]
): Completion[] {
  if (!repos?.length) return [];
  return repos.map((r) => ({
    label: `"${r.slug}"`,
    type: "constant",
    detail: r.name,
  }));
}

export function buildBranchCompletions(
  repos: AgentfileCompletionContext["repositories"]
): Completion[] {
  const branches = new Set<string>(["main", "master", "develop"]);
  repos?.forEach((r) => { if (r.default_branch) branches.add(r.default_branch); });
  return Array.from(branches).map((b) => ({
    label: `"${b}"`,
    type: "constant",
    detail: "branch",
  }));
}

export function buildCredentialCompletions(
  profiles: AgentfileCompletionContext["credentialProfiles"]
): Completion[] {
  if (!profiles?.length) return [];
  return profiles.map((p) => ({
    label: `"${p.name}"`,
    type: "constant",
    detail: p.description || "credential profile",
  }));
}
