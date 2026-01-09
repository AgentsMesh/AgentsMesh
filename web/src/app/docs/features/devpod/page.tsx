export default function DevPodPage() {
  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">DevPod</h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        DevPod provides remote AI development workstations with integrated
        terminal access. Each session runs on a Runner and provides a full
        development environment with AI assistance.
      </p>

      {/* Overview */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Overview</h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          A DevPod session is an isolated development environment where an AI
          coding agent (like Claude Code) runs with full terminal access. Each
          session includes:
        </p>
        <ul className="list-disc list-inside text-muted-foreground space-y-2 mb-4">
          <li>
            <strong>Web Terminal</strong> - Real-time terminal access via
            WebSocket
          </li>
          <li>
            <strong>Git Worktree</strong> - Isolated workspace per session
          </li>
          <li>
            <strong>AI Agent</strong> - Your chosen coding assistant
          </li>
          <li>
            <strong>MCP Tools</strong> - 25+ tools for file operations,
            collaboration, and more
          </li>
        </ul>
      </section>

      {/* Features */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Key Features</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">🖥️ Web-based Terminal</h3>
            <p className="text-sm text-muted-foreground">
              Full terminal emulation in your browser with WebSocket support for
              real-time interaction. No local setup required.
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">📊 Agent Status Monitoring</h3>
            <p className="text-sm text-muted-foreground">
              Real-time visibility into what your AI agent is doing: idle,
              working, waiting for input, or finished.
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">🌿 Git Worktree Isolation</h3>
            <p className="text-sm text-muted-foreground">
              Each session gets its own Git worktree, ensuring complete
              isolation between concurrent sessions.
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">🔄 Multiple Sessions</h3>
            <p className="text-sm text-muted-foreground">
              Run multiple AI agent sessions concurrently, each working on
              different tasks or branches.
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">🎫 Ticket Integration</h3>
            <p className="text-sm text-muted-foreground">
              Link sessions to tickets for automatic context and progress
              tracking.
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">🧹 Automatic Cleanup</h3>
            <p className="text-sm text-muted-foreground">
              Sessions are automatically cleaned up when terminated, freeing
              resources.
            </p>
          </div>
        </div>
      </section>

      {/* Session Lifecycle */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Session Lifecycle</h2>
        <div className="space-y-4">
          <div className="flex items-start gap-4">
            <div className="w-24 text-sm font-medium text-primary">
              Initializing
            </div>
            <p className="text-muted-foreground">
              Session is being created, worktree is being set up, and agent is
              starting.
            </p>
          </div>
          <div className="flex items-start gap-4">
            <div className="w-24 text-sm font-medium text-green-400">
              Running
            </div>
            <p className="text-muted-foreground">
              Session is active and ready for interaction.
            </p>
          </div>
          <div className="flex items-start gap-4">
            <div className="w-24 text-sm font-medium text-yellow-400">
              Paused
            </div>
            <p className="text-muted-foreground">
              Session is temporarily paused (agent not active).
            </p>
          </div>
          <div className="flex items-start gap-4">
            <div className="w-24 text-sm font-medium text-orange-400">
              Disconnected
            </div>
            <p className="text-muted-foreground">
              Runner lost connection but session may recover.
            </p>
          </div>
          <div className="flex items-start gap-4">
            <div className="w-24 text-sm font-medium text-muted-foreground">
              Completed
            </div>
            <p className="text-muted-foreground">
              Session finished successfully.
            </p>
          </div>
          <div className="flex items-start gap-4">
            <div className="w-24 text-sm font-medium text-red-400">
              Terminated
            </div>
            <p className="text-muted-foreground">
              Session was manually stopped or encountered an error.
            </p>
          </div>
        </div>
      </section>

      {/* Agent Status */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Agent Status</h2>
        <p className="text-muted-foreground mb-4">
          Within a running session, the AI agent has its own status:
        </p>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">Status</th>
                <th className="text-left p-3 border-b border-border">
                  Description
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border font-medium">Idle</td>
                <td className="p-3 border-b border-border">
                  Agent is waiting for a task or prompt
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  Working
                </td>
                <td className="p-3 border-b border-border">
                  Agent is actively executing a task
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  Waiting
                </td>
                <td className="p-3 border-b border-border">
                  Agent is waiting for user input or approval
                </td>
              </tr>
              <tr>
                <td className="p-3 font-medium">Finished</td>
                <td className="p-3">Agent has completed its task</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Configuration */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Session Configuration</h2>
        <p className="text-muted-foreground mb-4">
          When creating a session, you can configure:
        </p>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">Option</th>
                <th className="text-left p-3 border-b border-border">
                  Description
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  Agent Type
                </td>
                <td className="p-3 border-b border-border">
                  Claude Code, Codex CLI, Gemini CLI, Aider, or custom
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  Model
                </td>
                <td className="p-3 border-b border-border">
                  opus, sonnet, or haiku (for Claude)
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  Permission Mode
                </td>
                <td className="p-3 border-b border-border">
                  plan (ask before changes), default, or bypassPermissions
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  Repository
                </td>
                <td className="p-3 border-b border-border">
                  Link to a Git repository for automatic cloning
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  Ticket
                </td>
                <td className="p-3 border-b border-border">
                  Link to a ticket for context and tracking
                </td>
              </tr>
              <tr>
                <td className="p-3 font-medium">Initial Prompt</td>
                <td className="p-3">
                  Starting instructions for the AI agent
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>
  );
}
