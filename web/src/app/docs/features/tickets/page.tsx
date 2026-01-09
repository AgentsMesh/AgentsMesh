export default function TicketsPage() {
  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">Tickets</h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        Integrated task management with Kanban board view. Create tickets,
        assign them to AI sessions, and track progress through your workflow.
      </p>

      {/* Overview */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Overview</h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          AgentMesh Tickets provide a lightweight issue tracking system designed
          for AI-driven development. Key features include:
        </p>
        <ul className="list-disc list-inside text-muted-foreground space-y-2">
          <li>Kanban board visualization</li>
          <li>Ticket ↔ Session linking for context</li>
          <li>Git commit and merge request associations</li>
          <li>Priority and estimation tracking</li>
          <li>Labels and assignees</li>
        </ul>
      </section>

      {/* Ticket Types */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Ticket Types</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">📋 Task</h3>
            <p className="text-sm text-muted-foreground">
              General development tasks. The most common ticket type for
              day-to-day work.
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">🐛 Bug</h3>
            <p className="text-sm text-muted-foreground">
              Bug reports and fixes. Supports severity levels: critical, major,
              minor, trivial.
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">✨ Feature</h3>
            <p className="text-sm text-muted-foreground">
              New feature requests. Use for enhancements and new capabilities.
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">🎯 Epic</h3>
            <p className="text-sm text-muted-foreground">
              Large initiatives that contain multiple tasks. Use for tracking
              major projects.
            </p>
          </div>
        </div>
      </section>

      {/* Ticket Status */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Ticket Status</h2>
        <p className="text-muted-foreground mb-4">
          Tickets flow through these statuses in the Kanban board:
        </p>
        <div className="flex flex-wrap gap-2">
          <span className="px-3 py-1 bg-muted rounded text-sm">Backlog</span>
          <span className="text-muted-foreground">→</span>
          <span className="px-3 py-1 bg-muted rounded text-sm">Todo</span>
          <span className="text-muted-foreground">→</span>
          <span className="px-3 py-1 bg-blue-500/20 text-blue-400 rounded text-sm">
            In Progress
          </span>
          <span className="text-muted-foreground">→</span>
          <span className="px-3 py-1 bg-yellow-500/20 text-yellow-400 rounded text-sm">
            In Review
          </span>
          <span className="text-muted-foreground">→</span>
          <span className="px-3 py-1 bg-green-500/20 text-green-400 rounded text-sm">
            Done
          </span>
        </div>
        <p className="text-sm text-muted-foreground mt-4">
          Tickets can also be marked as <strong>Cancelled</strong> if no longer
          needed.
        </p>
      </section>

      {/* Priority */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Priority Levels</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">
                  Priority
                </th>
                <th className="text-left p-3 border-b border-border">
                  Description
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border text-red-400 font-medium">
                  Urgent
                </td>
                <td className="p-3 border-b border-border">
                  Critical issues requiring immediate attention
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border text-orange-400 font-medium">
                  High
                </td>
                <td className="p-3 border-b border-border">
                  Important tasks that should be addressed soon
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border text-yellow-400 font-medium">
                  Medium
                </td>
                <td className="p-3 border-b border-border">
                  Normal priority work
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border text-muted-foreground font-medium">
                  Low
                </td>
                <td className="p-3 border-b border-border">
                  Nice-to-have items when time permits
                </td>
              </tr>
              <tr>
                <td className="p-3 font-medium">None</td>
                <td className="p-3">Priority not yet assigned</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Session Integration */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Session Integration</h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          Link tickets to DevPod sessions to:
        </p>
        <ul className="list-disc list-inside text-muted-foreground space-y-2">
          <li>
            <strong>Provide context</strong> - The AI agent receives ticket
            details as initial context
          </li>
          <li>
            <strong>Track progress</strong> - See which sessions are working on
            which tickets
          </li>
          <li>
            <strong>Auto-update</strong> - Session completion can update ticket
            status
          </li>
          <li>
            <strong>View history</strong> - See all sessions that worked on a
            ticket
          </li>
        </ul>
      </section>

      {/* Git Integration */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Git Integration</h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          Tickets can be associated with:
        </p>
        <div className="space-y-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">Commits</h3>
            <p className="text-sm text-muted-foreground">
              Link commits to tickets by SHA. View commit messages, authors, and
              links to the source.
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">Merge Requests / Pull Requests</h3>
            <p className="text-sm text-muted-foreground">
              Track MRs/PRs associated with tickets. See status, pipeline
              results, and review state.
            </p>
          </div>
        </div>
      </section>

      {/* Estimation */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Story Points</h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          Estimate effort using Fibonacci story points:
        </p>
        <div className="flex flex-wrap gap-2">
          {[1, 2, 3, 5, 8, 13, 21].map((point) => (
            <span
              key={point}
              className="w-10 h-10 flex items-center justify-center bg-muted rounded text-sm font-medium"
            >
              {point}
            </span>
          ))}
        </div>
      </section>
    </div>
  );
}
