"use client";

import Link from "next/link";
import { useTranslations } from "next-intl";

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

      {/* What You Can Do */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.whatYouCanDo.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.whatYouCanDo.description")}
        </p>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {(["orchestrate", "remoteWorkstation", "taskDriven", "selfHosted"] as const).map((key) => (
            <div key={key} className="border border-border rounded-lg p-4">
              <h3 className="font-medium mb-1">{t(`docs.whatYouCanDo.${key}.title`)}</h3>
              <p className="text-sm text-muted-foreground">
                {t(`docs.whatYouCanDo.${key}.description`)}
              </p>
            </div>
          ))}
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
