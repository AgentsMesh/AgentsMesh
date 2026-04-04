"use client";

/**
 * AgentFile source code editor powered by CodeMirror 6.
 *
 * Features:
 * - AgentFile syntax highlighting (keywords, strings, comments, etc.)
 * - Context-aware autocomplete (keywords + data candidates per keyword)
 * - Real-time lint diagnostics (syntax errors, unknown keywords)
 */
import React, { useMemo } from "react";
import { keymap } from "@codemirror/view";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { autocompletion, closeBrackets } from "@codemirror/autocomplete";
import { linter } from "@codemirror/lint";
import {
  agentfileLanguage,
  agentfileSyntaxHighlighting,
  agentfileCompletion,
  agentfileLinter,
} from "@/lib/codemirror-agentfile";
import type { AgentfileCompletionContext } from "@/lib/codemirror-agentfile";
import { CodeMirrorEditor } from "@/lib/codemirror-agentfile/CodeMirrorEditor";
import { agentfileEditorTheme } from "./agentfileEditorTheme";

interface AgentfileCodeEditorProps {
  value: string;
  onChange: (value: string) => void;
  /** Full completion context with agents, repos, credentials, config schema */
  completionContext: AgentfileCompletionContext;
}

export function AgentfileCodeEditor({
  value,
  onChange,
  completionContext,
}: AgentfileCodeEditorProps) {
  const extensions = useMemo(() => [
    keymap.of([...defaultKeymap, ...historyKeymap]),
    history(),
    closeBrackets(),
    agentfileLanguage,
    agentfileSyntaxHighlighting,
    autocompletion({
      override: [agentfileCompletion(completionContext)],
      activateOnTyping: true,
    }),
    linter(agentfileLinter, { delay: 500 }),
    agentfileEditorTheme,
  ], [completionContext]);

  return (
    <CodeMirrorEditor
      value={value}
      onChange={onChange}
      extensions={extensions}
      className="agentfile-editor rounded-md border border-border overflow-hidden"
    />
  );
}
