/**
 * AgentFile lint extension for CodeMirror.
 *
 * Provides real-time syntax error highlighting by validating each line
 * against known AgentFile declaration patterns.
 */
import type { Diagnostic } from "@codemirror/lint";
import type { EditorView } from "@codemirror/view";

const VALID_LINE_STARTERS = new Set([
  "AGENT", "EXECUTABLE", "CONFIG", "ENV", "REPO", "BRANCH",
  "GIT_CREDENTIAL", "MCP", "SKILLS", "SETUP",
  "REMOVE", "MODE", "CREDENTIAL",
  "PROMPT", "PROMPT_POSITION",
  "arg", "file", "mkdir",
  "when", "if", "else", "for",
]);

const REQUIRES_VALUE = new Set([
  "AGENT", "EXECUTABLE", "REPO", "BRANCH", "GIT_CREDENTIAL",
  "MODE", "CREDENTIAL", "PROMPT", "PROMPT_POSITION",
]);

const VALID_MODES = new Set(["pty", "acp"]);

export function agentfileLinter(view: EditorView): Diagnostic[] {
  const diagnostics: Diagnostic[] = [];
  const doc = view.state.doc;

  let inHeredoc = false;
  let heredocMarker = "";

  for (let i = 1; i <= doc.lines; i++) {
    const line = doc.line(i);
    const text = line.text.trim();

    if (!text || text.startsWith("#")) continue;

    if (inHeredoc) {
      if (text === heredocMarker) {
        inHeredoc = false;
        heredocMarker = "";
      }
      continue;
    }

    const heredocMatch = text.match(/<<([A-Z_]+)\s*$/);
    if (heredocMatch) {
      inHeredoc = true;
      heredocMarker = heredocMatch[1];
      continue;
    }

    if (text === "}" || text === "}") continue;

    const firstWordMatch = text.match(/^([A-Za-z_]\w*)/);
    if (!firstWordMatch) {
      // Line doesn't start with a word — likely a syntax error
      // But could be continuation (string, etc.) — only warn on obvious cases
      if (!text.startsWith('"') && !text.startsWith("{")) {
        diagnostics.push({
          from: line.from,
          to: line.to,
          severity: "error",
          message: `Unexpected token: ${text.slice(0, 20)}`,
        });
      }
      continue;
    }

    const keyword = firstWordMatch[1];

    if (!VALID_LINE_STARTERS.has(keyword)) {
      // Could be an identifier in a block context (e.g., inside for/if)
      // Only warn at indent level 0
      const indent = line.text.length - line.text.trimStart().length;
      if (indent === 0) {
        diagnostics.push({
          from: line.from,
          to: line.from + keyword.length,
          severity: "warning",
          message: `Unknown declaration: ${keyword}`,
        });
      }
      continue;
    }

    if (REQUIRES_VALUE.has(keyword)) {
      const rest = text.slice(keyword.length).trim();
      if (!rest) {
        diagnostics.push({
          from: line.from,
          to: line.to,
          severity: "error",
          message: `${keyword} requires a value`,
        });
        continue;
      }
    }

    if (keyword === "CONFIG") {
      const configMatch = text.match(/^CONFIG\s+(\w+)\s*=\s*(.+)$/);
      if (!configMatch) {
        const typeMatch = text.match(/^CONFIG\s+\w+\s+(BOOL|STRING|NUMBER|SECRET|TEXT|SELECT)\b/);
        if (!typeMatch) {
          diagnostics.push({
            from: line.from,
            to: line.to,
            severity: "warning",
            message: 'CONFIG syntax: CONFIG name = value',
          });
        }
      }
    }

    if (keyword === "MODE") {
      const modeValue = text.slice(4).trim();
      if (modeValue && !VALID_MODES.has(modeValue)) {
        const valueStart = line.from + text.indexOf(modeValue);
        diagnostics.push({
          from: valueStart,
          to: valueStart + modeValue.length,
          severity: "error",
          message: `Invalid MODE value "${modeValue}". Use: pty, acp`,
        });
      }
    }

    const quoteCount = (text.match(/(?<!\\)"/g) || []).length;
    if (quoteCount % 2 !== 0 && !heredocMatch) {
      diagnostics.push({
        from: line.from,
        to: line.to,
        severity: "error",
        message: "Unclosed string literal",
      });
    }
  }

  if (inHeredoc) {
    const lastLine = doc.line(doc.lines);
    diagnostics.push({
      from: lastLine.from,
      to: lastLine.to,
      severity: "error",
      message: `Unclosed heredoc: missing ${heredocMarker}`,
    });
  }

  return diagnostics;
}
