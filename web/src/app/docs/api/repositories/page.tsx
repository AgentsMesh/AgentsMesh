"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function ApiRepositoriesPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.api.repositories.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.api.repositories.description")}
      </p>

      {/* Endpoints */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.api.repositories.endpoints.title")}
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
                  /repositories
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  repos:read
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.repositories.endpoints.list")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-blue-600 dark:text-blue-400">
                    GET
                  </code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  /repositories/:id
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  repos:read
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.repositories.endpoints.get")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-blue-600 dark:text-blue-400">
                    GET
                  </code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  /repositories/:id/branches
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  repos:read
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.repositories.endpoints.branches")}
                </td>
              </tr>
              <tr>
                <td className="p-3">
                  <code className="bg-muted px-1 rounded text-blue-600 dark:text-blue-400">
                    GET
                  </code>
                </td>
                <td className="p-3 font-mono text-xs">
                  /repositories/:id/merge-requests
                </td>
                <td className="p-3 font-mono text-xs">repos:read</td>
                <td className="p-3">
                  {t("docs.api.repositories.endpoints.mergeRequests")}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Endpoint Details */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-6">
          {t("docs.api.repositories.details.title")}
        </h2>
        <div className="space-y-8">
          {/* GET /repositories */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <h3 className="text-lg font-semibold">
              <code className="bg-muted px-2 py-1 rounded text-blue-600 dark:text-blue-400">
                GET
              </code>{" "}
              /repositories
            </h3>
            <p className="text-muted-foreground">
              {t("docs.api.repositories.details.listRepos.description")}
            </p>

            {/* Response Example */}
            <div>
              <h4 className="font-medium mb-2">
                {t("docs.api.common.responseExample")}
              </h4>
              <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto">
                <pre>{`{
  "repositories": [
    {
      "id": 1,
      "name": "agentsmesh",
      "full_name": "org/agentsmesh",
      "provider": "gitlab",
      "url": "https://gitlab.com/org/agentsmesh",
      "default_branch": "main",
      "created_at": "2025-01-01T00:00:00Z"
    }
  ]
}`}</pre>
              </div>
            </div>
          </div>

          {/* GET /repositories/:id */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <h3 className="text-lg font-semibold">
              <code className="bg-muted px-2 py-1 rounded text-blue-600 dark:text-blue-400">
                GET
              </code>{" "}
              /repositories/:id
            </h3>
            <p className="text-muted-foreground">
              {t("docs.api.repositories.details.getRepo.description")}
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
                        {t(
                          "docs.api.repositories.details.getRepo.params.id"
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
  "repository": {
    "id": 1,
    "name": "agentsmesh",
    "full_name": "org/agentsmesh",
    "provider": "gitlab",
    "url": "https://gitlab.com/org/agentsmesh",
    "default_branch": "main",
    "created_at": "2025-01-01T00:00:00Z"
  }
}`}</pre>
              </div>
            </div>
          </div>

          {/* GET /repositories/:id/branches */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <h3 className="text-lg font-semibold">
              <code className="bg-muted px-2 py-1 rounded text-blue-600 dark:text-blue-400">
                GET
              </code>{" "}
              /repositories/:id/branches
            </h3>
            <p className="text-muted-foreground">
              {t("docs.api.repositories.details.listBranches.description")}
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
                        {t(
                          "docs.api.repositories.details.listBranches.params.id"
                        )}
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
                        {t("docs.api.common.descriptionHeader")}
                      </th>
                    </tr>
                  </thead>
                  <tbody className="text-muted-foreground">
                    <tr>
                      <td className="p-3 font-mono text-xs">access_token</td>
                      <td className="p-3">string</td>
                      <td className="p-3">
                        <span className="text-xs bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.requiredBadge")}
                        </span>
                      </td>
                      <td className="p-3">
                        {t(
                          "docs.api.repositories.details.listBranches.params.access_token"
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
  "branches": [
    "main",
    "develop",
    "feature/auth",
    "fix/login-bug"
  ]
}`}</pre>
              </div>
            </div>
          </div>

          {/* GET /repositories/:id/merge-requests */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <h3 className="text-lg font-semibold">
              <code className="bg-muted px-2 py-1 rounded text-blue-600 dark:text-blue-400">
                GET
              </code>{" "}
              /repositories/:id/merge-requests
            </h3>
            <p className="text-muted-foreground">
              {t(
                "docs.api.repositories.details.listMergeRequests.description"
              )}
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
                        {t(
                          "docs.api.repositories.details.listMergeRequests.params.id"
                        )}
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
                        {t("docs.api.common.descriptionHeader")}
                      </th>
                    </tr>
                  </thead>
                  <tbody className="text-muted-foreground">
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
                        {t(
                          "docs.api.repositories.details.listMergeRequests.params.branch"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 font-mono text-xs">state</td>
                      <td className="p-3">string</td>
                      <td className="p-3">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3">
                        {t(
                          "docs.api.repositories.details.listMergeRequests.params.state"
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
  "merge_requests": [
    {
      "id": 101,
      "title": "Add JWT authentication",
      "state": "opened",
      "source_branch": "feature/auth",
      "target_branch": "main",
      "author": "john.doe",
      "url": "https://gitlab.com/org/repo/-/merge_requests/101",
      "created_at": "2025-01-14T09:00:00Z"
    }
  ]
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
