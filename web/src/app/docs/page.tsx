import Link from "next/link";

export default function DocsPage() {
  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">Documentation</h1>

      {/* Introduction */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Introduction</h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          AgentMesh is a{" "}
          <strong className="text-foreground">
            Terminal-based Coding Agent collaboration platform
          </strong>{" "}
          that enables teams to leverage AI coding assistants like Claude Code,
          Codex CLI, Gemini CLI, and Aider at scale. Unlike IDE plugins (Cursor,
          Copilot), AgentMesh focuses on autonomous terminal agents with full
          system access, enabling multi-agent coordination (DevMesh), remote AI
          workstations (DevPod), and integrated task management.
        </p>
        <div className="bg-muted rounded-lg p-4 mt-4">
          <p className="font-medium mb-2">Supported AI Agents:</p>
          <ul className="list-disc list-inside text-muted-foreground space-y-1">
            <li>Claude Code (Anthropic)</li>
            <li>Codex CLI (OpenAI)</li>
            <li>Gemini CLI (Google)</li>
            <li>Aider</li>
            <li>OpenCode</li>
            <li>Custom agents via MCP protocol</li>
          </ul>
        </div>
      </section>

      {/* Why Terminal-based Agents */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          Why Terminal-based Agents?
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          Terminal-based coding agents offer significant advantages over IDE
          plugins:
        </p>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">
                  Feature
                </th>
                <th className="text-left p-3 border-b border-border">
                  IDE Plugins (Copilot/Cursor)
                </th>
                <th className="text-left p-3 border-b border-border">
                  Terminal Agents (Claude Code/Aider)
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border">Autonomy</td>
                <td className="p-3 border-b border-border">
                  Passive, requires triggers
                </td>
                <td className="p-3 border-b border-border text-green-400">
                  ✓ Active execution of complex tasks
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">Capabilities</td>
                <td className="p-3 border-b border-border">Code completion</td>
                <td className="p-3 border-b border-border text-green-400">
                  ✓ Full-stack dev, refactoring, testing
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">Environment</td>
                <td className="p-3 border-b border-border">IDE sandbox</td>
                <td className="p-3 border-b border-border text-green-400">
                  ✓ Full terminal access
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  Multi-Agent Collaboration
                </td>
                <td className="p-3 border-b border-border">✗</td>
                <td className="p-3 border-b border-border text-green-400">
                  ✓ AgentMesh supported
                </td>
              </tr>
              <tr>
                <td className="p-3">Self-hosted</td>
                <td className="p-3">✗</td>
                <td className="p-3 text-green-400">✓ Fully controllable</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Quick Links */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Quick Links</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Link
            href="/docs/getting-started"
            className="border border-border rounded-lg p-4 hover:border-primary transition-colors"
          >
            <h3 className="font-medium mb-1">Quick Start →</h3>
            <p className="text-sm text-muted-foreground">
              Get up and running in 5 minutes
            </p>
          </Link>
          <Link
            href="/docs/features/devpod"
            className="border border-border rounded-lg p-4 hover:border-primary transition-colors"
          >
            <h3 className="font-medium mb-1">DevPod →</h3>
            <p className="text-sm text-muted-foreground">
              Remote AI development workstations
            </p>
          </Link>
          <Link
            href="/docs/features/devmesh"
            className="border border-border rounded-lg p-4 hover:border-primary transition-colors"
          >
            <h3 className="font-medium mb-1">DevMesh →</h3>
            <p className="text-sm text-muted-foreground">
              Multi-agent collaboration network
            </p>
          </Link>
          <Link
            href="/docs/runners/mcp-tools"
            className="border border-border rounded-lg p-4 hover:border-primary transition-colors"
          >
            <h3 className="font-medium mb-1">MCP Tools →</h3>
            <p className="text-sm text-muted-foreground">
              25+ tools for AI agents
            </p>
          </Link>
        </div>
      </section>
    </div>
  );
}
