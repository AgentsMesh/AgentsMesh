"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function ApiRunnersPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.api.runners.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.api.runners.description")}
      </p>

      {/* Endpoints */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.api.runners.endpoints.title")}
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
                  /runners
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  runners:read
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.runners.endpoints.list")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-blue-600 dark:text-blue-400">
                    GET
                  </code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  /runners/:id
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  runners:read
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.runners.endpoints.get")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-blue-600 dark:text-blue-400">
                    GET
                  </code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  /runners/available
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  runners:read
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.runners.endpoints.available")}
                </td>
              </tr>
              <tr>
                <td className="p-3">
                  <code className="bg-muted px-1 rounded text-blue-600 dark:text-blue-400">
                    GET
                  </code>
                </td>
                <td className="p-3 font-mono text-xs">/runners/:id/pods</td>
                <td className="p-3 font-mono text-xs">runners:read</td>
                <td className="p-3">
                  {t("docs.api.runners.endpoints.pods")}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Endpoint Details */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-6">
          {t("docs.api.runners.details.title")}
        </h2>
        <div className="space-y-8">
          {/* GET /runners */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <h3 className="text-lg font-semibold">
              <code className="bg-muted px-2 py-1 rounded text-blue-600 dark:text-blue-400">
                GET
              </code>{" "}
              /runners
            </h3>
            <p className="text-muted-foreground">
              {t("docs.api.runners.details.listRunners.description")}
            </p>

            {/* Response Example */}
            <div>
              <h4 className="font-medium mb-2">
                {t("docs.api.common.responseExample")}
              </h4>
              <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto">
                <pre>{`{
  "runners": [
    {
      "id": 1,
      "name": "dev-runner-01",
      "status": "online",
      "version": "1.2.0",
      "os": "linux",
      "arch": "amd64",
      "labels": ["gpu", "high-memory"],
      "pod_count": 3,
      "max_pods": 10,
      "last_heartbeat_at": "2025-01-15T14:30:00Z",
      "created_at": "2025-01-01T00:00:00Z"
    }
  ]
}`}</pre>
              </div>
            </div>
          </div>

          {/* GET /runners/:id */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <h3 className="text-lg font-semibold">
              <code className="bg-muted px-2 py-1 rounded text-blue-600 dark:text-blue-400">
                GET
              </code>{" "}
              /runners/:id
            </h3>
            <p className="text-muted-foreground">
              {t("docs.api.runners.details.getRunner.description")}
            </p>

            {/* Path Parameters */}
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
                      <td className="p-3 font-mono text-xs">id</td>
                      <td className="p-3">integer</td>
                      <td className="p-3">
                        <span className="text-xs bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.requiredBadge")}
                        </span>
                      </td>
                      <td className="p-3">
                        {t("docs.api.runners.details.getRunner.params.id")}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>

            {/* Response Example */}
            <div>
              <h4 className="font-medium mb-2">
                {t("docs.api.common.responseExample")}
              </h4>
              <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto">
                <pre>{`{
  "runner": {
    "id": 1,
    "name": "dev-runner-01",
    "status": "online",
    "version": "1.2.0",
    "os": "linux",
    "arch": "amd64",
    "labels": ["gpu", "high-memory"],
    "pod_count": 3,
    "max_pods": 10,
    "last_heartbeat_at": "2025-01-15T14:30:00Z",
    "created_at": "2025-01-01T00:00:00Z"
  }
}`}</pre>
              </div>
            </div>
          </div>

          {/* GET /runners/available */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <h3 className="text-lg font-semibold">
              <code className="bg-muted px-2 py-1 rounded text-blue-600 dark:text-blue-400">
                GET
              </code>{" "}
              /runners/available
            </h3>
            <p className="text-muted-foreground">
              {t("docs.api.runners.details.availableRunners.description")}
            </p>

            {/* Response Example */}
            <div>
              <h4 className="font-medium mb-2">
                {t("docs.api.common.responseExample")}
              </h4>
              <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto">
                <pre>{`{
  "runners": [
    {
      "id": 1,
      "name": "dev-runner-01",
      "status": "online",
      "version": "1.2.0",
      "os": "linux",
      "arch": "amd64",
      "labels": ["gpu", "high-memory"],
      "pod_count": 3,
      "max_pods": 10,
      "last_heartbeat_at": "2025-01-15T14:30:00Z",
      "created_at": "2025-01-01T00:00:00Z"
    }
  ]
}`}</pre>
              </div>
            </div>
          </div>

          {/* GET /runners/:id/pods */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <h3 className="text-lg font-semibold">
              <code className="bg-muted px-2 py-1 rounded text-blue-600 dark:text-blue-400">
                GET
              </code>{" "}
              /runners/:id/pods
            </h3>
            <p className="text-muted-foreground">
              {t("docs.api.runners.details.runnerPods.description")}
            </p>

            {/* Path Parameters */}
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
                      <td className="p-3 font-mono text-xs">id</td>
                      <td className="p-3">integer</td>
                      <td className="p-3">
                        <span className="text-xs bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.requiredBadge")}
                        </span>
                      </td>
                      <td className="p-3">
                        {t("docs.api.runners.details.runnerPods.params.id")}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>

            {/* Query Parameters */}
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
                        {t(
                          "docs.api.runners.details.runnerPods.params.status"
                        )}
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
                      <td className="p-3 border-b border-border">50</td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.runners.details.runnerPods.params.limit"
                        )}
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
                        {t(
                          "docs.api.runners.details.runnerPods.params.offset"
                        )}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>

            {/* Response Example */}
            <div>
              <h4 className="font-medium mb-2">
                {t("docs.api.common.responseExample")}
              </h4>
              <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto">
                <pre>{`{
  "pods": [
    {
      "key": "pod-abc123",
      "status": "running",
      "agent_type_id": "claude-code",
      "initial_prompt": "Fix the login bug",
      "created_at": "2025-01-15T10:30:00Z"
    }
  ],
  "total": 8,
  "limit": 50,
  "offset": 0
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
