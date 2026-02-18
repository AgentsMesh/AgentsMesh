"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function ApiTicketsPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.api.tickets.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.api.tickets.description")}
      </p>

      {/* Endpoints */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.api.tickets.endpoints.title")}
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
                  /tickets
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  tickets:read
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.tickets.endpoints.list")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-blue-600 dark:text-blue-400">
                    GET
                  </code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  /tickets/board
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  tickets:read
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.tickets.endpoints.board")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-blue-600 dark:text-blue-400">
                    GET
                  </code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  /tickets/:identifier
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  tickets:read
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.tickets.endpoints.get")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-green-600 dark:text-green-400">
                    POST
                  </code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  /tickets
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  tickets:write
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.tickets.endpoints.create")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-orange-600 dark:text-orange-400">
                    PUT
                  </code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  /tickets/:identifier
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  tickets:write
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.tickets.endpoints.update")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-orange-600 dark:text-orange-400">
                    PATCH
                  </code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  /tickets/:identifier/status
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  tickets:write
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.tickets.endpoints.updateStatus")}
                </td>
              </tr>
              <tr>
                <td className="p-3">
                  <code className="bg-muted px-1 rounded text-red-600 dark:text-red-400">
                    DELETE
                  </code>
                </td>
                <td className="p-3 font-mono text-xs">
                  /tickets/:identifier
                </td>
                <td className="p-3 font-mono text-xs">tickets:write</td>
                <td className="p-3">
                  {t("docs.api.tickets.endpoints.delete")}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Code Examples */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.api.tickets.examples.title")}
        </h2>
        <div className="space-y-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-3">
              {t("docs.api.tickets.examples.listTickets")}
            </h3>
            <div className="bg-muted rounded-lg p-4 font-mono text-sm">
              <pre className="text-green-500 dark:text-green-400">{`curl -X GET \\
  "https://your-domain.com/api/v1/ext/orgs/my-org/tickets" \\
  -H "X-API-Key: amk_your_api_key_here"`}</pre>
            </div>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-3">
              {t("docs.api.tickets.examples.createTicket")}
            </h3>
            <div className="bg-muted rounded-lg p-4 font-mono text-sm">
              <pre className="text-green-500 dark:text-green-400">{`curl -X POST \\
  "https://your-domain.com/api/v1/ext/orgs/my-org/tickets" \\
  -H "X-API-Key: amk_your_api_key_here" \\
  -H "Content-Type: application/json" \\
  -d '{
    "title": "Implement user auth",
    "description": "Add JWT authentication to the API",
    "type": "feature",
    "priority": "high"
  }'`}</pre>
            </div>
          </div>
        </div>
      </section>

      {/* Endpoint Details */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-6">
          {t("docs.api.tickets.details.title")}
        </h2>
        <div className="space-y-8">
          {/* GET /tickets */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <div>
              <h3 className="text-lg font-semibold mb-2">
                <code className="bg-muted px-2 py-1 rounded text-blue-600 dark:text-blue-400">
                  GET
                </code>{" "}
                <code className="text-sm">/tickets</code>
              </h3>
              <p className="text-muted-foreground text-sm">
                {t("docs.api.tickets.details.listTickets.description")}
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
                        repository_id
                      </td>
                      <td className="p-3 border-b border-border">integer</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">-</td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.tickets.details.listTickets.params.repository_id"
                        )}
                      </td>
                    </tr>
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
                          "docs.api.tickets.details.listTickets.params.status"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        type
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
                          "docs.api.tickets.details.listTickets.params.type"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        assignee_id
                      </td>
                      <td className="p-3 border-b border-border">integer</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">-</td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.tickets.details.listTickets.params.assignee_id"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        labels
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
                          "docs.api.tickets.details.listTickets.params.labels"
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
                      <td className="p-3 border-b border-border">20</td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.tickets.details.listTickets.params.limit"
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
                          "docs.api.tickets.details.listTickets.params.offset"
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
  "tickets": [
    {
      "id": 1,
      "identifier": "AM-42",
      "type": "feature",
      "title": "Implement user authentication",
      "description": "Add JWT authentication to the API",
      "status": "in_progress",
      "priority": "high",
      "assignee_id": 10,
      "repository_id": 1,
      "labels": ["backend", "auth"],
      "created_at": "2025-01-10T08:00:00Z",
      "updated_at": "2025-01-15T14:20:00Z"
    }
  ],
  "total": 156,
  "limit": 20,
  "offset": 0
}`}</pre>
              </div>
            </div>
          </div>

          {/* GET /tickets/board */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <div>
              <h3 className="text-lg font-semibold mb-2">
                <code className="bg-muted px-2 py-1 rounded text-blue-600 dark:text-blue-400">
                  GET
                </code>{" "}
                <code className="text-sm">/tickets/board</code>
              </h3>
              <p className="text-muted-foreground text-sm">
                {t("docs.api.tickets.details.getBoard.description")}
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
                        {t("docs.api.common.descriptionHeader")}
                      </th>
                    </tr>
                  </thead>
                  <tbody className="text-muted-foreground">
                    <tr>
                      <td className="p-3 font-mono text-xs">repository_id</td>
                      <td className="p-3">integer</td>
                      <td className="p-3">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3">
                        {t(
                          "docs.api.tickets.details.getBoard.params.repository_id"
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
  "board": {
    "columns": [
      {
        "status": "open",
        "tickets": [
          {
            "id": 2,
            "identifier": "AM-43",
            "title": "Fix CSS layout issue",
            "type": "bug",
            "priority": "medium",
            "assignee_id": 5
          }
        ]
      },
      {
        "status": "in_progress",
        "tickets": []
      }
    ]
  }
}`}</pre>
              </div>
            </div>
          </div>

          {/* GET /tickets/:identifier */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <div>
              <h3 className="text-lg font-semibold mb-2">
                <code className="bg-muted px-2 py-1 rounded text-blue-600 dark:text-blue-400">
                  GET
                </code>{" "}
                <code className="text-sm">/tickets/:identifier</code>
              </h3>
              <p className="text-muted-foreground text-sm">
                {t("docs.api.tickets.details.getTicket.description")}
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
                      <td className="p-3 font-mono text-xs">identifier</td>
                      <td className="p-3">string</td>
                      <td className="p-3">
                        <span className="text-xs bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.requiredBadge")}
                        </span>
                      </td>
                      <td className="p-3">
                        {t(
                          "docs.api.tickets.details.getTicket.params.identifier"
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
  "ticket": {
    "id": 1,
    "identifier": "AM-42",
    "type": "feature",
    "title": "Implement user authentication",
    "description": "Add JWT authentication to the API",
    "status": "in_progress",
    "priority": "high",
    "assignee_id": 10,
    "repository_id": 1,
    "labels": ["backend", "auth"],
    "parent_identifier": null,
    "created_at": "2025-01-10T08:00:00Z",
    "updated_at": "2025-01-15T14:20:00Z"
  }
}`}</pre>
              </div>
            </div>
          </div>

          {/* POST /tickets */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <div>
              <h3 className="text-lg font-semibold mb-2">
                <code className="bg-muted px-2 py-1 rounded text-green-600 dark:text-green-400">
                  POST
                </code>{" "}
                <code className="text-sm">/tickets</code>
              </h3>
              <p className="text-muted-foreground text-sm">
                {t("docs.api.tickets.details.createTicket.description")}
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
                        type
                      </td>
                      <td className="p-3 border-b border-border">string</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.requiredBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.tickets.details.createTicket.fields.type"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        title
                      </td>
                      <td className="p-3 border-b border-border">string</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.requiredBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.tickets.details.createTicket.fields.title"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        description
                      </td>
                      <td className="p-3 border-b border-border">string</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.tickets.details.createTicket.fields.description"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        priority
                      </td>
                      <td className="p-3 border-b border-border">string</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.tickets.details.createTicket.fields.priority"
                        )}
                      </td>
                    </tr>
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
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.tickets.details.createTicket.fields.status"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        assignee_id
                      </td>
                      <td className="p-3 border-b border-border">integer</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.tickets.details.createTicket.fields.assignee_id"
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
                          "docs.api.tickets.details.createTicket.fields.repository_id"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        labels
                      </td>
                      <td className="p-3 border-b border-border">string[]</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.tickets.details.createTicket.fields.labels"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 font-mono text-xs">
                        parent_identifier
                      </td>
                      <td className="p-3">string</td>
                      <td className="p-3">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3">
                        {t(
                          "docs.api.tickets.details.createTicket.fields.parent_identifier"
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
  "ticket": {
    "id": 1,
    "identifier": "AM-42",
    "type": "feature",
    "title": "Implement user authentication",
    "description": "Add JWT authentication to the API",
    "status": "in_progress",
    "priority": "high",
    "assignee_id": 10,
    "repository_id": 1,
    "labels": ["backend", "auth"],
    "parent_identifier": null,
    "created_at": "2025-01-10T08:00:00Z",
    "updated_at": "2025-01-15T14:20:00Z"
  }
}`}</pre>
              </div>
            </div>
          </div>

          {/* PUT /tickets/:identifier */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <div>
              <h3 className="text-lg font-semibold mb-2">
                <code className="bg-muted px-2 py-1 rounded text-orange-600 dark:text-orange-400">
                  PUT
                </code>{" "}
                <code className="text-sm">/tickets/:identifier</code>
              </h3>
              <p className="text-muted-foreground text-sm">
                {t("docs.api.tickets.details.updateTicket.description")}
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
                      <td className="p-3 font-mono text-xs">identifier</td>
                      <td className="p-3">string</td>
                      <td className="p-3">
                        <span className="text-xs bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.requiredBadge")}
                        </span>
                      </td>
                      <td className="p-3">
                        {t(
                          "docs.api.tickets.details.updateTicket.params.identifier"
                        )}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
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
                        title
                      </td>
                      <td className="p-3 border-b border-border">string</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.tickets.details.updateTicket.fields.title"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        description
                      </td>
                      <td className="p-3 border-b border-border">string</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.tickets.details.updateTicket.fields.description"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        type
                      </td>
                      <td className="p-3 border-b border-border">string</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.tickets.details.updateTicket.fields.type"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        priority
                      </td>
                      <td className="p-3 border-b border-border">string</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.tickets.details.updateTicket.fields.priority"
                        )}
                      </td>
                    </tr>
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
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.tickets.details.updateTicket.fields.status"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        assignee_id
                      </td>
                      <td className="p-3 border-b border-border">integer</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.tickets.details.updateTicket.fields.assignee_id"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        labels
                      </td>
                      <td className="p-3 border-b border-border">string[]</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.tickets.details.updateTicket.fields.labels"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 font-mono text-xs">
                        parent_identifier
                      </td>
                      <td className="p-3">string</td>
                      <td className="p-3">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3">
                        {t(
                          "docs.api.tickets.details.updateTicket.fields.parent_identifier"
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
  "ticket": {
    "id": 1,
    "identifier": "AM-42",
    "type": "feature",
    "title": "Implement user authentication",
    "description": "Add JWT authentication to the API",
    "status": "in_progress",
    "priority": "high",
    "assignee_id": 10,
    "repository_id": 1,
    "labels": ["backend", "auth"],
    "parent_identifier": null,
    "created_at": "2025-01-10T08:00:00Z",
    "updated_at": "2025-01-15T14:20:00Z"
  }
}`}</pre>
              </div>
            </div>
          </div>

          {/* PATCH /tickets/:identifier/status */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <div>
              <h3 className="text-lg font-semibold mb-2">
                <code className="bg-muted px-2 py-1 rounded text-orange-600 dark:text-orange-400">
                  PATCH
                </code>{" "}
                <code className="text-sm">/tickets/:identifier/status</code>
              </h3>
              <p className="text-muted-foreground text-sm">
                {t("docs.api.tickets.details.updateStatus.description")}
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
                      <td className="p-3 font-mono text-xs">identifier</td>
                      <td className="p-3">string</td>
                      <td className="p-3">
                        <span className="text-xs bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.requiredBadge")}
                        </span>
                      </td>
                      <td className="p-3">
                        {t(
                          "docs.api.tickets.details.updateStatus.params.identifier"
                        )}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
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
                      <td className="p-3 font-mono text-xs">status</td>
                      <td className="p-3">string</td>
                      <td className="p-3">
                        <span className="text-xs bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.requiredBadge")}
                        </span>
                      </td>
                      <td className="p-3">
                        {t(
                          "docs.api.tickets.details.updateStatus.fields.status"
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
  "message": "Status updated"
}`}</pre>
              </div>
            </div>
          </div>

          {/* DELETE /tickets/:identifier */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <div>
              <h3 className="text-lg font-semibold mb-2">
                <code className="bg-muted px-2 py-1 rounded text-red-600 dark:text-red-400">
                  DELETE
                </code>{" "}
                <code className="text-sm">/tickets/:identifier</code>
              </h3>
              <p className="text-muted-foreground text-sm">
                {t("docs.api.tickets.details.deleteTicket.description")}
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
                      <td className="p-3 font-mono text-xs">identifier</td>
                      <td className="p-3">string</td>
                      <td className="p-3">
                        <span className="text-xs bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.requiredBadge")}
                        </span>
                      </td>
                      <td className="p-3">
                        {t(
                          "docs.api.tickets.details.deleteTicket.params.identifier"
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
  "message": "Ticket deleted"
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
