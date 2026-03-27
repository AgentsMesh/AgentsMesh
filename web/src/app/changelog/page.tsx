"use client";

import { PageHeader, PageFooter } from "@/components/common";

interface ChangelogEntry {
  version: string;
  date: string;
  changes: {
    type: "added" | "changed" | "fixed" | "removed";
    items: string[];
  }[];
}

const changelog: ChangelogEntry[] = [
  {
    version: "0.9.1",
    date: "2026-03-07",
    changes: [
      {
        type: "fixed",
        items: [
          "gRPC TLS ServerName correctly set to prevent advancedtls SNI port bug",
        ],
      },
      {
        type: "changed",
        items: [
          "Channel UI/UX refactoring with docs i18n sync",
        ],
      },
    ],
  },
  {
    version: "0.9.0",
    date: "2026-03-07",
    changes: [
      {
        type: "added",
        items: [
          "Runner cross-platform abstraction for Windows support (ConPTY, path handling)",
          "Public relay URL for scalable multi-relay routing",
          "Runner endpoint auto-discovery, PID file, and login shell PATH injection",
          "HeartbeatAck and recv watchdog for half-dead connection detection",
        ],
      },
      {
        type: "fixed",
        items: [
          "Terminal content duplication on page refresh",
          "4 bugs in Runner PATH injection and config update code",
          "Relay flap detection to prevent 500ms reconnect storm",
          "Deep architecture issues in Zustand stores and React rendering",
          "Relay concurrency issues causing heartbeat deadlock and 503 errors",
        ],
      },
    ],
  },
  {
    version: "0.8.0",
    date: "2026-03-04",
    changes: [
      {
        type: "added",
        items: [
          "MCP post_comment tool for ticket collaboration",
          "Runner relay URL fallback for local development",
        ],
      },
      {
        type: "fixed",
        items: [
          "Runner MCP server bound to localhost with macOS code signing",
          "PATH fallback for agent detection in Runner service mode",
          "Self-update download failure due to missing v-prefix",
          "Runner CLI binary name in help and error messages",
          "gRPC chain-only TLS verification with dynamic server cert SANs",
          "Redundant pod capacity check removed from Runner side",
        ],
      },
    ],
  },
  {
    version: "0.7.0",
    date: "2026-03-04",
    changes: [
      {
        type: "added",
        items: [
          "Support ticket system with admin interface",
          "Blog content migrated from JSON to Markdown files",
        ],
      },
      {
        type: "changed",
        items: [
          "Runner release migrated from standalone repo to main monorepo",
          "Runner GoReleaser config modernized, deprecation warnings resolved",
        ],
      },
    ],
  },
  {
    version: "0.6.0",
    date: "2026-03-03",
    changes: [
      {
        type: "added",
        items: [
          "Enterprise page and inquiry flow",
          "Help & feedback button in ActivityBar",
          "Server-side filtering and pagination for pod list",
          "Admin email verification action",
        ],
      },
      {
        type: "fixed",
        items: [
          "gRPC downstream silent failure for runner connections",
          "LemonSqueezy seat billing and subscription edge cases",
          "OAuth empty email linking to unrelated users",
        ],
      },
      {
        type: "changed",
        items: [
          "Runner install directory changed to ~/.local/bin",
        ],
      },
    ],
  },
  {
    version: "0.5.0",
    date: "2026-02-27",
    changes: [
      {
        type: "added",
        items: [
          "Skills & MCP Server capabilities system for extensibility",
          "Agent version detection, heartbeat reporting, and server-side adaptation",
          "Dual clone URL support (HTTP + SSH) for repository imports",
          "Image paste support with clipboard shim and native backends",
        ],
      },
      {
        type: "fixed",
        items: [
          "Runner stops reconnecting on fatal auth errors with actionable hints",
          "Cross-org runner registration causing connection failures",
          "Channel message deduplication to prevent double display",
        ],
      },
    ],
  },
  {
    version: "0.4.0",
    date: "2026-02-22",
    changes: [
      {
        type: "added",
        items: [
          "Admin Console subscription management",
          "Runner three-layer reliability defense system",
          "Runner enhanced logging with daily rotation and directory size limit",
          "Headless login mode with default server for Runner",
          "Follow Runner model option for Claude Code pods",
          "MCP tool results optimized from JSON to Markdown format",
        ],
      },
      {
        type: "changed",
        items: [
          "Unified ticket identifier to slug naming across the stack",
          "Standardized API error responses with structured error codes",
        ],
      },
      {
        type: "fixed",
        items: [
          "Terminal blank after reconnect due to stale relay connection",
          "Relay reconnect race condition when multiple clients connect to same pod",
        ],
      },
    ],
  },
  {
    version: "0.3.0",
    date: "2026-02-07",
    changes: [
      {
        type: "added",
        items: [
          "Runner relay connection status reporting and display",
          "Auto-reconnect for Runner-Relay WebSocket connection with token refresh",
          "Terminal state restoration after server restart",
          "Bandwidth-aware full redraw throttling for PTY output",
          "Priority-based dual-channel architecture for Runner",
          "Auto-update system with graceful updater reliability",
          "Structured logging migration to slog",
          "PR/MR status awareness mechanism",
          "iOS PWA notification support with OSC terminal notifications",
          "Sandbox status query and resume support",
        ],
      },
      {
        type: "fixed",
        items: [
          "Terminal flickering with Synchronized Output frame detection",
          "Terminal output loss with backpressure mechanism",
          "Terminal size synchronization issues",
        ],
      },
      {
        type: "changed",
        items: [
          "Relay routing switched from session_id to podKey",
          "Runner binary renamed to agentsmesh-runner",
        ],
      },
    ],
  },
  {
    version: "0.2.0",
    date: "2026-01-16",
    changes: [
      {
        type: "added",
        items: [
          "Runner migrated from WebSocket to gRPC + mTLS communication",
          "MCP tool: create_pod with agent_slug parameter",
          "macOS universal binary support in GoReleaser",
          "Automated release pipeline for all platforms",
          "Windows ConPTY support for terminal",
        ],
      },
      {
        type: "changed",
        items: [
          "Unified brand name to AgentsMesh across codebase",
          "Agent config system refactored: removed Lua plugins, added backend-driven config",
          "WebSocket compression enabled for better performance",
        ],
      },
      {
        type: "fixed",
        items: [
          "Race conditions in terminal and token refresh tests",
          "Post-quantum TLS and WebSocket connection issues",
          "Pod OSC title parsing accuracy",
        ],
      },
    ],
  },
  {
    version: "0.1.0",
    date: "2026-01-11",
    changes: [
      {
        type: "added",
        items: [
          "Initial release of AgentsMesh platform",
          "AgentPod: remote AI coding workstation with PTY terminal",
          "Support for Claude Code, Codex CLI, Gemini CLI, Aider",
          "Self-hosted Runner deployment with mTLS certificate management",
          "Git repository integration (GitHub, GitLab, Gitee)",
          "Web terminal with real-time bidirectional interaction",
          "Organization and team management with multi-tenancy",
          "AgentsMesh multi-agent collaboration channels",
          "Ticket management with kanban board",
          "Multi-language support (8 languages)",
        ],
      },
    ],
  },
];

