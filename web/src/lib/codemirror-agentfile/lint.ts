/**
 * AgentFile lint extension for CodeMirror.
 *
 * Provides real-time syntax error highlighting by validating each line
 * against known AgentFile declaration patterns.
 */
import type { Diagnostic } from "@codemirror/lint";
import type { EditorView } from "@codemirror/view";

/** Valid declaration keywords that start a line */
const VALID_LINE_STARTERS = new Set([
  "AGENT", "EXECUTABLE", "CONFIG", "ENV", "REPO", "BRANCH",
  "GIT_CREDENTIAL", "MCP", "SKILLS", "SETUP",
  "REMOVE", "MODE", "CREDENTIAL",
  "PROMPT", "PROMPT_POSITION",
  // Build logic
  "arg", "file", "mkdir",
  "when", "if", "else", "for",
]);

/** Declarations requiring a value after keyword */
const REQUIRES_VALUE = new Set([
  "AGENT", "EXECUTABLE", "REPO", "BRANCH", "GIT_CREDENTIAL",
  "MODE", "CREDENTIAL", "PROMPT", "PROMPT_POSITION",
]);

/** Valid MODE values */
const VALID_MODES = new Set(["pty", "acp"]);

/**
 * Lint source for AgentFile.
 * Checks each non-empty, non-comment line for basic syntax errors.
 */
export function agentfileLinter(view: EditorView): Diagnostic[] {
  const diagnostics: Diagnostic[] = [];
  const doc = view.state.doc;

  let inHeredoc = false;
  let heredocMarker = "";

  for (let i = 1; i <= doc.lines; i++) {
    const line = doc.line(i);
    const text = line.text.trim();

    // Skip empty lines and comments
    if (!text || text.startsWith("#")) continue;

    // Handle heredoc body
    if (inHeredoc) {
      if (text === heredocMarker) {
        inHeredoc = false;
        heredocMarker = "";
      }
      continue;
    }

    // Check for heredoc start on this line
    const heredocMatch = text.match(/<<([A-Z_]+)\s*$/);
    if (heredocMatch) {
      inHeredoc = true;
      heredocMarker = heredocMatch[1];
      continue;
    }

    // Closing braces are valid (block end)
    if (text === "}" || text === "}") continue;

    // Extract first word
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

    // Check if first word is a valid line starter
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

    // Validate declarations that require a value
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

    // Validate CONFIG syntax: CONFIG name = value
    if (keyword === "CONFIG") {
      const configMatch = text.match(/^CONFIG\s+(\w+)\s*=\s*(.+)$/);
      if (!configMatch) {
        // Check if it's a CONFIG type declaration (e.g., CONFIG name STRING "label" { ... })
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

    // Validate MODE value
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

    // Validate unclosed strings
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

  // Unclosed heredoc
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
