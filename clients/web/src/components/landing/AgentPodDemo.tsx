"use client";

import { useState, useEffect } from "react";

interface PodTerminal {
  name: string;
  agent: string;
  workspace: string;
  lines: string[];
  color: string;
}

const PODS: PodTerminal[] = [
  {
    name: "pod-alpha",
    agent: "Claude Code",
    workspace: "/projects/api",
    color: "text-orange-400",
    lines: [
      "$ claude --resume",
      "> Resuming session...",
      "> Writing src/auth/handler.ts",
      "> Writing src/auth/oauth.ts",
      "$ go test ./internal/auth/...",
      "ok  internal/auth  0.34s",
      "> Creating merge request...",
      "> ✓ MR !41 created",
    ],
  },
  {
    name: "pod-beta",
    agent: "Codex CLI",
    workspace: "/projects/web",
    color: "text-blue-400",
    lines: [
      "$ codex start",
      "> Analyzing codebase...",
      "> Writing src/components/Auth.tsx",
      "> Writing src/hooks/useAuth.ts",
      "$ pnpm test --run",
      "✓ 8 tests passed",
      "> Pushing to feature/auth-ui",
      "> ✓ Branch pushed",
    ],
  },
  {
    name: "pod-gamma",
    agent: "Aider",
    workspace: "/projects/mobile",
    color: "text-purple-400",
    lines: [
      "$ aider --model opus",
      "> Loading repo map...",
      "> Editing lib/auth/login.dart",
      "> Editing lib/auth/token.dart",
      "$ flutter test",
      "All 14 tests passed!",
      "> Committing changes...",
      "> ✓ 2 files changed",
    ],
  },
];

function Terminal({
  pod,
  displayedLines,
}: {
  pod: PodTerminal;
  displayedLines: number;
}) {
  return (
    <div className="bg-[#0d1117] rounded-lg border border-[#30363d] overflow-hidden shadow-xl">
      <div className="flex items-center justify-between px-3 py-2 bg-[#161b22] border-b border-[#30363d]">
        <div className="flex items-center gap-1.5">
          <div className="w-2.5 h-2.5 rounded-full bg-[#f85149]" />
          <div className="w-2.5 h-2.5 rounded-full bg-[#d29922]" />
          <div className="w-2.5 h-2.5 rounded-full bg-[#3fb950]" />
        </div>
        <span className="text-[10px] font-mono text-[#8b949e]">{pod.name}</span>
        <div className="flex items-center gap-1">
          <span className="w-1.5 h-1.5 rounded-full bg-[#3fb950] animate-pulse" />
        </div>
      </div>

      <div className="px-3 py-1 bg-[#0d1117] border-b border-[#21262d] text-[10px] font-mono text-[#8b949e] flex gap-3">
        <span className={pod.color}>{pod.agent}</span>
        <span className="text-[#484f58]">|</span>
        <span>{pod.workspace}</span>
      </div>

      <div className="p-3 font-mono text-[11px] leading-[1.6] h-[140px] overflow-hidden">
        {pod.lines.slice(0, displayedLines).map((line, i) => (
          <div
            key={i}
            className={
              line.startsWith("$")
                ? "text-[#58a6ff]"
                : line.startsWith(">")
                  ? line.includes("✓")
                    ? "text-[#3fb950]"
                    : "text-[#d2a8ff]"
                  : line.includes("passed") || line.includes("ok ")
                    ? "text-[#3fb950]"
                    : "text-[#c9d1d9]"
            }
          >
            {line}
          </div>
        ))}
        {displayedLines < pod.lines.length && (
          <span className="text-[#58a6ff] animate-pulse">▋</span>
        )}
      </div>
    </div>
  );
}

export function AgentPodDemo() {
  const [displayedLines, setDisplayedLines] = useState(0);
  const maxLines = Math.max(...PODS.map((p) => p.lines.length));

  useEffect(() => {
    if (displayedLines < maxLines) {
      const timer = setTimeout(() => {
        setDisplayedLines((prev) => prev + 1);
      }, 600);
      return () => clearTimeout(timer);
    } else {
      const timer = setTimeout(() => {
        setDisplayedLines(0);
      }, 3000);
      return () => clearTimeout(timer);
    }
  }, [displayedLines, maxLines]);

  return (
    <div className="space-y-2.5">
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
        <Terminal pod={PODS[0]} displayedLines={displayedLines} />
        <Terminal pod={PODS[1]} displayedLines={displayedLines} />
      </div>
      <Terminal pod={PODS[2]} displayedLines={displayedLines} />
    </div>
  );
}
