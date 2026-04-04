/**
 * AgentFile autocomplete extension for CodeMirror.
 *
 * Provides context-aware completions:
 * 1. Declaration keywords at line start (CONFIG, ENV, MODE, REPO, etc.)
 * 2. CONFIG field names from agent schema
 * 3. CONFIG field values from schema options/defaults
 * 4. Keyword-specific data completions (AGENT → slug, REPO → URL, etc.)
 */
import type { CompletionContext, CompletionResult, Completion } from "@codemirror/autocomplete";
import type { ConfigField } from "@/lib/api/agent";
import {
  MODE_VALUES, GIT_CREDENTIAL_VALUES, PROMPT_POSITION_VALUES,
  buildFieldCompletions, buildValueCompletions,
  buildAgentCompletions, buildRepoCompletions,
  buildBranchCompletions, buildCredentialCompletions,
} from "./completionBuilders";

/**
 * Context data for AgentFile autocomplete.
 * Provides domain-specific candidates for each keyword.
 */
export interface AgentfileCompletionContext {
  configFields: ConfigField[];
  agents?: { slug: string; name: string }[];
  repositories?: { slug: string; name: string; default_branch: string }[];
  credentialProfiles?: { name: string; description?: string }[];
}

// ---------------------------------------------------------------------------
// Static keyword completions
// ---------------------------------------------------------------------------

const DECLARATION_COMPLETIONS: Completion[] = [
  { label: "AGENT", type: "keyword", detail: "Agent identifier" },
  { label: "CONFIG", type: "keyword", detail: "Configuration key = value" },
  { label: "ENV", type: "keyword", detail: "Environment variable" },
  { label: "MODE", type: "keyword", detail: "Interaction mode (pty/acp)" },
  { label: "CREDENTIAL", type: "keyword", detail: "Credential profile name" },
  { label: "REPO", type: "keyword", detail: "Repository slug" },
  { label: "BRANCH", type: "keyword", detail: "Git branch name" },
  { label: "GIT_CREDENTIAL", type: "keyword", detail: "Git credential type" },
  { label: "PROMPT", type: "keyword", detail: "Initial prompt content" },
  { label: "PROMPT_POSITION", type: "keyword", detail: "Prompt position (prepend/append/none)" },
  { label: "MCP", type: "keyword", detail: "MCP server config" },
  { label: "SKILLS", type: "keyword", detail: "Skill module list" },
  { label: "SETUP", type: "keyword", detail: "Setup command block" },
  { label: "REMOVE", type: "keyword", detail: "Remove declaration or build artifact" },
  { label: "EXECUTABLE", type: "keyword", detail: "Agent executable path" },
];

const BUILD_COMPLETIONS: Completion[] = [
  { label: "arg", type: "keyword", detail: "Build argument" },
  { label: "file", type: "keyword", detail: "Create file" },
  { label: "mkdir", type: "keyword", detail: "Create directory" },
  { label: "if", type: "keyword", detail: "Conditional" },
  { label: "for", type: "keyword", detail: "Loop" },
];

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function findConfigFieldOnLine(text: string): string | null {
  const m = text.match(/^\s*CONFIG\s+(\w+)\s*=\s*/);
  return m ? m[1] : null;
}

/** Match "KEYWORD partial" for value completion. Returns [keyword, partialValue]. */
function matchKeywordValue(text: string): [string, string] | null {
  const m = text.match(
    /^\s*(AGENT|REPO|BRANCH|CREDENTIAL|GIT_CREDENTIAL|MODE|MCP|EXECUTABLE|PROMPT_POSITION)\s+"?([^"]*)$/
  );
  if (m) return [m[1], m[2]];
  const m2 = text.match(/^\s*(MODE|GIT_CREDENTIAL|PROMPT_POSITION)\s+(\w*)$/);
  if (m2) return [m2[1], m2[2]];
  return null;
}

function keywordValueOptions(kw: string, ctx: AgentfileCompletionContext): Completion[] {
  switch (kw) {
    case "MODE": return MODE_VALUES;
    case "AGENT": return buildAgentCompletions(ctx.agents);
    case "REPO": return buildRepoCompletions(ctx.repositories);
    case "BRANCH": return buildBranchCompletions(ctx.repositories);
    case "CREDENTIAL": return buildCredentialCompletions(ctx.credentialProfiles);
    case "GIT_CREDENTIAL": return GIT_CREDENTIAL_VALUES;
    case "PROMPT_POSITION": return PROMPT_POSITION_VALUES;
    default: return [];
  }
}

// ---------------------------------------------------------------------------
// Main completion source
// ---------------------------------------------------------------------------

export function agentfileCompletion(
  context: AgentfileCompletionContext
): (ctx: CompletionContext) => CompletionResult | null {
  return (ctx: CompletionContext): CompletionResult | null => {
    const line = ctx.state.doc.lineAt(ctx.pos);
    const textBefore = line.text.slice(0, ctx.pos - line.from);

    // 1. Empty line → all keywords
    if (/^\s*$/.test(textBefore)) {
      return { from: ctx.pos, options: [...DECLARATION_COMPLETIONS, ...BUILD_COMPLETIONS] };
    }

    // 2. CONFIG field name
    const cfgPrefix = textBefore.match(/^\s*CONFIG\s+(\w*)$/);
    if (cfgPrefix) {
      return { from: ctx.pos - cfgPrefix[1].length, options: buildFieldCompletions(context.configFields) };
    }

    // 3. CONFIG field value
    const fieldName = findConfigFieldOnLine(textBefore);
    if (fieldName) {
      const field = context.configFields.find((f) => f.name === fieldName);
      if (field) {
        const afterEq = textBefore.match(/=\s*(.*)$/);
        const partial = afterEq ? afterEq[1] : "";
        return { from: ctx.pos - partial.length, options: buildValueCompletions(field) };
      }
    }

    // 4. Keyword-specific value completions
    const kwMatch = matchKeywordValue(textBefore);
    if (kwMatch) {
      const [kw, partial] = kwMatch;
      const options = keywordValueOptions(kw, context);
      if (options.length > 0) return { from: ctx.pos - partial.length, options };
    }

    // 5. Partial keyword at line start
    const partialKw = textBefore.match(/^\s*([A-Za-z_]\w*)$/);
    if (partialKw) {
      const word = partialKw[1];
      const from = ctx.pos - word.length;
      const all = [...DECLARATION_COMPLETIONS, ...BUILD_COMPLETIONS];
      const filtered = all.filter((c) => c.label.toLowerCase().startsWith(word.toLowerCase()));
      if (filtered.length > 0) return { from, options: filtered };
    }

    return null;
  };
}
