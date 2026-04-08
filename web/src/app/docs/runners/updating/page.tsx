"use client";

import { useServerUrl } from "@/hooks/useServerUrl";
import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function RunnerUpdatingPage() {
  const serverUrl = useServerUrl();
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.runners.updating.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.runners.updating.description")}
      </p>

      {/* Update Command */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.runners.updating.updateCommand.title")}
        </h2>
        <p className="text-muted-foreground mb-4">
          {t("docs.runners.updating.updateCommand.description")}
        </p>
        <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto">
          <pre className="text-green-500 dark:text-green-400">{`agentsmesh-runner update              # Interactive update
agentsmesh-runner update --check      # Only check for updates
agentsmesh-runner update -y           # Silent update (skip confirmation)
agentsmesh-runner update -f           # Force immediate update
agentsmesh-runner update -v v1.2.3    # Update to specific version
agentsmesh-runner update --pre        # Include prerelease versions`}</pre>
        </div>
        <div className="bg-muted/50 border border-border rounded-lg p-4 mt-4 text-sm text-muted-foreground">
          {t("docs.runners.updating.updateCommand.backupNote")}
        </div>
      </section>

      {/* Reinstall Fallback */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.runners.updating.reinstall.title")}
        </h2>
        <p className="text-muted-foreground mb-4">
          {t("docs.runners.updating.reinstall.description")}
        </p>

        <h3 className="text-lg font-medium mb-2 mt-6">
          {t("docs.runners.updating.reinstall.step1Title")}
        </h3>
        <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto">
          <pre className="text-green-500 dark:text-green-400">{`# macOS / Linux
curl -fsSL ${serverUrl}/install.sh | sh

# Windows (PowerShell)
irm ${serverUrl}/install.ps1 | iex`}</pre>
        </div>

        <h3 className="text-lg font-medium mb-2 mt-6">
          {t("docs.runners.updating.reinstall.step2Title")}
        </h3>
        <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto mb-4">
          <pre className="text-green-500 dark:text-green-400">{`# If running as a system service
sudo agentsmesh-runner service stop
sudo agentsmesh-runner service start

# If running in CLI mode, kill the old process first
pkill agentsmesh-runner
agentsmesh-runner run`}</pre>
        </div>

        <div className="bg-muted/50 border border-border rounded-lg p-4 text-sm text-muted-foreground">
          {t("docs.runners.updating.reinstall.configNote")}
        </div>
      </section>

      <DocNavigation />
    </div>
  );
}
