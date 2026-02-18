"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function ApiPodsPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.api.pods.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.api.pods.description")}
      </p>

      {/* Endpoints */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.api.pods.endpoints.title")}
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">
                  {t("docs.api.common.methodHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.api.common.pathHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.api.common.scopeHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.api.common.descriptionHeader")}
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-blue-600 dark:text-blue-400">
                    GET
                  </code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  /pods
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  pods:read
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.pods.endpoints.list")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-blue-600 dark:text-blue-400">
                    GET
                  </code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  /pods/:key
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  pods:read
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.pods.endpoints.get")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-green-600 dark:text-green-400">
                    POST
                  </code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  /pods
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  pods:write
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.pods.endpoints.create")}
                </td>
              </tr>
              <tr>
                <td className="p-3">
                  <code className="bg-muted px-1 rounded text-green-600 dark:text-green-400">
                    POST
                  </code>
                </td>
                <td className="p-3 font-mono text-xs">
                  /pods/:key/terminate
                </td>
                <td className="p-3 font-mono text-xs">pods:write</td>
                <td className="p-3">
                  {t("docs.api.pods.endpoints.terminate")}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Code Examples */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.api.pods.examples.title")}
        </h2>
        <div className="space-y-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-3">
              {t("docs.api.pods.examples.listPods")}
            </h3>
            <div className="bg-muted rounded-lg p-4 font-mono text-sm">
              <pre className="text-green-500 dark:text-green-400">{`curl -X GET \\
  "https://your-domain.com/api/v1/ext/orgs/my-org/pods" \\
  -H "X-API-Key: amk_your_api_key_here"`}</pre>
            </div>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-3">
              {t("docs.api.pods.examples.createPod")}
            </h3>
            <div className="bg-muted rounded-lg p-4 font-mono text-sm">
              <pre className="text-green-500 dark:text-green-400">{`curl -X POST \\
  "https://your-domain.com/api/v1/ext/orgs/my-org/pods" \\
  -H "X-API-Key: amk_your_api_key_here" \\
  -H "Content-Type: application/json" \\
  -d '{
    "agent_type_id": "claude-code",
    "runner_id": "runner-uuid",
    "initial_prompt": "Fix the login bug"
  }'`}</pre>
            </div>
          </div>
        </div>
      </section>

      {/* Endpoint Details */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-6">
          {t("docs.api.pods.details.title")}
        </h2>
        <div className="space-y-8">
          {/* GET /pods */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <div>
              <h3 className="text-lg font-semibold mb-2">
                <code className="bg-muted px-2 py-1 rounded text-blue-600 dark:text-blue-400">
                  GET
                </code>{" "}
                <code className="text-sm">/pods</code>
              </h3>
              <p className="text-muted-foreground text-sm">
                {t("docs.api.pods.details.listPods.description")}
              </p>
            </div>

            <div>
              <h4 className="font-medium mb-2">
                {t("docs.api.common.queryParams")}
              </h4>
              <div className="overflow-x-auto">
                <table className="w-full text-sm border border-border rounded-lg">
                  <thead>
                    <tr className="bg-muted">
                      <th className="text-left p-3 border-b border-border">
                        {t("docs.api.common.paramHeader")}
                      </th>
                      <th className="text-left p-3 border-b border-border">
                        {t("docs.api.common.typeHeader")}
                      </th>
                      <th className="text-left p-3 border-b border-border">
                        {t("docs.api.common.requiredHeader")}
                      </th>
                      <th className="text-left p-3 border-b border-border">
                        {t("docs.api.common.defaultHeader")}
                      </th>
                      <th className="text-left p-3 border-b border-border">
                        {t("docs.api.common.descriptionHeader")}
                      </th>
                    </tr>
                  </thead>
                  <tbody className="text-muted-foreground">
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        status
                      </td>
                      <td className="p-3 border-b border-border">string</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">-</td>
                      <td className="p-3 border-b border-border">
                        {t("docs.api.pods.details.listPods.params.status")}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        limit
                      </td>
                      <td className="p-3 border-b border-border">integer</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">20</td>
                      <td className="p-3 border-b border-border">
                        {t("docs.api.pods.details.listPods.params.limit")}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 font-mono text-xs">offset</td>
                      <td className="p-3">integer</td>
                      <td className="p-3">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3">0</td>
                      <td className="p-3">
                        {t("docs.api.pods.details.listPods.params.offset")}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>

            <div>
              <h4 className="font-medium mb-2">
                {t("docs.api.common.responseExample")}
              </h4>
              <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto">
                <pre className="text-green-500 dark:text-green-400">{`{
  "pods": [
    {
      "key": "pod-abc123",
      "status": "running",
      "agent_type_id": "claude-code",
      "runner_id": "550e8400-e29b-41d4-a716-446655440000",
      "initial_prompt": "Fix the login bug",
      "repository_id": 1,
      "branch": "main",
      "created_at": "2025-01-15T10:30:00Z",
      "updated_at": "2025-01-15T10:35:00Z"
    }
  ],
  "total": 42,
  "limit": 20,
  "offset": 0
}`}</pre>
              </div>
            </div>
          </div>

          {/* GET /pods/:key */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <div>
              <h3 className="text-lg font-semibold mb-2">
                <code className="bg-muted px-2 py-1 rounded text-blue-600 dark:text-blue-400">
                  GET
                </code>{" "}
                <code className="text-sm">/pods/:key</code>
              </h3>
              <p className="text-muted-foreground text-sm">
                {t("docs.api.pods.details.getPod.description")}
              </p>
            </div>

            <div>
              <h4 className="font-medium mb-2">
                {t("docs.api.common.pathParams")}
              </h4>
              <div className="overflow-x-auto">
                <table className="w-full text-sm border border-border rounded-lg">
                  <thead>
                    <tr className="bg-muted">
                      <th className="text-left p-3 border-b border-border">
                        {t("docs.api.common.paramHeader")}
                      </th>
                      <th className="text-left p-3 border-b border-border">
                        {t("docs.api.common.typeHeader")}
                      </th>
                      <th className="text-left p-3 border-b border-border">
                        {t("docs.api.common.requiredHeader")}
                      </th>
                      <th className="text-left p-3 border-b border-border">
                        {t("docs.api.common.descriptionHeader")}
                      </th>
                    </tr>
                  </thead>
                  <tbody className="text-muted-foreground">
                    <tr>
                      <td className="p-3 font-mono text-xs">key</td>
                      <td className="p-3">string</td>
                      <td className="p-3">
                        <span className="text-xs bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.requiredBadge")}
                        </span>
                      </td>
                      <td className="p-3">
                        {t("docs.api.pods.details.getPod.params.key")}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>

            <div>
              <h4 className="font-medium mb-2">
                {t("docs.api.common.responseExample")}
              </h4>
              <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto">
                <pre className="text-green-500 dark:text-green-400">{`{
  "pod": {
    "key": "pod-abc123",
    "status": "running",
    "agent_type_id": "claude-code",
    "runner_id": "550e8400-e29b-41d4-a716-446655440000",
    "initial_prompt": "Fix the login bug",
    "repository_id": 1,
    "branch": "main",
    "ticket_identifier": "AM-42",
    "channel_id": 5,
    "sandbox_type": "worktree",
    "auto_close": false,
    "pod_timeout_minutes": 60,
    "max_turns": 100,
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-15T10:35:00Z"
  }
}`}</pre>
              </div>
            </div>
          </div>

          {/* POST /pods */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <div>
              <h3 className="text-lg font-semibold mb-2">
                <code className="bg-muted px-2 py-1 rounded text-green-600 dark:text-green-400">
                  POST
                </code>{" "}
                <code className="text-sm">/pods</code>
              </h3>
              <p className="text-muted-foreground text-sm">
                {t("docs.api.pods.details.createPod.description")}
              </p>
            </div>

            <div>
              <h4 className="font-medium mb-2">
                {t("docs.api.common.requestBody")}
              </h4>
              <div className="overflow-x-auto">
                <table className="w-full text-sm border border-border rounded-lg">
                  <thead>
                    <tr className="bg-muted">
                      <th className="text-left p-3 border-b border-border">
                        {t("docs.api.common.fieldHeader")}
                      </th>
                      <th className="text-left p-3 border-b border-border">
                        {t("docs.api.common.typeHeader")}
                      </th>
                      <th className="text-left p-3 border-b border-border">
                        {t("docs.api.common.requiredHeader")}
                      </th>
                      <th className="text-left p-3 border-b border-border">
                        {t("docs.api.common.descriptionHeader")}
                      </th>
                    </tr>
                  </thead>
                  <tbody className="text-muted-foreground">
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        runner_id
                      </td>
                      <td className="p-3 border-b border-border">string</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.requiredBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t("docs.api.pods.details.createPod.fields.runner_id")}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        agent_type_id
                      </td>
                      <td className="p-3 border-b border-border">string</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.pods.details.createPod.fields.agent_type_id"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        initial_prompt
                      </td>
                      <td className="p-3 border-b border-border">string</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.pods.details.createPod.fields.initial_prompt"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        repository_id
                      </td>
                      <td className="p-3 border-b border-border">integer</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.pods.details.createPod.fields.repository_id"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        branch
                      </td>
                      <td className="p-3 border-b border-border">string</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t("docs.api.pods.details.createPod.fields.branch")}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        ticket_identifier
                      </td>
                      <td className="p-3 border-b border-border">string</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.pods.details.createPod.fields.ticket_identifier"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        channel_id
                      </td>
                      <td className="p-3 border-b border-border">integer</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.pods.details.createPod.fields.channel_id"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        sandbox_type
                      </td>
                      <td className="p-3 border-b border-border">string</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.pods.details.createPod.fields.sandbox_type"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        environment_variables
                      </td>
                      <td className="p-3 border-b border-border">object</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.pods.details.createPod.fields.environment_variables"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        custom_agent_command
                      </td>
                      <td className="p-3 border-b border-border">string</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.pods.details.createPod.fields.custom_agent_command"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        custom_agent_args
                      </td>
                      <td className="p-3 border-b border-border">string[]</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.pods.details.createPod.fields.custom_agent_args"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        custom_mcp_config
                      </td>
                      <td className="p-3 border-b border-border">object</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.pods.details.createPod.fields.custom_mcp_config"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        auto_close
                      </td>
                      <td className="p-3 border-b border-border">boolean</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.pods.details.createPod.fields.auto_close"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        auto_close_delay_seconds
                      </td>
                      <td className="p-3 border-b border-border">integer</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.pods.details.createPod.fields.auto_close_delay_seconds"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        pod_timeout_minutes
                      </td>
                      <td className="p-3 border-b border-border">integer</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.pods.details.createPod.fields.pod_timeout_minutes"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 font-mono text-xs">max_turns</td>
                      <td className="p-3">integer</td>
                      <td className="p-3">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3">
                        {t(
                          "docs.api.pods.details.createPod.fields.max_turns"
                        )}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>

            <div>
              <h4 className="font-medium mb-2">
                {t("docs.api.common.responseExample")}
              </h4>
              <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto">
                <pre className="text-green-500 dark:text-green-400">{`{
  "pod": {
    "key": "pod-xyz789",
    "status": "creating",
    "agent_type_id": "claude-code",
    "runner_id": "550e8400-e29b-41d4-a716-446655440000",
    "initial_prompt": "Fix the login bug",
    "repository_id": 1,
    "branch": "feature/auth",
    "sandbox_type": "worktree",
    "auto_close": true,
    "auto_close_delay_seconds": 30,
    "pod_timeout_minutes": 60,
    "max_turns": 100,
    "created_at": "2025-01-15T10:30:00Z",
    "updated_at": "2025-01-15T10:30:00Z"
  }
}`}</pre>
              </div>
            </div>
          </div>

          {/* POST /pods/:key/terminate */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <div>
              <h3 className="text-lg font-semibold mb-2">
                <code className="bg-muted px-2 py-1 rounded text-green-600 dark:text-green-400">
                  POST
                </code>{" "}
                <code className="text-sm">/pods/:key/terminate</code>
              </h3>
              <p className="text-muted-foreground text-sm">
                {t("docs.api.pods.details.terminatePod.description")}
              </p>
            </div>

            <div>
              <h4 className="font-medium mb-2">
                {t("docs.api.common.pathParams")}
              </h4>
              <div className="overflow-x-auto">
                <table className="w-full text-sm border border-border rounded-lg">
                  <thead>
                    <tr className="bg-muted">
                      <th className="text-left p-3 border-b border-border">
                        {t("docs.api.common.paramHeader")}
                      </th>
                      <th className="text-left p-3 border-b border-border">
                        {t("docs.api.common.typeHeader")}
                      </th>
                      <th className="text-left p-3 border-b border-border">
                        {t("docs.api.common.requiredHeader")}
                      </th>
                      <th className="text-left p-3 border-b border-border">
                        {t("docs.api.common.descriptionHeader")}
                      </th>
                    </tr>
                  </thead>
                  <tbody className="text-muted-foreground">
                    <tr>
                      <td className="p-3 font-mono text-xs">key</td>
                      <td className="p-3">string</td>
                      <td className="p-3">
                        <span className="text-xs bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.requiredBadge")}
                        </span>
                      </td>
                      <td className="p-3">
                        {t("docs.api.pods.details.terminatePod.params.key")}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>

            <div>
              <h4 className="font-medium mb-2">
                {t("docs.api.common.responseExample")}
              </h4>
              <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto">
                <pre className="text-green-500 dark:text-green-400">{`{
  "message": "Pod terminated"
}`}</pre>
              </div>
            </div>
          </div>
        </div>
      </section>

      <DocNavigation />
    </div>
  );
}
