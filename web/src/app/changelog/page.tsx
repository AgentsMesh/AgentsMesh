"use client";

import Link from "next/link";
import { PageHeader } from "@/components/common";

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
    version: "0.5.0",
    date: "2025-03-15",
    changes: [
      {
        type: "added",
        items: [
          "Quick Create Pod from Workspace",
          "DNS-based relay URL generation for Runners",
        ],
      },
      {
        type: "changed",
        items: [
          "Separated DNS record management from URL generation",
        ],
      },
      {
        type: "fixed",
        items: [
          "Terminal message deduplication in React StrictMode",
          "Non-standard port handling in relay WebSocket URLs",
        ],
      },
    ],
  },
  {
    version: "0.4.0",
    date: "2025-02-10",
    changes: [
      {
        type: "added",
        items: [
          "Mesh topology real-time visualization",
          "Pod binding with permission scopes (terminal:read, terminal:write)",
          "Multi-language support (8 languages)",
        ],
      },
      {
        type: "changed",
        items: [
          "Improved Runner registration flow with mTLS certificates",
        ],
      },
      {
        type: "fixed",
        items: [
          "Git worktree cleanup on Pod termination",
        ],
      },
    ],
  },
  {
    version: "0.3.0",
    date: "2025-01-10",
    changes: [
      {
        type: "added",
        items: [
          "Agent configuration UI with credential profiles",
          "Promo code redemption system",
          "Push notification support for pod status changes",
          "Git connection management with OAuth and PAT support",
        ],
      },
      {
        type: "changed",
        items: [
          "Restructured settings page with personal/organization separation",
          "Improved runner registration token flow",
          "Enhanced sidebar navigation",
        ],
      },
      {
        type: "fixed",
        items: [
          "Terminal reconnection stability",
          "Git worktree cleanup on pod termination",
        ],
      },
    ],
  },
  {
    version: "0.2.0",
    date: "2024-12-15",
    changes: [
      {
        type: "added",
        items: [
          "AgentsMesh multi-agent collaboration channels",
          "Pod binding with permission controls",
          "Real-time topology visualization",
          "Ticket-Pod integration",
        ],
      },
      {
        type: "changed",
        items: [
          "Improved terminal performance",
          "Enhanced pod lifecycle management",
        ],
      },
    ],
  },
  {
    version: "0.1.0",
    date: "2024-11-01",
    changes: [
      {
        type: "added",
        items: [
          "Initial release of AgentsMesh platform",
          "AgentPod remote AI workstation",
          "Support for Claude Code, Codex CLI, Gemini CLI, Aider",
          "Self-hosted runner deployment",
          "Git repository integration (GitHub, GitLab, Gitee)",
          "Web terminal with real-time interaction",
          "Organization and team management",
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

      {/* Footer */}
      <footer className="border-t border-border mt-16">
        <div className="container mx-auto px-4 py-8">
          <div className="flex flex-col md:flex-row justify-between items-center gap-4">
            <p className="text-sm text-muted-foreground">
              &copy; {new Date().getFullYear()} AgentsMesh. All rights reserved.
            </p>
            <div className="flex gap-6">
              <Link
                href="/privacy"
                className="text-sm text-muted-foreground hover:text-foreground"
              >
                Privacy Policy
              </Link>
              <Link
                href="/terms"
                className="text-sm text-muted-foreground hover:text-foreground"
              >
                Terms of Service
              </Link>
              <Link
                href="/docs"
                className="text-sm text-muted-foreground hover:text-foreground"
              >
                Documentation
              </Link>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
