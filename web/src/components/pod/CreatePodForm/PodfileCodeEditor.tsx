"use client";

/**
 * PodFile source code editor powered by CodeMirror 6.
 *
 * Features:
 * - PodFile syntax highlighting (keywords, strings, comments, etc.)
 * - Context-aware autocomplete (keywords + data candidates per keyword)
 * - Real-time lint diagnostics (syntax errors, unknown keywords)
 */
import React, { useMemo } from "react";
import { keymap } from "@codemirror/view";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { autocompletion, closeBrackets } from "@codemirror/autocomplete";
import { linter } from "@codemirror/lint";
import {
  podfileLanguage,
  podfileSyntaxHighlighting,
  podfileCompletion,
  podfileLinter,
} from "@/lib/codemirror-podfile";
import type { PodfileCompletionContext } from "@/lib/codemirror-podfile";
import { CodeMirrorEditor } from "@/lib/codemirror-podfile/CodeMirrorEditor";
import { podfileEditorTheme } from "./podfileEditorTheme";

interface PodfileCodeEditorProps {
  value: string;
  onChange: (value: string) => void;
  /** Full completion context with agents, repos, credentials, config schema */
  completionContext: PodfileCompletionContext;
}

export function PodfileCodeEditor({
  value,
  onChange,
  completionContext,
}: PodfileCodeEditorProps) {
  const extensions = useMemo(() => [
    keymap.of([...defaultKeymap, ...historyKeymap]),
    history(),
    closeBrackets(),
    podfileLanguage,
    podfileSyntaxHighlighting,
    autocompletion({
      override: [podfileCompletion(completionContext)],
      activateOnTyping: true,
    }),
    linter(podfileLinter, { delay: 500 }),
    podfileEditorTheme,
  ], [completionContext]);

  return (
    <CodeMirrorEditor
      value={value}
      onChange={onChange}
      extensions={extensions}
      className="podfile-editor rounded-md border border-border overflow-hidden"
    />
  );
}
