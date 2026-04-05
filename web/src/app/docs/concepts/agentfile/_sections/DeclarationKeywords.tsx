"use client";

import { useTranslations } from "next-intl";

export function DeclarationKeywords() {
  const t = useTranslations();

  return (
    <section className="mb-12">
      <h2 className="text-2xl font-semibold mb-4">
        {t("docs.concepts.agentfile.declarations.title")}
      </h2>
      <p className="text-muted-foreground leading-relaxed mb-6">
        {t("docs.concepts.agentfile.declarations.description")}
      </p>

      <div className="space-y-6">
        {/* AGENT */}
        <div className="border border-border rounded-lg p-4">
          <h3 className="font-medium mb-2 font-mono">AGENT</h3>
          <p className="text-sm text-muted-foreground mb-2">
            {t("docs.concepts.agentfile.declarations.agentDesc")}
          </p>
          <pre className="bg-muted rounded-lg p-3 text-sm overflow-x-auto">
            <code>{`AGENT "claude-code"
AGENT "aider --model opus"`}</code>
          </pre>
        </div>

        {/* EXECUTABLE */}
        <div className="border border-border rounded-lg p-4">
          <h3 className="font-medium mb-2 font-mono">EXECUTABLE</h3>
          <p className="text-sm text-muted-foreground mb-2">
            {t("docs.concepts.agentfile.declarations.executableDesc")}
          </p>
          <pre className="bg-muted rounded-lg p-3 text-sm overflow-x-auto">
            <code>{`EXECUTABLE "claude"`}</code>
          </pre>
        </div>

        {/* CONFIG */}
        <div className="border border-border rounded-lg p-4">
          <h3 className="font-medium mb-2 font-mono">CONFIG</h3>
          <p className="text-sm text-muted-foreground mb-2">
            {t("docs.concepts.agentfile.declarations.configDesc")}
          </p>
          <pre className="bg-muted rounded-lg p-3 text-sm overflow-x-auto">
            <code>{`CONFIG model STRING = "sonnet"
CONFIG verbose BOOL = false
CONFIG max_tokens NUMBER = 4096
CONFIG api_key SECRET
CONFIG mode SELECT("fast","balanced","quality") = "balanced"`}</code>
          </pre>
        </div>

        {/* ENV */}
        <div className="border border-border rounded-lg p-4">
          <h3 className="font-medium mb-2 font-mono">ENV</h3>
          <p className="text-sm text-muted-foreground mb-2">
            {t("docs.concepts.agentfile.declarations.envDesc")}
          </p>
          <pre className="bg-muted rounded-lg p-3 text-sm overflow-x-auto">
            <code>{`ENV ANTHROPIC_API_KEY SECRET
ENV DEBUG TEXT OPTIONAL
ENV NODE_ENV = "production"`}</code>
          </pre>
        </div>

        {/* MODE */}
        <div className="border border-border rounded-lg p-4">
          <h3 className="font-medium mb-2 font-mono">MODE</h3>
          <p className="text-sm text-muted-foreground mb-2">
            {t("docs.concepts.agentfile.declarations.modeDesc")}
          </p>
          <pre className="bg-muted rounded-lg p-3 text-sm overflow-x-auto">
            <code>{`MODE pty
MODE acp --model opus`}</code>
          </pre>
        </div>

        {/* CREDENTIAL */}
        <div className="border border-border rounded-lg p-4">
          <h3 className="font-medium mb-2 font-mono">CREDENTIAL</h3>
          <p className="text-sm text-muted-foreground mb-2">
            {t("docs.concepts.agentfile.declarations.credentialDesc")}
          </p>
          <pre className="bg-muted rounded-lg p-3 text-sm overflow-x-auto">
            <code>{`CREDENTIAL "my-anthropic-profile"
CREDENTIAL runner_host`}</code>
          </pre>
        </div>

        {/* PROMPT & PROMPT_POSITION */}
        <div className="border border-border rounded-lg p-4">
          <h3 className="font-medium mb-2 font-mono">
            PROMPT / PROMPT_POSITION
          </h3>
          <p className="text-sm text-muted-foreground mb-2">
            {t("docs.concepts.agentfile.declarations.promptDesc")}
          </p>
          <pre className="bg-muted rounded-lg p-3 text-sm overflow-x-auto">
            <code>{`PROMPT "Fix the login bug in auth.ts"
PROMPT_POSITION prepend  # prepend | append | none`}</code>
          </pre>
        </div>

        {/* REPO & BRANCH & GIT_CREDENTIAL */}
        <div className="border border-border rounded-lg p-4">
          <h3 className="font-medium mb-2 font-mono">
            REPO / BRANCH / GIT_CREDENTIAL
          </h3>
          <p className="text-sm text-muted-foreground mb-2">
            {t("docs.concepts.agentfile.declarations.repoDesc")}
          </p>
          <pre className="bg-muted rounded-lg p-3 text-sm overflow-x-auto">
            <code>{`REPO "my-org/my-repo"
BRANCH "feature/new-auth"
GIT_CREDENTIAL http  # http | ssh | token`}</code>
          </pre>
        </div>

        {/* MCP */}
        <div className="border border-border rounded-lg p-4">
          <h3 className="font-medium mb-2 font-mono">MCP</h3>
          <p className="text-sm text-muted-foreground mb-2">
            {t("docs.concepts.agentfile.declarations.mcpDesc")}
          </p>
          <pre className="bg-muted rounded-lg p-3 text-sm overflow-x-auto">
            <code>{`MCP ON
MCP ON FORMAT streamable-http
MCP OFF`}</code>
          </pre>
        </div>

        {/* SKILLS */}
        <div className="border border-border rounded-lg p-4">
          <h3 className="font-medium mb-2 font-mono">SKILLS</h3>
          <p className="text-sm text-muted-foreground mb-2">
            {t("docs.concepts.agentfile.declarations.skillsDesc")}
          </p>
          <pre className="bg-muted rounded-lg p-3 text-sm overflow-x-auto">
            <code>{`SKILLS code-review, deploy-helper`}</code>
          </pre>
        </div>

        {/* SETUP */}
        <div className="border border-border rounded-lg p-4">
          <h3 className="font-medium mb-2 font-mono">SETUP</h3>
          <p className="text-sm text-muted-foreground mb-2">
            {t("docs.concepts.agentfile.declarations.setupDesc")}
          </p>
          <pre className="bg-muted rounded-lg p-3 text-sm overflow-x-auto">
            <code>{`SETUP timeout=60 <<SCRIPT
npm install
npm run build
SCRIPT`}</code>
          </pre>
        </div>

        {/* REMOVE */}
        <div className="border border-border rounded-lg p-4">
          <h3 className="font-medium mb-2 font-mono">REMOVE</h3>
          <p className="text-sm text-muted-foreground mb-2">
            {t("docs.concepts.agentfile.declarations.removeDesc")}
          </p>
          <pre className="bg-muted rounded-lg p-3 text-sm overflow-x-auto">
            <code>{`REMOVE ENV DEBUG
REMOVE SKILLS deploy-helper
REMOVE arg --verbose
REMOVE file /tmp/cache`}</code>
          </pre>
        </div>
      </div>
    </section>
  );
}
