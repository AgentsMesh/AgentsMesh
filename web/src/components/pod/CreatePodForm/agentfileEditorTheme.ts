/**
 * CodeMirror theme for AgentFile editor.
 * Uses CSS custom properties for dark/light mode compatibility.
 */
import { EditorView } from "@codemirror/view";

export const agentfileEditorTheme = EditorView.theme({
  "&": {
    fontSize: "12px",
    fontFamily: "ui-monospace, SFMono-Regular, 'SF Mono', Menlo, Consolas, monospace",
    minHeight: "120px",
    maxHeight: "300px",
  },
  "&.cm-focused": {
    outline: "none",
  },
  ".cm-scroller": {
    overflow: "auto",
  },
  ".cm-content": {
    padding: "8px 0",
  },
  ".cm-line": {
    padding: "0 12px",
  },
  // Syntax token colors — use hsl for theme compatibility
  ".cm-keyword": {
    color: "hsl(var(--chart-1, 220 70% 50%))",
    fontWeight: "600",
  },
  ".cm-string": {
    color: "hsl(var(--chart-2, 142 71% 45%))",
  },
  ".cm-number": {
    color: "hsl(var(--chart-3, 30 80% 55%))",
  },
  ".cm-bool": {
    color: "hsl(var(--chart-4, 280 65% 60%))",
    fontWeight: "600",
  },
  ".cm-comment": {
    color: "hsl(var(--muted-foreground, 0 0% 45%))",
    fontStyle: "italic",
  },
  ".cm-operator": {
    color: "hsl(var(--foreground, 0 0% 10%))",
  },
  ".cm-variable": {
    color: "hsl(var(--foreground, 0 0% 10%))",
  },
  ".cm-punctuation": {
    color: "hsl(var(--muted-foreground, 0 0% 45%))",
  },
  // Autocomplete popup
  ".cm-tooltip-autocomplete": {
    border: "1px solid hsl(var(--border))",
    borderRadius: "6px",
    backgroundColor: "hsl(var(--popover))",
    color: "hsl(var(--popover-foreground))",
    fontSize: "12px",
    boxShadow: "0 4px 6px -1px rgb(0 0 0 / 0.1)",
  },
  ".cm-tooltip-autocomplete > ul > li": {
    padding: "4px 8px",
  },
  ".cm-tooltip-autocomplete > ul > li[aria-selected]": {
    backgroundColor: "hsl(var(--accent))",
    color: "hsl(var(--accent-foreground))",
  },
  // Lint gutter / diagnostics
  ".cm-diagnostic-error": {
    borderLeft: "3px solid hsl(var(--destructive, 0 84% 60%))",
  },
  ".cm-diagnostic-warning": {
    borderLeft: "3px solid hsl(var(--chart-3, 30 80% 55%))",
  },
  // Cursor
  ".cm-cursor": {
    borderLeftColor: "hsl(var(--foreground))",
  },
  // Selection
  ".cm-selectionBackground": {
    backgroundColor: "hsl(var(--accent) / 0.4) !important",
  },
});
