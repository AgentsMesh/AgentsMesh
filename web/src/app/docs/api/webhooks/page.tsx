export default function WebhooksPage() {
  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">Webhooks</h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        AgentMesh can receive webhooks from Git providers to trigger automated
        workflows and sync repository events.
      </p>

      {/* Supported Providers */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Supported Providers</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">GitHub</h3>
            <p className="text-sm text-muted-foreground">
              HMAC-SHA256 signature verification
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">GitLab</h3>
            <p className="text-sm text-muted-foreground">
              Token-based verification
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">Gitee</h3>
            <p className="text-sm text-muted-foreground">
              Token and HMAC verification
            </p>
          </div>
        </div>
      </section>

      {/* Webhook URL */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Webhook URL</h2>
        <p className="text-muted-foreground mb-4">
          Configure your Git provider to send webhooks to:
        </p>
        <div className="bg-[#1a1a1a] rounded-lg p-4 font-mono text-sm overflow-x-auto">
          <pre className="text-green-400">{`https://api.agentmesh.dev/api/v1/webhooks/:provider/:organization_id

# Examples:
https://api.agentmesh.dev/api/v1/webhooks/github/123
https://api.agentmesh.dev/api/v1/webhooks/gitlab/123
https://api.agentmesh.dev/api/v1/webhooks/gitee/123`}</pre>
        </div>
      </section>

      {/* Supported Events */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Supported Events</h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">Event</th>
                <th className="text-left p-3 border-b border-border">
                  Description
                </th>
                <th className="text-left p-3 border-b border-border">
                  Providers
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border font-medium">Push</td>
                <td className="p-3 border-b border-border">
                  Code pushed to repository
                </td>
                <td className="p-3 border-b border-border">All</td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  Pull Request / Merge Request
                </td>
                <td className="p-3 border-b border-border">
                  PR/MR opened, updated, merged, or closed
                </td>
                <td className="p-3 border-b border-border">All</td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  Pipeline / Workflow
                </td>
                <td className="p-3 border-b border-border">
                  CI/CD pipeline status changes
                </td>
                <td className="p-3 border-b border-border">GitLab, GitHub</td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  Issue
                </td>
                <td className="p-3 border-b border-border">
                  Issue created, updated, or closed
                </td>
                <td className="p-3 border-b border-border">All</td>
              </tr>
              <tr>
                <td className="p-3 font-medium">Comment</td>
                <td className="p-3">Comments on issues or PRs/MRs</td>
                <td className="p-3">All</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* GitHub Setup */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">GitHub Setup</h2>
        <ol className="list-decimal list-inside text-muted-foreground space-y-3">
          <li>
            Go to your repository&apos;s <strong>Settings → Webhooks</strong>
          </li>
          <li>
            Click <strong>Add webhook</strong>
          </li>
          <li>
            Set <strong>Payload URL</strong> to your AgentMesh webhook endpoint
          </li>
          <li>
            Set <strong>Content type</strong> to{" "}
            <code className="bg-muted px-1 rounded">application/json</code>
          </li>
          <li>
            Set <strong>Secret</strong> to your webhook secret (from AgentMesh
            settings)
          </li>
          <li>
            Select events: Push, Pull requests, Issues, Issue comments,
            Workflow runs
          </li>
          <li>
            Click <strong>Add webhook</strong>
          </li>
        </ol>
      </section>

      {/* GitLab Setup */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">GitLab Setup</h2>
        <ol className="list-decimal list-inside text-muted-foreground space-y-3">
          <li>
            Go to your project&apos;s <strong>Settings → Webhooks</strong>
          </li>
          <li>
            Set <strong>URL</strong> to your AgentMesh webhook endpoint
          </li>
          <li>
            Set <strong>Secret token</strong> to your webhook secret
          </li>
          <li>
            Select triggers: Push events, Merge request events, Pipeline events,
            Issues events, Note events
          </li>
          <li>
            Click <strong>Add webhook</strong>
          </li>
        </ol>
      </section>

      {/* Webhook Actions */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Automated Actions</h2>
        <p className="text-muted-foreground mb-4">
          Webhooks can trigger the following automated actions:
        </p>
        <div className="space-y-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">Commit Linking</h3>
            <p className="text-sm text-muted-foreground">
              Automatically link commits to tickets when commit messages
              reference ticket identifiers (e.g., &quot;Fix AM-123: Bug in
              auth&quot;).
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">MR/PR Status Sync</h3>
            <p className="text-sm text-muted-foreground">
              Keep merge request status synchronized with ticket status.
              Automatically update tickets when PRs are merged.
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">Pipeline Notifications</h3>
            <p className="text-sm text-muted-foreground">
              Notify sessions when CI/CD pipelines complete, allowing agents to
              respond to build failures or deployments.
            </p>
          </div>
        </div>
      </section>

      {/* Security */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Security</h2>
        <div className="bg-muted rounded-lg p-4">
          <p className="text-sm text-muted-foreground mb-4">
            <strong>Important:</strong> Always verify webhook signatures to
            ensure requests are from your Git provider.
          </p>
          <ul className="list-disc list-inside text-sm text-muted-foreground space-y-2">
            <li>
              <strong>GitHub:</strong> Uses HMAC-SHA256 in{" "}
              <code className="bg-background px-1 rounded">
                X-Hub-Signature-256
              </code>{" "}
              header
            </li>
            <li>
              <strong>GitLab:</strong> Uses secret token in{" "}
              <code className="bg-background px-1 rounded">X-Gitlab-Token</code>{" "}
              header
            </li>
            <li>
              <strong>Gitee:</strong> Supports both token and HMAC verification
            </li>
          </ul>
        </div>
      </section>

      {/* Troubleshooting */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">Troubleshooting</h2>
        <div className="space-y-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">Webhook not received</h3>
            <p className="text-sm text-muted-foreground">
              Check that your AgentMesh server is accessible from the internet.
              For self-hosted installations, ensure the webhook URL is publicly
              reachable.
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">Signature verification failed</h3>
            <p className="text-sm text-muted-foreground">
              Verify that the webhook secret in your Git provider matches the
              one configured in AgentMesh. Regenerate the secret if needed.
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">Events not triggering actions</h3>
            <p className="text-sm text-muted-foreground">
              Check that the correct events are selected in your Git provider
              webhook settings. AgentMesh ignores events it doesn&apos;t handle.
            </p>
          </div>
        </div>
      </section>
    </div>
  );
}
