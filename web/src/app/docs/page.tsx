"use client";

import Link from "next/link";
import { useTranslations } from "@/lib/i18n/client";

export default function DocsPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">{t("docs.title")}</h1>

      {/* Introduction */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">{t("docs.intro.title")}</h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.intro.description")}
        </p>
        <div className="bg-muted rounded-lg p-4 mt-4">
          <p className="font-medium mb-2">{t("docs.intro.supportedAgents")}</p>
          <ul className="list-disc list-inside text-muted-foreground space-y-1">
            <li>Claude Code (Anthropic)</li>
            <li>Codex CLI (OpenAI)</li>
            <li>Gemini CLI (Google)</li>
            <li>Aider</li>
            <li>OpenCode</li>
            <li>{t("docs.intro.customAgents")}</li>
          </ul>
        </div>
      </section>

      {/* Why Terminal-based Agents */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.whyTerminal.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.whyTerminal.description")}
        </p>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">
                  {t("docs.whyTerminal.feature")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.whyTerminal.idePlugins")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.whyTerminal.terminalAgents")}
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border">
                  {t("docs.whyTerminal.autonomy")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.whyTerminal.autonomyIde")}
                </td>
                <td className="p-3 border-b border-border text-green-400">
                  ✓ {t("docs.whyTerminal.autonomyTerminal")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  {t("docs.whyTerminal.capabilities")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.whyTerminal.capabilitiesIde")}
                </td>
                <td className="p-3 border-b border-border text-green-400">
                  ✓ {t("docs.whyTerminal.capabilitiesTerminal")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  {t("docs.whyTerminal.environment")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.whyTerminal.environmentIde")}
                </td>
                <td className="p-3 border-b border-border text-green-400">
                  ✓ {t("docs.whyTerminal.environmentTerminal")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">
                  {t("docs.whyTerminal.multiAgent")}
                </td>
                <td className="p-3 border-b border-border">✗</td>
                <td className="p-3 border-b border-border text-green-400">
                  ✓ {t("docs.whyTerminal.multiAgentTerminal")}
                </td>
              </tr>
              <tr>
                <td className="p-3">{t("docs.whyTerminal.selfHosted")}</td>
                <td className="p-3">✗</td>
                <td className="p-3 text-green-400">
                  ✓ {t("docs.whyTerminal.selfHostedTerminal")}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Quick Links */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">{t("docs.quickLinks.title")}</h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Link
            href="/docs/getting-started"
            className="border border-border rounded-lg p-4 hover:border-primary transition-colors"
          >
            <h3 className="font-medium mb-1">{t("docs.quickLinks.quickStart")} →</h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.quickLinks.quickStartDesc")}
            </p>
          </Link>
          <Link
            href="/docs/features/agentpod"
            className="border border-border rounded-lg p-4 hover:border-primary transition-colors"
          >
            <h3 className="font-medium mb-1">AgentPod →</h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.quickLinks.agentpodDesc")}
            </p>
          </Link>
          <Link
            href="/docs/features/channels"
            className="border border-border rounded-lg p-4 hover:border-primary transition-colors"
          >
            <h3 className="font-medium mb-1">AgentsMesh →</h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.quickLinks.agentsmeshDesc")}
            </p>
          </Link>
          <Link
            href="/docs/runners/mcp-tools"
            className="border border-border rounded-lg p-4 hover:border-primary transition-colors"
          >
            <h3 className="font-medium mb-1">{t("docs.quickLinks.mcpTools")} →</h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.quickLinks.mcpToolsDesc")}
            </p>
          </Link>
        </div>
      </section>
    </div>
  );
}
