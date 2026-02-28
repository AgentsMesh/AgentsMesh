"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function ApiLoopsPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.api.loops.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.api.loops.description")}
      </p>

      {/* Endpoints */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.api.loops.endpoints.title")}
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
                  <code className="bg-muted px-1 rounded text-blue-600 dark:text-blue-400">GET</code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">/loops</td>
                <td className="p-3 border-b border-border font-mono text-xs">loops:read</td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.loops.endpoints.list")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-green-600 dark:text-green-400">POST</code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">/loops</td>
                <td className="p-3 border-b border-border font-mono text-xs">loops:write</td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.loops.endpoints.create")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-blue-600 dark:text-blue-400">GET</code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">/loops/:slug</td>
                <td className="p-3 border-b border-border font-mono text-xs">loops:read</td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.loops.endpoints.get")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-orange-600 dark:text-orange-400">PUT</code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">/loops/:slug</td>
                <td className="p-3 border-b border-border font-mono text-xs">loops:write</td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.loops.endpoints.update")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-red-600 dark:text-red-400">DELETE</code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">/loops/:slug</td>
                <td className="p-3 border-b border-border font-mono text-xs">loops:write</td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.loops.endpoints.delete")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-green-600 dark:text-green-400">POST</code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">/loops/:slug/enable</td>
                <td className="p-3 border-b border-border font-mono text-xs">loops:write</td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.loops.endpoints.enable")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-green-600 dark:text-green-400">POST</code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">/loops/:slug/disable</td>
                <td className="p-3 border-b border-border font-mono text-xs">loops:write</td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.loops.endpoints.disable")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-green-600 dark:text-green-400">POST</code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">/loops/:slug/trigger</td>
                <td className="p-3 border-b border-border font-mono text-xs">loops:write</td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.loops.endpoints.trigger")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-blue-600 dark:text-blue-400">GET</code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">/loops/:slug/runs</td>
                <td className="p-3 border-b border-border font-mono text-xs">loops:read</td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.loops.endpoints.listRuns")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-blue-600 dark:text-blue-400">GET</code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">/loops/:slug/runs/:run_id</td>
                <td className="p-3 border-b border-border font-mono text-xs">loops:read</td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.loops.endpoints.getRun")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-green-600 dark:text-green-400">POST</code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">/loops/:slug/runs/:run_id/cancel</td>
                <td className="p-3 border-b border-border font-mono text-xs">loops:write</td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.loops.endpoints.cancelRun")}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Code Examples */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.api.loops.examples.title")}
        </h2>

        <div className="space-y-6">
          <div>
            <h3 className="font-medium mb-2">
              {t("docs.api.loops.examples.createLoop")}
            </h3>
            <pre className="bg-muted p-4 rounded-lg text-sm overflow-x-auto">
              <code>{`curl -X POST /api/v1/orgs/{org}/loops \\
  -H "Authorization: Bearer {token}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "name": "Daily Code Review",
    "agent_type_id": 1,
    "prompt_template": "Review changes in {{branch}} branch",
    "prompt_variables": {"branch": "main"},
    "execution_mode": "autopilot",
    "cron_expression": "0 9 * * *",
    "sandbox_strategy": "persistent",
    "timeout_minutes": 30
  }'`}</code>
            </pre>
          </div>

          <div>
            <h3 className="font-medium mb-2">
              {t("docs.api.loops.examples.triggerRun")}
            </h3>
            <pre className="bg-muted p-4 rounded-lg text-sm overflow-x-auto">
              <code>{`curl -X POST /api/v1/orgs/{org}/loops/{slug}/trigger \\
  -H "Authorization: Bearer {token}" \\
  -H "Content-Type: application/json" \\
  -d '{
    "variables": {
      "branch": "feature/new-api",
      "focus_area": "security"
    }
  }'`}</code>
            </pre>
          </div>
        </div>
      </section>

      {/* Endpoint Details */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-6">
          {t("docs.api.loops.details.title")}
        </h2>

        {/* List Loops */}
        <div className="mb-8 border border-border rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-2 font-mono">
            {t("docs.api.loops.details.listLoops.title")}
          </h3>
          <p className="text-muted-foreground mb-4">
            {t("docs.api.loops.details.listLoops.description")}
          </p>
        </div>

        {/* Create Loop */}
        <div className="mb-8 border border-border rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-2 font-mono">
            {t("docs.api.loops.details.createLoop.title")}
          </h3>
          <p className="text-muted-foreground mb-4">
            {t("docs.api.loops.details.createLoop.description")}
          </p>
          <h4 className="font-medium mb-2">{t("docs.api.common.requestBody")}</h4>
          <div className="overflow-x-auto">
            <table className="w-full text-sm border border-border rounded-lg">
              <thead>
                <tr className="bg-muted">
                  <th className="text-left p-2 border-b border-border">{t("docs.api.common.fieldHeader")}</th>
                  <th className="text-left p-2 border-b border-border">{t("docs.api.common.descriptionHeader")}</th>
                </tr>
              </thead>
              <tbody className="text-muted-foreground">
                {(["name", "description", "agent_type_id", "custom_agent_type_id", "prompt_template", "prompt_variables", "repository_id", "runner_id", "branch_name", "execution_mode", "cron_expression", "sandbox_strategy", "session_persistence", "concurrency_policy", "max_concurrent_runs", "timeout_minutes", "callback_url", "autopilot_config"] as const).map((field) => (
                  <tr key={field}>
                    <td className="p-2 border-b border-border font-mono text-xs">{field}</td>
                    <td className="p-2 border-b border-border">
                      {t(`docs.api.loops.details.createLoop.fields.${field}`)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Get Loop */}
        <div className="mb-8 border border-border rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-2 font-mono">
            {t("docs.api.loops.details.getLoop.title")}
          </h3>
          <p className="text-muted-foreground">
            {t("docs.api.loops.details.getLoop.description")}
          </p>
        </div>

        {/* Update Loop */}
        <div className="mb-8 border border-border rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-2 font-mono">
            {t("docs.api.loops.details.updateLoop.title")}
          </h3>
          <p className="text-muted-foreground">
            {t("docs.api.loops.details.updateLoop.description")}
          </p>
        </div>

        {/* Delete Loop */}
        <div className="mb-8 border border-border rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-2 font-mono">
            {t("docs.api.loops.details.deleteLoop.title")}
          </h3>
          <p className="text-muted-foreground">
            {t("docs.api.loops.details.deleteLoop.description")}
          </p>
        </div>

        {/* Enable / Disable */}
        <div className="mb-8 border border-border rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-2 font-mono">
            {t("docs.api.loops.details.enableLoop.title")}
          </h3>
          <p className="text-muted-foreground mb-4">
            {t("docs.api.loops.details.enableLoop.description")}
          </p>
          <h3 className="text-lg font-semibold mb-2 font-mono">
            {t("docs.api.loops.details.disableLoop.title")}
          </h3>
          <p className="text-muted-foreground">
            {t("docs.api.loops.details.disableLoop.description")}
          </p>
        </div>

        {/* Trigger Run */}
        <div className="mb-8 border border-border rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-2 font-mono">
            {t("docs.api.loops.details.triggerLoop.title")}
          </h3>
          <p className="text-muted-foreground mb-4">
            {t("docs.api.loops.details.triggerLoop.description")}
          </p>
          <h4 className="font-medium mb-2">{t("docs.api.common.requestBody")}</h4>
          <div className="overflow-x-auto">
            <table className="w-full text-sm border border-border rounded-lg">
              <thead>
                <tr className="bg-muted">
                  <th className="text-left p-2 border-b border-border">{t("docs.api.common.fieldHeader")}</th>
                  <th className="text-left p-2 border-b border-border">{t("docs.api.common.descriptionHeader")}</th>
                </tr>
              </thead>
              <tbody className="text-muted-foreground">
                <tr>
                  <td className="p-2 border-b border-border font-mono text-xs">variables</td>
                  <td className="p-2 border-b border-border">
                    {t("docs.api.loops.details.triggerLoop.fields.variables")}
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        {/* List Runs */}
        <div className="mb-8 border border-border rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-2 font-mono">
            {t("docs.api.loops.details.listRuns.title")}
          </h3>
          <p className="text-muted-foreground">
            {t("docs.api.loops.details.listRuns.description")}
          </p>
        </div>

        {/* Get Run */}
        <div className="mb-8 border border-border rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-2 font-mono">
            {t("docs.api.loops.details.getRun.title")}
          </h3>
          <p className="text-muted-foreground">
            {t("docs.api.loops.details.getRun.description")}
          </p>
        </div>

        {/* Cancel Run */}
        <div className="mb-8 border border-border rounded-lg p-6">
          <h3 className="text-lg font-semibold mb-2 font-mono">
            {t("docs.api.loops.details.cancelRun.title")}
          </h3>
          <p className="text-muted-foreground">
            {t("docs.api.loops.details.cancelRun.description")}
          </p>
        </div>
      </section>

      <DocNavigation />
    </div>
  );
}
