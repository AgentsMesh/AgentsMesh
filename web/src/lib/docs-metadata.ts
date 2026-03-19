import type { Metadata } from "next";

interface DocsMeta {
  title: string;
  description: string;
}

const docsMetadataMap: Record<string, DocsMeta> = {
  "/docs": {
    title: "Documentation",
    description:
      "Complete documentation for AgentsMesh — the AI agent fleet command center for orchestrating multi-agent collaboration.",
  },
  "/docs/getting-started": {
    title: "Quick Start",
    description:
      "Get up and running with AgentsMesh in minutes. Set up your Runner, create your first Pod, and start collaborating with AI agents.",
  },
  "/docs/concepts": {
    title: "Core Concepts",
    description:
      "Understand the key concepts behind AgentsMesh — Pods, Runners, Channels, Mesh topology, and how they work together.",
  },
  "/docs/faq": {
    title: "FAQ",
    description:
      "Frequently asked questions about AgentsMesh — troubleshooting Runners, Pods, API keys, Git integration, and billing.",
  },
  "/docs/features/agentpod": {
    title: "AgentPod",
    description:
      "Learn about AgentPod — isolated execution environments with PTY terminals for running AI coding agents securely.",
  },
  "/docs/features/channels": {
    title: "Channels",
    description:
      "Multi-agent collaboration spaces where AI agents communicate, share context, and coordinate work in real time.",
  },
  "/docs/features/loops": {
    title: "Loops",
    description:
      "Automated feedback loops for iterative agent-driven development — define triggers, conditions, and actions.",
  },
  "/docs/features/mesh": {
    title: "Mesh Topology",
    description:
      "Visualize and manage agent relationships, task dependencies, and collaboration patterns across your organization.",
  },
  "/docs/features/repositories": {
    title: "Repositories",
    description:
      "Connect Git providers (GitHub, GitLab) and manage repository access for your AI agents with OAuth integration.",
  },
  "/docs/features/tickets": {
    title: "Tickets",
    description:
      "Kanban-style task management integrated with AI agent workflows — create, assign, and track development tasks.",
  },
  "/docs/features/workspace": {
    title: "Workspace",
    description:
      "Git worktree-based workspace isolation ensuring each agent operates on its own branch without conflicts.",
  },
  "/docs/runners/setup": {
    title: "Runner Setup",
    description:
      "Install and configure the AgentsMesh Runner daemon — self-hosted agent execution with gRPC and mTLS security.",
  },
  "/docs/runners/mcp-tools": {
    title: "MCP Tools",
    description:
      "Model Context Protocol integration for Runners — extend agent capabilities with custom tools and resources.",
  },
  "/docs/guides/git-integration": {
    title: "Git Integration",
    description:
      "Set up Git provider connections, manage SSH keys, and configure repository access for your AI agent workflows.",
  },
  "/docs/guides/multi-agent-workflows": {
    title: "Multi-Agent Workflows",
    description:
      "Design and run multi-agent collaboration workflows — parallel development, code review, and coordinated shipping.",
  },
  "/docs/guides/team-management": {
    title: "Team Management",
    description:
      "Manage teams, roles, and permissions in AgentsMesh — organize your AI agent fleet for maximum productivity.",
  },
  "/docs/api": {
    title: "API Overview",
    description:
      "AgentsMesh REST API reference — authenticate, manage Pods, Tickets, Channels, and more programmatically.",
  },
  "/docs/api/authentication": {
    title: "API Authentication",
    description:
      "Authenticate with the AgentsMesh API using JWT tokens and OAuth — secure access to all platform endpoints.",
  },
  "/docs/api/channels": {
    title: "Channels API",
    description:
      "Create, list, and manage multi-agent collaboration channels via the AgentsMesh REST API.",
  },
  "/docs/api/loops": {
    title: "Loops API",
    description:
      "Manage automated feedback loops programmatically — create triggers, monitor executions, and retrieve results.",
  },
  "/docs/api/pods": {
    title: "Pods API",
    description:
      "Create, monitor, and terminate AgentPods via the REST API — full lifecycle management for agent execution.",
  },
  "/docs/api/repositories": {
    title: "Repositories API",
    description:
      "Manage Git repository connections and access tokens via the AgentsMesh REST API.",
  },
  "/docs/api/runners": {
    title: "Runners API",
    description:
      "Register, monitor, and manage Runner daemons via the REST API — health checks, certificates, and configuration.",
  },
  "/docs/api/tickets": {
    title: "Tickets API",
    description:
      "Create, update, and query development tickets via the AgentsMesh REST API — integrate with your workflow.",
  },
};

const defaultMeta: DocsMeta = {
  title: "Documentation",
  description:
    "AgentsMesh documentation — orchestrate AI coding agents at scale.",
};

/**
 * Create Next.js Metadata for a docs page path.
 * Used by individual docs sub-page layout.tsx files.
 */
export function createDocsMetadata(path: string): Metadata {
  const meta = docsMetadataMap[path] ?? defaultMeta;
  return {
    title: meta.title,
    description: meta.description,
    alternates: {
      canonical: `https://agentsmesh.ai${path}`,
    },
    openGraph: {
      title: `${meta.title} | AgentsMesh Docs`,
      description: meta.description,
      url: `https://agentsmesh.ai${path}`,
    },
  };
}
