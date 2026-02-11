"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function ChannelsPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.features.channels.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.features.channels.description")}
      </p>

      {/* Overview */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.channels.overview.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.features.channels.overview.description")}
        </p>
        <ul className="list-disc list-inside text-muted-foreground space-y-2">
          <li>{t("docs.features.channels.overview.item1")}</li>
          <li>{t("docs.features.channels.overview.item2")}</li>
          <li>{t("docs.features.channels.overview.item3")}</li>
          <li>{t("docs.features.channels.overview.item4")}</li>
        </ul>
      </section>

      {/* Creating Channels */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.channels.creating.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.features.channels.creating.description")}
        </p>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">
                  {t("docs.features.channels.creating.fieldHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.features.channels.creating.descriptionHeader")}
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.features.channels.creating.name")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.features.channels.creating.nameDesc")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.features.channels.creating.channelDescription")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.features.channels.creating.channelDescriptionDesc")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.features.channels.creating.projectId")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.features.channels.creating.projectIdDesc")}
                </td>
              </tr>
              <tr>
                <td className="p-3 font-medium">
                  {t("docs.features.channels.creating.ticketId")}
                </td>
                <td className="p-3">
                  {t("docs.features.channels.creating.ticketIdDesc")}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Message Types */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.channels.messageTypes.title")}
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              <code className="bg-muted px-1 rounded">
                {t("docs.features.channels.messageTypes.text")}
              </code>
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.channels.messageTypes.textDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              <code className="bg-muted px-1 rounded">
                {t("docs.features.channels.messageTypes.system")}
              </code>
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.channels.messageTypes.systemDesc")}
            </p>
          </div>
        </div>
      </section>

      {/* Mentions */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.channels.mentions.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.features.channels.mentions.description")}
        </p>
        <div className="bg-muted rounded-lg p-4 font-mono text-sm">
          <pre className="text-green-500 dark:text-green-400">{`// Send a message with mentions
send_channel_message({
  channel_id: 123,
  content: "Can you review this implementation?",
  message_type: "text",
  mentions: ["pod-abc", "pod-xyz"]
})`}</pre>
        </div>
        <p className="text-sm text-muted-foreground mt-4">
          {t("docs.features.channels.mentions.hint", {
            param: "mentioned_pod",
          })}
        </p>
      </section>

      {/* Shared Documents */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.channels.sharedDocs.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.features.channels.sharedDocs.description")}
        </p>
        <div className="space-y-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.channels.sharedDocs.getDoc")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.channels.sharedDocs.getDocDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.channels.sharedDocs.updateDoc")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.channels.sharedDocs.updateDocDesc")}
            </p>
          </div>
        </div>
        <p className="text-sm text-muted-foreground mt-4">
          {t("docs.features.channels.sharedDocs.hint")}
        </p>
      </section>

      {/* MCP Tools */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.channels.mcpTools.title")}
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">
                  {t("docs.features.channels.mcpTools.toolHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.features.channels.mcpTools.descriptionHeader")}
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.features.channels.mcpTools.searchChannels")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.features.channels.mcpTools.searchChannelsDesc")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.features.channels.mcpTools.createChannel")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.features.channels.mcpTools.createChannelDesc")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.features.channels.mcpTools.getChannel")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.features.channels.mcpTools.getChannelDesc")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.features.channels.mcpTools.sendMessage")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.features.channels.mcpTools.sendMessageDesc")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.features.channels.mcpTools.getMessages")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.features.channels.mcpTools.getMessagesDesc")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.features.channels.mcpTools.getDocument")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.features.channels.mcpTools.getDocumentDesc")}
                </td>
              </tr>
              <tr>
                <td className="p-3 font-medium">
                  {t("docs.features.channels.mcpTools.updateDocument")}
                </td>
                <td className="p-3">
                  {t("docs.features.channels.mcpTools.updateDocumentDesc")}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Use Cases */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.channels.useCases.title")}
        </h2>
        <div className="space-y-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.channels.useCases.coordination")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.channels.useCases.coordinationDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.channels.useCases.designDocs")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.channels.useCases.designDocsDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.channels.useCases.notifications")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.channels.useCases.notificationsDesc")}
            </p>
          </div>
        </div>
      </section>

      {/* Web UI */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.channels.webUI.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.features.channels.webUI.description")}
        </p>
        <ol className="list-decimal list-inside text-muted-foreground space-y-2">
          <li>{t("docs.features.channels.webUI.step1")}</li>
          <li>{t("docs.features.channels.webUI.step2")}</li>
          <li>{t("docs.features.channels.webUI.step3")}</li>
          <li>{t("docs.features.channels.webUI.step4")}</li>
          <li>{t("docs.features.channels.webUI.step5")}</li>
        </ol>
      </section>

      <DocNavigation />
    </div>
  );
}
