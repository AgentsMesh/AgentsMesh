"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function AgentfileLayerPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.concepts.agentfileLayer.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.concepts.agentfileLayer.description")}
      </p>

      {/* What is AgentFile Layer */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.concepts.agentfileLayer.whatIs.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.concepts.agentfileLayer.whatIs.description")}
        </p>
        <div className="border border-border rounded-lg p-4">
          <pre className="text-sm overflow-x-auto">
            <code>{`Base AgentFile  →  User Preferences  →  Layer (per-Pod override)
 (agent default)    (saved settings)     (runtime customization)`}</code>
          </pre>
        </div>
      </section>

      {/* Supported Declarations */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.concepts.agentfileLayer.supported.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.concepts.agentfileLayer.supported.description")}
        </p>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {[
            "MODE",
            "CREDENTIAL",
            "PROMPT",
            "PROMPT_POSITION",
            "CONFIG",
            "REPO",
            "BRANCH",
            "GIT_CREDENTIAL",
          ].map((keyword) => (
            <div
              key={keyword}
              className="border border-border rounded-lg p-3 font-mono text-sm"
            >
              {keyword}
            </div>
          ))}
        </div>
      </section>

      {/* API Usage */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.concepts.agentfileLayer.apiUsage.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.concepts.agentfileLayer.apiUsage.description")}
        </p>
        <pre className="bg-muted rounded-lg p-4 text-sm overflow-x-auto">
          <code>{`POST /api/v1/pods
{
  "agent_slug": "claude-code",
  "runner_id": "runner-abc",
  "agentfile_layer": "CONFIG model = \\"opus\\"\\nPROMPT \\"Fix the login bug\\""
}`}</code>
        </pre>
      </section>

      {/* Frontend Editor */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.concepts.agentfileLayer.editor.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed">
          {t("docs.concepts.agentfileLayer.editor.description")}
        </p>
      </section>

      {/* Examples */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.concepts.agentfileLayer.examples.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-6">
          {t("docs.concepts.agentfileLayer.examples.description")}
        </p>
        <div className="space-y-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.concepts.agentfileLayer.examples.overrideModel")}
            </h3>
            <pre className="bg-muted rounded-lg p-3 text-sm overflow-x-auto">
              <code>{`CONFIG model = "opus"`}</code>
            </pre>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.concepts.agentfileLayer.examples.permissionMode")}
            </h3>
            <pre className="bg-muted rounded-lg p-3 text-sm overflow-x-auto">
              <code>{`CONFIG permission_mode = "bypassPermissions"`}</code>
            </pre>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.concepts.agentfileLayer.examples.addPrompt")}
            </h3>
            <pre className="bg-muted rounded-lg p-3 text-sm overflow-x-auto">
              <code>{`PROMPT "Fix the login bug in auth.ts"`}</code>
            </pre>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.concepts.agentfileLayer.examples.selectRepo")}
            </h3>
            <pre className="bg-muted rounded-lg p-3 text-sm overflow-x-auto">
              <code>{`REPO "my-org/my-repo"
BRANCH "feature/xyz"`}</code>
            </pre>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.concepts.agentfileLayer.examples.switchMode")}
            </h3>
            <pre className="bg-muted rounded-lg p-3 text-sm overflow-x-auto">
              <code>{`MODE acp`}</code>
            </pre>
          </div>
        </div>
      </section>

      <DocNavigation />
    </div>
  );
}
