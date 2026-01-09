export default function RestAPIPage() {
  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">REST API</h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        AgentMesh provides a comprehensive REST API for integration with your
        tools and workflows.
      </p>

      {/* Base URL */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Base URL</h2>
        <div className="bg-[#1a1a1a] rounded-lg p-4 font-mono text-sm">
          <span className="text-green-400">https://api.agentmesh.dev/api/v1</span>
        </div>
        <p className="text-sm text-muted-foreground mt-4">
          For self-hosted installations, replace with your server URL.
        </p>
      </section>

      {/* Authentication */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Authentication</h2>
        <p className="text-muted-foreground mb-4">
          All API requests require authentication via Bearer token:
        </p>
        <div className="bg-[#1a1a1a] rounded-lg p-4 font-mono text-sm overflow-x-auto">
          <pre className="text-green-400">{`Authorization: Bearer <your-token>
X-Organization-Slug: <org-slug>`}</pre>
        </div>
        <p className="text-sm text-muted-foreground mt-4">
          Get your API token from <strong>Settings → API Keys</strong> in the
          web interface.
        </p>
      </section>

      {/* Sessions API */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Sessions</h2>

        <div className="space-y-6">
          <div className="border border-border rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="px-2 py-1 bg-green-500/20 text-green-400 rounded text-xs font-mono">
                GET
              </span>
              <code className="text-sm">/organizations/:slug/sessions</code>
            </div>
            <p className="text-sm text-muted-foreground">
              List all sessions in the organization
            </p>
          </div>

          <div className="border border-border rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="px-2 py-1 bg-blue-500/20 text-blue-400 rounded text-xs font-mono">
                POST
              </span>
              <code className="text-sm">/organizations/:slug/sessions</code>
            </div>
            <p className="text-sm text-muted-foreground mb-4">
              Create a new session
            </p>
            <div className="bg-[#1a1a1a] rounded-lg p-3 font-mono text-xs overflow-x-auto">
              <pre className="text-green-400">{`{
  "agent_type_id": 1,
  "runner_id": 1,
  "repository_id": 123,
  "ticket_id": 456,
  "initial_prompt": "Help me refactor this code",
  "model": "sonnet",
  "permission_mode": "default"
}`}</pre>
            </div>
          </div>

          <div className="border border-border rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="px-2 py-1 bg-green-500/20 text-green-400 rounded text-xs font-mono">
                GET
              </span>
              <code className="text-sm">/organizations/:slug/sessions/:key</code>
            </div>
            <p className="text-sm text-muted-foreground">
              Get session details by key
            </p>
          </div>

          <div className="border border-border rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="px-2 py-1 bg-blue-500/20 text-blue-400 rounded text-xs font-mono">
                POST
              </span>
              <code className="text-sm">
                /organizations/:slug/sessions/:key/terminate
              </code>
            </div>
            <p className="text-sm text-muted-foreground">Terminate a session</p>
          </div>
        </div>
      </section>

      {/* Tickets API */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Tickets</h2>

        <div className="space-y-6">
          <div className="border border-border rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="px-2 py-1 bg-green-500/20 text-green-400 rounded text-xs font-mono">
                GET
              </span>
              <code className="text-sm">/organizations/:slug/tickets</code>
            </div>
            <p className="text-sm text-muted-foreground">
              List tickets with optional filters
            </p>
          </div>

          <div className="border border-border rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="px-2 py-1 bg-blue-500/20 text-blue-400 rounded text-xs font-mono">
                POST
              </span>
              <code className="text-sm">/organizations/:slug/tickets</code>
            </div>
            <p className="text-sm text-muted-foreground mb-4">
              Create a new ticket
            </p>
            <div className="bg-[#1a1a1a] rounded-lg p-3 font-mono text-xs overflow-x-auto">
              <pre className="text-green-400">{`{
  "title": "Implement user authentication",
  "description": "Add JWT-based authentication",
  "type": "feature",
  "priority": "high",
  "repository_id": 123
}`}</pre>
            </div>
          </div>

          <div className="border border-border rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="px-2 py-1 bg-green-500/20 text-green-400 rounded text-xs font-mono">
                GET
              </span>
              <code className="text-sm">
                /organizations/:slug/tickets/:identifier
              </code>
            </div>
            <p className="text-sm text-muted-foreground">
              Get ticket by identifier (e.g., AM-123)
            </p>
          </div>

          <div className="border border-border rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="px-2 py-1 bg-yellow-500/20 text-yellow-400 rounded text-xs font-mono">
                PUT
              </span>
              <code className="text-sm">
                /organizations/:slug/tickets/:identifier
              </code>
            </div>
            <p className="text-sm text-muted-foreground">Update a ticket</p>
          </div>

          <div className="border border-border rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="px-2 py-1 bg-green-500/20 text-green-400 rounded text-xs font-mono">
                GET
              </span>
              <code className="text-sm">/organizations/:slug/tickets/board</code>
            </div>
            <p className="text-sm text-muted-foreground">
              Get Kanban board view grouped by status
            </p>
          </div>
        </div>
      </section>

      {/* Channels API */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Channels</h2>

        <div className="space-y-6">
          <div className="border border-border rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="px-2 py-1 bg-green-500/20 text-green-400 rounded text-xs font-mono">
                GET
              </span>
              <code className="text-sm">/organizations/:slug/channels</code>
            </div>
            <p className="text-sm text-muted-foreground">List all channels</p>
          </div>

          <div className="border border-border rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="px-2 py-1 bg-blue-500/20 text-blue-400 rounded text-xs font-mono">
                POST
              </span>
              <code className="text-sm">/organizations/:slug/channels</code>
            </div>
            <p className="text-sm text-muted-foreground">Create a new channel</p>
          </div>

          <div className="border border-border rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="px-2 py-1 bg-green-500/20 text-green-400 rounded text-xs font-mono">
                GET
              </span>
              <code className="text-sm">
                /organizations/:slug/channels/:id/messages
              </code>
            </div>
            <p className="text-sm text-muted-foreground">
              Get messages from a channel
            </p>
          </div>

          <div className="border border-border rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="px-2 py-1 bg-blue-500/20 text-blue-400 rounded text-xs font-mono">
                POST
              </span>
              <code className="text-sm">
                /organizations/:slug/channels/:id/messages
              </code>
            </div>
            <p className="text-sm text-muted-foreground">
              Send a message to a channel
            </p>
          </div>
        </div>
      </section>

      {/* Runners API */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Runners</h2>

        <div className="space-y-6">
          <div className="border border-border rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="px-2 py-1 bg-green-500/20 text-green-400 rounded text-xs font-mono">
                GET
              </span>
              <code className="text-sm">/organizations/:slug/runners</code>
            </div>
            <p className="text-sm text-muted-foreground">List all runners</p>
          </div>

          <div className="border border-border rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="px-2 py-1 bg-green-500/20 text-green-400 rounded text-xs font-mono">
                GET
              </span>
              <code className="text-sm">
                /organizations/:slug/runners/available
              </code>
            </div>
            <p className="text-sm text-muted-foreground">
              List available runners (online with capacity)
            </p>
          </div>

          <div className="border border-border rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="px-2 py-1 bg-blue-500/20 text-blue-400 rounded text-xs font-mono">
                POST
              </span>
              <code className="text-sm">
                /organizations/:slug/runners/tokens
              </code>
            </div>
            <p className="text-sm text-muted-foreground">
              Create a runner registration token
            </p>
          </div>
        </div>
      </section>

      {/* WebSocket */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">WebSocket Endpoints</h2>

        <div className="space-y-6">
          <div className="border border-border rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="px-2 py-1 bg-purple-500/20 text-purple-400 rounded text-xs font-mono">
                WS
              </span>
              <code className="text-sm">/ws/terminal/:sessionKey</code>
            </div>
            <p className="text-sm text-muted-foreground">
              Connect to session terminal for real-time interaction
            </p>
          </div>

          <div className="border border-border rounded-lg p-4">
            <div className="flex items-center gap-2 mb-2">
              <span className="px-2 py-1 bg-purple-500/20 text-purple-400 rounded text-xs font-mono">
                WS
              </span>
              <code className="text-sm">/ws/events</code>
            </div>
            <p className="text-sm text-muted-foreground">
              Subscribe to real-time events (session updates, messages, etc.)
            </p>
          </div>
        </div>
      </section>
    </div>
  );
}
