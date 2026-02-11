"use client";

import { useServerUrl } from "@/hooks/useServerUrl";
import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function RunnerSetupPage() {
  const serverUrl = useServerUrl();
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.runners.setup.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.runners.setup.description")}
      </p>

      {/* Requirements */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.runners.setup.requirements.title")}
        </h2>
        <ul className="list-disc list-inside text-muted-foreground space-y-2">
          <li>{t("docs.runners.setup.requirements.item1")}</li>
          <li>{t("docs.runners.setup.requirements.item2")}</li>
          <li>{t("docs.runners.setup.requirements.item3")}</li>
          <li>{t("docs.runners.setup.requirements.item4")}</li>
          <li>{t("docs.runners.setup.requirements.item5")}</li>
        </ul>
      </section>

      {/* Quick Install */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.runners.setup.quickInstall.title")}
        </h2>

        <h3 className="text-lg font-medium mb-2 mt-6">
          {t("docs.runners.setup.quickInstall.oneLineTitle")}
        </h3>
        <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto">
          <pre className="text-green-500 dark:text-green-400">{`# macOS / Linux
curl -fsSL ${serverUrl}/install.sh | sh

# Windows (PowerShell)
irm ${serverUrl}/install.ps1 | iex`}</pre>
        </div>

        <h3 className="text-lg font-medium mb-2 mt-6">
          {t("docs.runners.setup.quickInstall.brewTitle")}
        </h3>
        <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto">
          <pre className="text-green-500 dark:text-green-400">{`# Add AgentsMesh tap
brew tap agentsmesh/tap https://github.com/AgentsMesh/BrewCask

# Install runner
brew install agentsmesh/tap/agentsmesh-runner`}</pre>
        </div>

        <h3 className="text-lg font-medium mb-2 mt-6">
          {t("docs.runners.setup.quickInstall.linuxTitle")}
        </h3>
        <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto">
          <pre className="text-green-500 dark:text-green-400">{`# Debian/Ubuntu
VERSION=$(curl -s https://api.github.com/repos/AgentsMesh/AgentsMeshRunner/releases/latest | grep tag_name | cut -d '"' -f 4 | sed 's/v//')
wget https://github.com/AgentsMesh/AgentsMeshRunner/releases/download/v\${VERSION}/agentsmesh-runner_\${VERSION}_linux_amd64.deb
sudo dpkg -i agentsmesh-runner_\${VERSION}_linux_amd64.deb

# RHEL/CentOS/Fedora
VERSION=$(curl -s https://api.github.com/repos/AgentsMesh/AgentsMeshRunner/releases/latest | grep tag_name | cut -d '"' -f 4 | sed 's/v//')
wget https://github.com/AgentsMesh/AgentsMeshRunner/releases/download/v\${VERSION}/agentsmesh-runner_\${VERSION}_linux_amd64.rpm
sudo rpm -i agentsmesh-runner_\${VERSION}_linux_amd64.rpm`}</pre>
        </div>

        <h3 className="text-lg font-medium mb-2 mt-6">
          {t("docs.runners.setup.quickInstall.afterInstall")}
        </h3>
        <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto">
          <pre className="text-green-500 dark:text-green-400">{`# Register with your server
agentsmesh-runner register --server ${serverUrl} --token <YOUR_TOKEN>

# Start the runner
agentsmesh-runner run`}</pre>
        </div>

        <p className="text-sm text-muted-foreground mt-4">
          {t("docs.runners.setup.quickInstall.tokenHint")}
        </p>
      </section>

      {/* Docker Installation */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.runners.setup.docker.title")}
        </h2>
        <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto">
          <pre className="text-green-500 dark:text-green-400">{`# Run with Docker
docker run -d \\
  --name agentsmesh-runner \\
  -e AGENTSMESH_TOKEN=<YOUR_TOKEN> \\
  -e AGENTSMESH_URL=${serverUrl} \\
  -v /var/run/docker.sock:/var/run/docker.sock \\
  -v ~/.ssh:/root/.ssh:ro \\
  agentsmesh/runner:latest`}</pre>
        </div>
      </section>

      {/* Docker Compose */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.runners.setup.dockerCompose.title")}
        </h2>
        <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto">
          <pre className="text-green-500 dark:text-green-400">{`# docker-compose.yml
version: '3.8'
services:
  runner:
    image: agentsmesh/runner:latest
    container_name: agentsmesh-runner
    restart: unless-stopped
    environment:
      - AGENTSMESH_TOKEN=\${AGENTSMESH_TOKEN}
      - AGENTSMESH_URL=\${AGENTSMESH_URL:-${serverUrl}}
      - MAX_CONCURRENT_PODS=5
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - ~/.ssh:/root/.ssh:ro
      - runner-data:/data
volumes:
  runner-data:`}</pre>
        </div>
      </section>

      {/* Environment Variables */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.runners.setup.envVars.title")}
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.setup.envVars.variableHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.setup.envVars.descriptionHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.runners.setup.envVars.defaultHeader")}
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border font-mono text-xs">
                  AGENTSMESH_TOKEN
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.setup.envVars.tokenDesc")}
                </td>
                <td className="p-3 border-b border-border">-</td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-mono text-xs">
                  AGENTSMESH_URL
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.setup.envVars.urlDesc")}
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  {serverUrl}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-mono text-xs">
                  MAX_CONCURRENT_PODS
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.setup.envVars.maxPodsDesc")}
                </td>
                <td className="p-3 border-b border-border">5</td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-mono text-xs">
                  WORKSPACE_DIR
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.runners.setup.envVars.workspaceDirDesc")}
                </td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  /data/workspaces
                </td>
              </tr>
              <tr>
                <td className="p-3 font-mono text-xs">MCP_PORT</td>
                <td className="p-3">
                  {t("docs.runners.setup.envVars.mcpPortDesc")}
                </td>
                <td className="p-3">19000</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Registration Token */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.runners.setup.registrationToken.title")}
        </h2>
        <ol className="list-decimal list-inside text-muted-foreground space-y-2">
          <li>{t("docs.runners.setup.registrationToken.step1")}</li>
          <li>{t("docs.runners.setup.registrationToken.step2")}</li>
          <li>{t("docs.runners.setup.registrationToken.step3")}</li>
          <li>{t("docs.runners.setup.registrationToken.step4")}</li>
          <li>{t("docs.runners.setup.registrationToken.step5")}</li>
        </ol>
        <div className="bg-muted rounded-lg p-4 mt-4">
          <p className="text-sm text-muted-foreground">
            {t("docs.runners.setup.registrationToken.securityNote")}
          </p>
        </div>
      </section>

      {/* Verifying Installation */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.runners.setup.verifying.title")}
        </h2>
        <p className="text-muted-foreground mb-4">
          {t("docs.runners.setup.verifying.description")}
        </p>
        <ol className="list-decimal list-inside text-muted-foreground space-y-2">
          <li>{t("docs.runners.setup.verifying.step1")}</li>
          <li>{t("docs.runners.setup.verifying.step2")}</li>
          <li>{t("docs.runners.setup.verifying.step3")}</li>
        </ol>
      </section>

      {/* Troubleshooting */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.runners.setup.troubleshooting.title")}
        </h2>
        <div className="space-y-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.runners.setup.troubleshooting.offline")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.runners.setup.troubleshooting.offlineDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.runners.setup.troubleshooting.podsFail")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.runners.setup.troubleshooting.podsFailDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.runners.setup.troubleshooting.tokenInvalid")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.runners.setup.troubleshooting.tokenInvalidDesc")}
            </p>
          </div>
        </div>
      </section>

      <DocNavigation />
    </div>
  );
}
