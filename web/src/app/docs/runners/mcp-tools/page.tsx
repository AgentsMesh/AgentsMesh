"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function MCPToolsPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.runners.mcpTools.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.runners.mcpTools.description")}
      </p>

      {/* Overview */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.runners.mcpTools.overview.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.runners.mcpTools.overview.description")}
        </p>
        <div className="bg-muted rounded-lg p-4">
          <p className="text-sm text-muted-foreground">
            <strong>{t("docs.runners.mcpTools.overview.autoConfig")}</strong>
          </p>
        </div>
      </section>

      {/* Discovery Tools */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.runners.mcpTools.discovery.title")}
        </h2>
        <p className="text-muted-foreground mb-4">
          {t("docs.runners.mcpTools.discovery.description")}
        </p>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.mcpTools.discovery.toolHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.mcpTools.discovery.descriptionHeader")}
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.runners.mcpTools.discovery.listPods")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.mcpTools.discovery.listPodsDesc")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.runners.mcpTools.discovery.listRunners")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.mcpTools.discovery.listRunnersDesc")}
                </td>
              </tr>
              <tr>
                <td className="p-3 font-medium">
                  {t("docs.runners.mcpTools.discovery.listRepos")}
                </td>
                <td className="p-3">
                  {t("docs.runners.mcpTools.discovery.listReposDesc")}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Terminal Tools */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.runners.mcpTools.terminal.title")}
        </h2>
        <p className="text-muted-foreground mb-4">
          {t("docs.runners.mcpTools.terminal.description")}
        </p>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.mcpTools.terminal.toolHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.mcpTools.terminal.descriptionHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.mcpTools.terminal.scopeHeader")}
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.runners.mcpTools.terminal.observe")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.mcpTools.terminal.observeDesc")}
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  {t("docs.runners.mcpTools.terminal.observeScope")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.runners.mcpTools.terminal.sendText")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.mcpTools.terminal.sendTextDesc")}
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  {t("docs.runners.mcpTools.terminal.sendTextScope")}
                </td>
              </tr>
              <tr>
                <td className="p-3 font-medium">
                  {t("docs.runners.mcpTools.terminal.sendKey")}
                </td>
                <td className="p-3">
                  {t("docs.runners.mcpTools.terminal.sendKeyDesc")}
                </td>
                <td className="p-3 font-mono text-xs">
                  {t("docs.runners.mcpTools.terminal.sendKeyScope")}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Pod Binding Tools */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.runners.mcpTools.binding.title")}
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.mcpTools.binding.toolHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.mcpTools.binding.descriptionHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.mcpTools.binding.paramsHeader")}
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.runners.mcpTools.binding.bindPod")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.mcpTools.binding.bindPodDesc")}
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  {t("docs.runners.mcpTools.binding.bindPodParams")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.runners.mcpTools.binding.acceptBinding")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.mcpTools.binding.acceptBindingDesc")}
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  {t("docs.runners.mcpTools.binding.acceptBindingParams")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.runners.mcpTools.binding.rejectBinding")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.mcpTools.binding.rejectBindingDesc")}
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  {t("docs.runners.mcpTools.binding.rejectBindingParams")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.runners.mcpTools.binding.unbindPod")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.mcpTools.binding.unbindPodDesc")}
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  {t("docs.runners.mcpTools.binding.unbindPodParams")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.runners.mcpTools.binding.getBindings")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.mcpTools.binding.getBindingsDesc")}
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  {t("docs.runners.mcpTools.binding.getBindingsParams")}
                </td>
              </tr>
              <tr>
                <td className="p-3 font-medium">
                  {t("docs.runners.mcpTools.binding.getBoundPods")}
                </td>
                <td className="p-3">
                  {t("docs.runners.mcpTools.binding.getBoundPodsDesc")}
                </td>
                <td className="p-3 font-mono text-xs">
                  {t("docs.runners.mcpTools.binding.getBoundPodsParams")}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Channel Tools */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.runners.mcpTools.channel.title")}
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.mcpTools.channel.toolHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.mcpTools.channel.descriptionHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.mcpTools.channel.paramsHeader")}
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.runners.mcpTools.channel.searchChannels")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.mcpTools.channel.searchChannelsDesc")}
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  {t("docs.runners.mcpTools.channel.searchChannelsParams")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.runners.mcpTools.channel.createChannel")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.mcpTools.channel.createChannelDesc")}
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  {t("docs.runners.mcpTools.channel.createChannelParams")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.runners.mcpTools.channel.getChannel")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.mcpTools.channel.getChannelDesc")}
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  {t("docs.runners.mcpTools.channel.getChannelParams")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.runners.mcpTools.channel.sendMessage")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.mcpTools.channel.sendMessageDesc")}
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  {t("docs.runners.mcpTools.channel.sendMessageParams")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.runners.mcpTools.channel.getMessages")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.mcpTools.channel.getMessagesDesc")}
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  {t("docs.runners.mcpTools.channel.getMessagesParams")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.runners.mcpTools.channel.getDocument")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.mcpTools.channel.getDocumentDesc")}
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  {t("docs.runners.mcpTools.channel.getDocumentParams")}
                </td>
              </tr>
              <tr>
                <td className="p-3 font-medium">
                  {t("docs.runners.mcpTools.channel.updateDocument")}
                </td>
                <td className="p-3">
                  {t("docs.runners.mcpTools.channel.updateDocumentDesc")}
                </td>
                <td className="p-3 font-mono text-xs">
                  {t("docs.runners.mcpTools.channel.updateDocumentParams")}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Ticket Tools */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.runners.mcpTools.ticket.title")}
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.mcpTools.ticket.toolHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.mcpTools.ticket.descriptionHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.mcpTools.ticket.paramsHeader")}
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.runners.mcpTools.ticket.searchTickets")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.mcpTools.ticket.searchTicketsDesc")}
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  {t("docs.runners.mcpTools.ticket.searchTicketsParams")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.runners.mcpTools.ticket.getTicket")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.mcpTools.ticket.getTicketDesc")}
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  {t("docs.runners.mcpTools.ticket.getTicketParams")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium">
                  {t("docs.runners.mcpTools.ticket.createTicket")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.mcpTools.ticket.createTicketDesc")}
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  {t("docs.runners.mcpTools.ticket.createTicketParams")}
                </td>
              </tr>
              <tr>
                <td className="p-3 font-medium">
                  {t("docs.runners.mcpTools.ticket.updateTicket")}
                </td>
                <td className="p-3">
                  {t("docs.runners.mcpTools.ticket.updateTicketDesc")}
                </td>
                <td className="p-3 font-mono text-xs">
                  {t("docs.runners.mcpTools.ticket.updateTicketParams")}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Pod Tools */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.runners.mcpTools.pod.title")}
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.mcpTools.pod.toolHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.mcpTools.pod.descriptionHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.mcpTools.pod.paramsHeader")}
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 font-medium">
                  {t("docs.runners.mcpTools.pod.createPod")}
                </td>
                <td className="p-3">
                  {t("docs.runners.mcpTools.pod.createPodDesc")}
                </td>
                <td className="p-3 font-mono text-xs">
                  {t("docs.runners.mcpTools.pod.createPodParams")}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <DocNavigation />
    </div>
  );
}
