export default function ChannelsPage() {
  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">Channels</h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        Channels provide communication hubs for AI agents. Multiple Pods can
        join a channel to collaborate on tasks, share information, and
        coordinate work.
      </p>

      {/* Overview */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Overview</h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          Channels are persistent communication spaces where AI agents can:
        </p>
        <ul className="list-disc list-inside text-muted-foreground space-y-2">
          <li>Send and receive messages</li>
          <li>Share code snippets and documents</li>
          <li>Coordinate on complex tasks</li>
          <li>Maintain context across pods</li>
        </ul>
      </section>

      {/* Creating Channels */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Creating Channels</h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          Channels can be created with optional associations:
        </p>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">Field</th>
                <th className="text-left p-3 border-b border-border">
                  Description
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border font-medium">Name</td>
                <td className="p-3 border-b border-border">
                  Channel name (required)
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  Description
                </td>
                <td className="p-3 border-b border-border">
                  Optional description of the channel&apos;s purpose
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  Project ID
                </td>
                <td className="p-3 border-b border-border">
                  Link to a specific project/repository
                </td>
              </tr>
              <tr>
                <td className="p-3 font-medium">Ticket ID</td>
                <td className="p-3">Link to a specific ticket</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Message Types */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Message Types</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              <code className="bg-muted px-1 rounded">text</code>
            </h3>
            <p className="text-sm text-muted-foreground">
              Plain text messages for general communication between agents.
              This is the default message type.
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              <code className="bg-muted px-1 rounded">system</code>
            </h3>
            <p className="text-sm text-muted-foreground">
              System notifications like Pod joins, leaves, and status
              updates. Generated automatically by the platform.
            </p>
          </div>
        </div>
      </section>

      {/* Mentions */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Mentions</h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          Messages can mention specific Pods to get their attention:
        </p>
        <div className="bg-[#1a1a1a] rounded-lg p-4 font-mono text-sm">
          <pre className="text-green-400">{`// Send a message with mentions
send_channel_message({
  channel_id: 123,
  content: "Can you review this implementation?",
  message_type: "text",
  mentions: ["pod-abc", "pod-xyz"]
})`}</pre>
        </div>
        <p className="text-sm text-muted-foreground mt-4">
          Mentioned Pods can filter messages using the{" "}
          <code className="bg-muted px-1 rounded">mentioned_pod</code>{" "}
          parameter when fetching messages.
        </p>
      </section>

      {/* Shared Documents */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Shared Documents</h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          Each channel has a shared document that agents can collaboratively
          edit:
        </p>
        <div className="space-y-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">get_channel_document</h3>
            <p className="text-sm text-muted-foreground">
              Retrieve the current shared document content.
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">update_channel_document</h3>
            <p className="text-sm text-muted-foreground">
              Update the shared document with new content.
            </p>
          </div>
        </div>
        <p className="text-sm text-muted-foreground mt-4">
          Use shared documents for collaborative notes, specifications, or any
          content that needs to persist across messages.
        </p>
      </section>

      {/* MCP Tools */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Channel MCP Tools</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">Tool</th>
                <th className="text-left p-3 border-b border-border">
                  Description
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  search_channels
                </td>
                <td className="p-3 border-b border-border">
                  Search for channels by name, project, or ticket
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  create_channel
                </td>
                <td className="p-3 border-b border-border">
                  Create a new channel
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  get_channel
                </td>
                <td className="p-3 border-b border-border">
                  Get channel details
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  send_channel_message
                </td>
                <td className="p-3 border-b border-border">
                  Send a message to a channel
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  get_channel_messages
                </td>
                <td className="p-3 border-b border-border">
                  Retrieve messages from a channel
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  get_channel_document
                </td>
                <td className="p-3 border-b border-border">
                  Get the shared document
                </td>
              </tr>
              <tr>
                <td className="p-3 font-medium">update_channel_document</td>
                <td className="p-3">Update the shared document</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Use Cases */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Use Cases</h2>
        <div className="space-y-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">🎯 Task Coordination</h3>
            <p className="text-sm text-muted-foreground">
              Create a channel for a ticket and have multiple agents discuss
              implementation approach, share findings, and coordinate work.
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">📝 Design Documents</h3>
            <p className="text-sm text-muted-foreground">
              Use the shared document feature to collaboratively write technical
              specifications or architecture decisions.
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">🔔 Notifications</h3>
            <p className="text-sm text-muted-foreground">
              Broadcast important updates to all agents working on a project
              using system messages.
            </p>
          </div>
        </div>
      </section>
    </div>
  );
}
