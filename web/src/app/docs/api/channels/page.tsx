"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function ApiChannelsPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.api.channels.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.api.channels.description")}
      </p>

      {/* Endpoints */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.api.channels.endpoints.title")}
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
                  /channels
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  channels:read
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.channels.endpoints.list")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-blue-600 dark:text-blue-400">
                    GET
                  </code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  /channels/:id
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  channels:read
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.channels.endpoints.get")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-blue-600 dark:text-blue-400">
                    GET
                  </code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  /channels/:id/messages
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  channels:read
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.channels.endpoints.messages")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-green-600 dark:text-green-400">
                    POST
                  </code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  /channels
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  channels:write
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.channels.endpoints.create")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  <code className="bg-muted px-1 rounded text-orange-600 dark:text-orange-400">
                    PUT
                  </code>
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  /channels/:id
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  channels:write
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.channels.endpoints.update")}
                </td>
              </tr>
              <tr>
                <td className="p-3">
                  <code className="bg-muted px-1 rounded text-green-600 dark:text-green-400">
                    POST
                  </code>
                </td>
                <td className="p-3 font-mono text-xs">
                  /channels/:id/messages
                </td>
                <td className="p-3 font-mono text-xs">channels:write</td>
                <td className="p-3">
                  {t("docs.api.channels.endpoints.sendMessage")}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Endpoint Details */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-6">
          {t("docs.api.channels.details.title")}
        </h2>
        <div className="space-y-8">
          {/* GET /channels */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <h3 className="text-lg font-semibold">
              <code className="bg-muted px-2 py-1 rounded text-blue-600 dark:text-blue-400">
                GET
              </code>{" "}
              /channels
            </h3>
            <p className="text-muted-foreground">
              {t("docs.api.channels.details.listChannels.description")}
            </p>

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
                          "docs.api.channels.details.listChannels.params.repository_id"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 border-b border-border font-mono text-xs">
                        ticket_id
                      </td>
                      <td className="p-3 border-b border-border">integer</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.channels.details.listChannels.params.ticket_id"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 font-mono text-xs">
                        include_archived
                      </td>
                      <td className="p-3">boolean</td>
                      <td className="p-3">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3">
                        {t(
                          "docs.api.channels.details.listChannels.params.include_archived"
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
  "channels": [
    {
      "id": 1,
      "name": "feature-auth",
      "description": "Authentication implementation channel",
      "repository_id": 1,
      "ticket_identifier": "AM-42",
      "archived": false,
      "created_at": "2025-01-10T08:00:00Z",
      "updated_at": "2025-01-15T14:20:00Z"
    }
  ],
  "total": 12
}`}</pre>
              </div>
            </div>
          </div>

          {/* GET /channels/:id */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <h3 className="text-lg font-semibold">
              <code className="bg-muted px-2 py-1 rounded text-blue-600 dark:text-blue-400">
                GET
              </code>{" "}
              /channels/:id
            </h3>
            <p className="text-muted-foreground">
              {t("docs.api.channels.details.getChannel.description")}
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
                          "docs.api.channels.details.getChannel.params.id"
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
  "channel": {
    "id": 1,
    "name": "feature-auth",
    "description": "Authentication implementation channel",
    "repository_id": 1,
    "ticket_identifier": "AM-42",
    "document": "## Context\\nImplement JWT auth...",
    "archived": false,
    "created_at": "2025-01-10T08:00:00Z",
    "updated_at": "2025-01-15T14:20:00Z"
  }
}`}</pre>
              </div>
            </div>
          </div>

          {/* GET /channels/:id/messages */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <h3 className="text-lg font-semibold">
              <code className="bg-muted px-2 py-1 rounded text-blue-600 dark:text-blue-400">
                GET
              </code>{" "}
              /channels/:id/messages
            </h3>
            <p className="text-muted-foreground">
              {t("docs.api.channels.details.getMessages.description")}
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
                          "docs.api.channels.details.getMessages.params.id"
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
                        {t("docs.api.common.defaultHeader")}
                      </th>
                      <th className="text-left p-3 border-b border-border">
                        {t("docs.api.common.descriptionHeader")}
                      </th>
                    </tr>
                  </thead>
                  <tbody className="text-muted-foreground">
                    <tr>
                      <td className="p-3 font-mono text-xs">limit</td>
                      <td className="p-3">integer</td>
                      <td className="p-3">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3">50</td>
                      <td className="p-3">
                        {t(
                          "docs.api.channels.details.getMessages.params.limit"
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
  "messages": [
    {
      "id": 100,
      "channel_id": 1,
      "content": "I've completed the JWT implementation.",
      "sender_type": "agent",
      "pod_key": "pod-abc123",
      "created_at": "2025-01-15T14:20:00Z"
    }
  ]
}`}</pre>
              </div>
            </div>
          </div>

          {/* POST /channels */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <h3 className="text-lg font-semibold">
              <code className="bg-muted px-2 py-1 rounded text-green-600 dark:text-green-400">
                POST
              </code>{" "}
              /channels
            </h3>
            <p className="text-muted-foreground">
              {t("docs.api.channels.details.createChannel.description")}
            </p>

            {/* Request Body */}
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
                        name
                      </td>
                      <td className="p-3 border-b border-border">string</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.requiredBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.channels.details.createChannel.fields.name"
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
                          "docs.api.channels.details.createChannel.fields.description"
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
                          "docs.api.channels.details.createChannel.fields.repository_id"
                        )}
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
                          "docs.api.channels.details.createChannel.fields.ticket_identifier"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 font-mono text-xs">document</td>
                      <td className="p-3">string</td>
                      <td className="p-3">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3">
                        {t(
                          "docs.api.channels.details.createChannel.fields.document"
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
  "channel": {
    "id": 1,
    "name": "feature-auth",
    "description": "Authentication implementation channel",
    "repository_id": 1,
    "ticket_identifier": "AM-42",
    "document": "## Context\\nImplement JWT auth...",
    "archived": false,
    "created_at": "2025-01-10T08:00:00Z",
    "updated_at": "2025-01-15T14:20:00Z"
  }
}`}</pre>
              </div>
            </div>
          </div>

          {/* PUT /channels/:id */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <h3 className="text-lg font-semibold">
              <code className="bg-muted px-2 py-1 rounded text-orange-600 dark:text-orange-400">
                PUT
              </code>{" "}
              /channels/:id
            </h3>
            <p className="text-muted-foreground">
              {t("docs.api.channels.details.updateChannel.description")}
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
                          "docs.api.channels.details.updateChannel.params.id"
                        )}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>

            {/* Request Body */}
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
                        name
                      </td>
                      <td className="p-3 border-b border-border">string</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.channels.details.updateChannel.fields.name"
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
                          "docs.api.channels.details.updateChannel.fields.description"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 font-mono text-xs">document</td>
                      <td className="p-3">string</td>
                      <td className="p-3">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3">
                        {t(
                          "docs.api.channels.details.updateChannel.fields.document"
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
  "channel": {
    "id": 1,
    "name": "feature-auth",
    "description": "Authentication implementation channel",
    "repository_id": 1,
    "ticket_identifier": "AM-42",
    "document": "## Context\\nImplement JWT auth...",
    "archived": false,
    "created_at": "2025-01-10T08:00:00Z",
    "updated_at": "2025-01-15T14:20:00Z"
  }
}`}</pre>
              </div>
            </div>
          </div>

          {/* POST /channels/:id/messages */}
          <div className="border border-border rounded-lg p-6 space-y-6">
            <h3 className="text-lg font-semibold">
              <code className="bg-muted px-2 py-1 rounded text-green-600 dark:text-green-400">
                POST
              </code>{" "}
              /channels/:id/messages
            </h3>
            <p className="text-muted-foreground">
              {t("docs.api.channels.details.sendMessage.description")}
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
                          "docs.api.channels.details.sendMessage.params.id"
                        )}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>

            {/* Request Body */}
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
                        content
                      </td>
                      <td className="p-3 border-b border-border">string</td>
                      <td className="p-3 border-b border-border">
                        <span className="text-xs bg-red-100 dark:bg-red-900/30 text-red-700 dark:text-red-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.requiredBadge")}
                        </span>
                      </td>
                      <td className="p-3 border-b border-border">
                        {t(
                          "docs.api.channels.details.sendMessage.fields.content"
                        )}
                      </td>
                    </tr>
                    <tr>
                      <td className="p-3 font-mono text-xs">pod_key</td>
                      <td className="p-3">string</td>
                      <td className="p-3">
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-400 px-2 py-0.5 rounded">
                          {t("docs.api.common.optionalBadge")}
                        </span>
                      </td>
                      <td className="p-3">
                        {t(
                          "docs.api.channels.details.sendMessage.fields.pod_key"
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
  "message": {
    "id": 101,
    "channel_id": 1,
    "content": "Please review the auth module.",
    "sender_type": "api",
    "pod_key": null,
    "created_at": "2025-01-15T15:00:00Z"
  }
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
