/**
 * CodeMirror 6 extensions for AgentFile DSL.
 *
 * Provides:
 * - Syntax highlighting (keyword, string, number, comment coloring)
 * - Autocomplete (keywords + context-aware data completions per keyword)
 * - Lint (real-time syntax error checking)
 */
export { agentfileLanguage, agentfileSyntaxHighlighting } from "./highlight";
export { agentfileCompletion } from "./autocomplete";
export type { AgentfileCompletionContext } from "./autocomplete";
export { agentfileLinter } from "./lint";
