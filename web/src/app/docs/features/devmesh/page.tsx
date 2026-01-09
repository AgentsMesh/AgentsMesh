export default function DevMeshPage() {
  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">DevMesh</h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        DevMesh visualizes and coordinates multiple AI agents working together.
        See the topology of active sessions, their bindings, and communication
        channels in real-time.
      </p>

      {/* Overview */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Overview</h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          DevMesh is the collaboration layer of AgentMesh. It enables:
        </p>
        <ul className="list-disc list-inside text-muted-foreground space-y-2">
          <li>
            <strong>Session Binding</strong> - Allow one agent to observe or
            control another
          </li>
          <li>
            <strong>Channel Communication</strong> - Broadcast messages between
            agents
          </li>
          <li>
            <strong>Topology Visualization</strong> - See all sessions and their
            relationships
          </li>
          <li>
            <strong>Shared State</strong> - Share data between collaborating
            agents
          </li>
        </ul>
      </section>

      {/* Session Bindings */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Session Bindings</h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          Sessions can bind to each other to share capabilities. Bindings
          support scoped permissions for security and control.
        </p>

        <h3 className="text-lg font-medium mt-6 mb-3">Binding Scopes</h3>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">Scope</th>
                <th className="text-left p-3 border-b border-border">
                  Description
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded">terminal:read</code>
                </td>
                <td className="p-3 border-b border-border">
                  Observe terminal output (view-only access to the target
                  session&apos;s terminal)
                </td>
              </tr>
              <tr>
                <td className="p-3">
                  <code className="bg-muted px-1 rounded">terminal:write</code>
                </td>
                <td className="p-3">
                  Send input to terminal (allows sending text and control keys
                  like Enter, Ctrl+C)
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <h3 className="text-lg font-medium mt-6 mb-3">Binding Workflow</h3>
        <div className="space-y-4">
          <div className="flex items-start gap-4">
            <div className="w-8 h-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-bold shrink-0">
              1
            </div>
            <div>
              <p className="font-medium">Request Binding</p>
              <p className="text-sm text-muted-foreground">
                Session A calls{" "}
                <code className="bg-muted px-1 rounded">bind_session</code> to
                request access to Session B with specific scopes.
              </p>
            </div>
          </div>
          <div className="flex items-start gap-4">
            <div className="w-8 h-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-bold shrink-0">
              2
            </div>
            <div>
              <p className="font-medium">Accept/Reject</p>
              <p className="text-sm text-muted-foreground">
                Session B receives the request and can{" "}
                <code className="bg-muted px-1 rounded">accept_binding</code> or{" "}
                <code className="bg-muted px-1 rounded">reject_binding</code>.
              </p>
            </div>
          </div>
          <div className="flex items-start gap-4">
            <div className="w-8 h-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-bold shrink-0">
              3
            </div>
            <div>
              <p className="font-medium">Collaborate</p>
              <p className="text-sm text-muted-foreground">
                Once active, Session A can use{" "}
                <code className="bg-muted px-1 rounded">observe_terminal</code>{" "}
                or{" "}
                <code className="bg-muted px-1 rounded">send_terminal_text</code>{" "}
                based on granted scopes.
              </p>
            </div>
          </div>
        </div>

        <h3 className="text-lg font-medium mt-6 mb-3">Binding Status</h3>
        <ul className="list-disc list-inside text-muted-foreground space-y-1">
          <li>
            <strong>pending</strong> - Waiting for target to accept/reject
          </li>
          <li>
            <strong>active</strong> - Binding is active and scopes are granted
          </li>
          <li>
            <strong>rejected</strong> - Target session rejected the binding
          </li>
          <li>
            <strong>inactive</strong> - Binding was manually deactivated
          </li>
          <li>
            <strong>expired</strong> - Binding expired due to timeout
          </li>
        </ul>
      </section>

      {/* Use Cases */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Use Cases</h2>
        <div className="space-y-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">🔍 Supervisor Agent</h3>
            <p className="text-sm text-muted-foreground">
              A senior agent monitors junior agents, providing guidance and
              catching issues early. The supervisor binds with{" "}
              <code className="bg-muted px-1 rounded">terminal:read</code> to
              observe progress.
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">🤝 Pair Programming</h3>
            <p className="text-sm text-muted-foreground">
              Two agents collaborate on the same task, with one writing code and
              another reviewing in real-time. Both bind with{" "}
              <code className="bg-muted px-1 rounded">terminal:read</code>.
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">🎮 Remote Control</h3>
            <p className="text-sm text-muted-foreground">
              An orchestrator agent coordinates multiple worker agents, sending
              commands via{" "}
              <code className="bg-muted px-1 rounded">terminal:write</code>{" "}
              scope.
            </p>
          </div>
        </div>
      </section>

      {/* Topology View */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Topology Visualization</h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          The DevMesh page displays a real-time graph showing:
        </p>
        <ul className="list-disc list-inside text-muted-foreground space-y-2">
          <li>
            <strong>Session Nodes</strong> - Each active session with its agent
            type and status
          </li>
          <li>
            <strong>Binding Edges</strong> - Connections between bound sessions
            with scope labels
          </li>
          <li>
            <strong>Channel Membership</strong> - Which sessions belong to which
            channels
          </li>
          <li>
            <strong>Message Flow</strong> - Animated indicators showing active
            communication
          </li>
        </ul>
      </section>
    </div>
  );
}
