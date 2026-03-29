/**
 * CodeMirror 6 extensions for PodFile DSL.
 *
 * Provides:
 * - Syntax highlighting (keyword, string, number, comment coloring)
 * - Autocomplete (keywords + context-aware data completions per keyword)
 * - Lint (real-time syntax error checking)
 */
export { podfileLanguage, podfileSyntaxHighlighting } from "./highlight";
export { podfileCompletion } from "./autocomplete";
export type { PodfileCompletionContext } from "./autocomplete";
export { podfileLinter } from "./lint";
