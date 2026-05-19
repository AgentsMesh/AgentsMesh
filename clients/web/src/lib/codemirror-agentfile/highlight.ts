/**
 * AgentFile syntax highlighting via CodeMirror StreamLanguage.
 *
 * Tokenizes AgentFile DSL for syntax coloring:
 * - Declaration keywords (AGENT, CONFIG, ENV, etc.) → keyword
 * - Build keywords (arg, file, if, for, etc.) → keyword
 * - Strings → string
 * - Numbers → number
 * - Booleans → bool
 * - Comments → comment
 * - Operators → operator
 */
import { StreamLanguage, type StringStream, HighlightStyle, syntaxHighlighting } from "@codemirror/language";
import { tags } from "@lezer/highlight";

const DECL_KEYWORDS = new Set([
  "AGENT", "EXECUTABLE", "CONFIG", "ENV", "REPO", "BRANCH",
  "GIT_CREDENTIAL", "MCP", "SKILLS", "SETUP", "ON", "OFF",
  "OPTIONAL", "REMOVE", "MODE", "CREDENTIAL",
  "PROMPT", "PROMPT_POSITION",
  "BOOL", "STRING", "NUMBER", "SECRET", "TEXT", "SELECT",
]);

const BUILD_KEYWORDS = new Set([
  "arg", "file", "mkdir",
  "when", "if", "else", "for", "in", "and", "or", "not",
  "prepend", "append", "none",
]);

const BOOLEANS = new Set(["true", "false"]);

interface AgentFileState {
  inHeredoc: boolean;
  heredocMarker: string;
}

const agentfileTokenizer = {
  startState(): AgentFileState {
    return { inHeredoc: false, heredocMarker: "" };
  },

  token(stream: StringStream, state: AgentFileState): string | null {
    if (state.inHeredoc) {
      if (stream.sol() && stream.match(new RegExp(`^${state.heredocMarker}\\s*$`))) {
        state.inHeredoc = false;
        state.heredocMarker = "";
        return "string";
      }
      stream.skipToEnd();
      return "string";
    }

    if (stream.eatSpace()) return null;

    if (stream.match("#")) {
      stream.skipToEnd();
      return "comment";
    }

    if (stream.match(/^<<([A-Z_]+)/)) {
      state.inHeredoc = true;
      state.heredocMarker = stream.current().slice(2);
      return "string";
    }

    if (stream.match('"')) {
      while (!stream.eol()) {
        const ch = stream.next();
        if (ch === "\\") { stream.next(); continue; }
        if (ch === '"') return "string";
      }
      return "string";
    }

    if (stream.match(/^-?\d+(\.\d+)?/)) return "number";

    if (stream.match("==") || stream.match("!=")) return "operator";
    if (stream.match("=") || stream.match("+")) return "operator";

    if (stream.match(/^[(){}[\]:,.]/)) return "punctuation";

    if (stream.match(/^[a-zA-Z_][a-zA-Z0-9_]*/)) {
      const word = stream.current();
      if (DECL_KEYWORDS.has(word)) return "keyword";
      if (BUILD_KEYWORDS.has(word)) return "keyword";
      if (BOOLEANS.has(word)) return "bool";
      return "variableName";
    }

    stream.next();
    return null;
  },

  languageData: {
    commentTokens: { line: "#" },
  },
};

export const agentfileLanguage = StreamLanguage.define(agentfileTokenizer);

const agentfileHighlight = HighlightStyle.define([
  { tag: tags.keyword, color: "#7c3aed", fontWeight: "600" },      // purple
  { tag: tags.string, color: "#16a34a" },                           // green
  { tag: tags.number, color: "#ea580c" },                           // orange
  { tag: tags.bool, color: "#9333ea", fontWeight: "600" },          // violet
  { tag: tags.comment, color: "#9ca3af", fontStyle: "italic" },     // gray
  { tag: tags.operator, color: "#64748b" },                         // slate
  { tag: tags.variableName, color: "#0284c7" },                     // sky blue
  { tag: tags.punctuation, color: "#94a3b8" },                      // light slate
]);

export const agentfileSyntaxHighlighting = syntaxHighlighting(agentfileHighlight);