const typeLabels = {
  added: { label: "Added", color: "bg-green-500/20 text-green-600 dark:text-green-400" },
  changed: { label: "Changed", color: "bg-blue-500/20 text-blue-600 dark:text-blue-400" },
  fixed: { label: "Fixed", color: "bg-yellow-500/20 text-yellow-600 dark:text-yellow-400" },
  removed: { label: "Removed", color: "bg-red-500/20 text-red-600 dark:text-red-400" },
};

export default function ChangelogPage() {
  return (
    <div className="min-h-screen bg-background">
      <PageHeader />

      {/* Content */}
      <main className="container mx-auto px-4 py-12 max-w-4xl">
        <h1 className="text-4xl font-bold mb-4">Changelog</h1>
        <p className="text-muted-foreground mb-12">
          All notable changes to AgentsMesh will be documented here.
        </p>

        <div className="space-y-12">
          {changelog.map((entry) => (
            <article key={entry.version} className="relative">
              {/* Version header */}
              <div className="flex items-center gap-4 mb-6">
                <h2 className="text-2xl font-bold">v{entry.version}</h2>
                <time className="text-sm text-muted-foreground">
                  {new Date(entry.date).toLocaleDateString("en-US", {
                    year: "numeric",
                    month: "long",
                    day: "numeric",
                  })}
                </time>
              </div>

              {/* Changes */}
              <div className="space-y-6 pl-4 border-l-2 border-border">
                {entry.changes.map((change, idx) => (
                  <div key={idx}>
                    <span
                      className={`inline-block px-2 py-1 rounded text-xs font-medium mb-3 ${typeLabels[change.type].color}`}
                    >
                      {typeLabels[change.type].label}
                    </span>
                    <ul className="space-y-2">
                      {change.items.map((item, itemIdx) => (
                        <li
                          key={itemIdx}
                          className="text-muted-foreground flex items-start gap-2"
                        >
                          <span className="text-primary mt-1.5">•</span>
                          {item}
                        </li>
                      ))}
                    </ul>
                  </div>
                ))}
              </div>
            </article>
          ))}
        </div>
      </main>

      <PageFooter />
    </div>
  );
}
