"use client";

import Link from "next/link";
import { useServerUrl } from "@/hooks/useServerUrl";
import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function GettingStartedPage() {
  const serverUrl = useServerUrl();
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.gettingStarted.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.gettingStarted.description")}
      </p>

      {/* Step 1 */}
      <section className="mb-8">
        <div className="border border-border rounded-lg p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-8 h-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-bold">
              1
            </div>
            <h2 className="text-xl font-semibold">
              {t("docs.gettingStarted.step1.title")}
            </h2>
          </div>
          <p className="text-muted-foreground mb-4">
            {t("docs.gettingStarted.step1.description")}
          </p>
          <div className="bg-muted rounded-lg p-4 text-sm">
            <p className="font-medium mb-2">
              {t("docs.gettingStarted.step1.whatYouSetUp")}
            </p>
            <ul className="list-disc list-inside text-muted-foreground space-y-1">
              <li>{t("docs.gettingStarted.step1.item1")}</li>
              <li>{t("docs.gettingStarted.step1.item2")}</li>
              <li>{t("docs.gettingStarted.step1.item3")}</li>
            </ul>
          </div>
        </div>
      </section>

      {/* Step 2 */}
      <section className="mb-8">
        <div className="border border-border rounded-lg p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-8 h-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-bold">
              2
            </div>
            <h2 className="text-xl font-semibold">
              {t("docs.gettingStarted.step2.title")}
            </h2>
          </div>
          <p className="text-muted-foreground mb-4">
            {t("docs.gettingStarted.step2.description")}
          </p>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="bg-muted rounded-lg p-4">
              <h4 className="font-medium mb-2">
                {t("docs.gettingStarted.step2.anthropic")}
              </h4>
              <p className="text-sm text-muted-foreground">
                {t("docs.gettingStarted.step2.anthropicDesc")}{" "}
                <a
                  href="https://console.anthropic.com"
                  className="text-primary hover:underline"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  console.anthropic.com
                </a>
              </p>
            </div>
            <div className="bg-muted rounded-lg p-4">
              <h4 className="font-medium mb-2">
                {t("docs.gettingStarted.step2.openai")}
              </h4>
              <p className="text-sm text-muted-foreground">
                {t("docs.gettingStarted.step2.openaiDesc")}{" "}
                <a
                  href="https://platform.openai.com"
                  className="text-primary hover:underline"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  platform.openai.com
                </a>
              </p>
            </div>
            <div className="bg-muted rounded-lg p-4">
              <h4 className="font-medium mb-2">
                {t("docs.gettingStarted.step2.google")}
              </h4>
              <p className="text-sm text-muted-foreground">
                {t("docs.gettingStarted.step2.googleDesc")}{" "}
                <a
                  href="https://aistudio.google.com"
                  className="text-primary hover:underline"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  aistudio.google.com
                </a>
              </p>
            </div>
            <div className="bg-muted rounded-lg p-4">
              <h4 className="font-medium mb-2">
                {t("docs.gettingStarted.step2.custom")}
              </h4>
              <p className="text-sm text-muted-foreground">
                {t("docs.gettingStarted.step2.customDesc")}
              </p>
            </div>
          </div>
        </div>
      </section>

      {/* Step 3 */}
      <section className="mb-8">
        <div className="border border-border rounded-lg p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-8 h-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-bold">
              3
            </div>
            <h2 className="text-xl font-semibold">
              {t("docs.gettingStarted.step3.title")}
            </h2>
          </div>
          <p className="text-muted-foreground mb-4">
            {t("docs.gettingStarted.step3.description")}
          </p>
          <div className="bg-muted rounded-lg p-4 font-mono text-sm overflow-x-auto">
            <pre className="text-green-500 dark:text-green-400">{`# Download and install the runner
curl -fsSL ${serverUrl}/install.sh | sh

# Register with your token (from Settings → Runners)
agentsmesh-runner register --server ${serverUrl} --token <YOUR_TOKEN>

# Start the runner
agentsmesh-runner run`}</pre>
          </div>
          <p className="text-sm text-muted-foreground mt-4">
            {(() => {
              const raw = t("docs.gettingStarted.step3.seeSetup");
              const parts = raw.split("{link}");
              if (parts.length < 2) return raw;
              return (
                <>
                  {parts[0]}
                  <Link
                    href="/docs/runners/setup"
                    className="text-primary hover:underline"
                  >
                    {t("docs.nav.runnerSetup")}
                  </Link>
                  {parts[1]}
                </>
              );
            })()}
          </p>
        </div>
      </section>

      {/* Step 4 */}
      <section className="mb-8">
        <div className="border border-border rounded-lg p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-8 h-8 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-bold">
              4
            </div>
            <h2 className="text-xl font-semibold">
              {t("docs.gettingStarted.step4.title")}
            </h2>
          </div>
          <p className="text-muted-foreground mb-4">
            {t("docs.gettingStarted.step4.description")}
          </p>
          <ol className="list-decimal list-inside text-muted-foreground space-y-2">
            <li>{t("docs.gettingStarted.step4.item1")}</li>
            <li>{t("docs.gettingStarted.step4.item2")}</li>
            <li>{t("docs.gettingStarted.step4.item3")}</li>
            <li>{t("docs.gettingStarted.step4.item4")}</li>
            <li>{t("docs.gettingStarted.step4.item5")}</li>
            <li>{t("docs.gettingStarted.step4.item6")}</li>
          </ol>
        </div>
      </section>

      {/* Next Steps */}
      <section className="mb-8">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.gettingStarted.nextSteps.title")}
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <Link
            href="/docs/features/agentpod"
            className="border border-border rounded-lg p-4 hover:border-primary transition-colors"
          >
            <h3 className="font-medium mb-1">
              {t("docs.gettingStarted.nextSteps.agentpod")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.gettingStarted.nextSteps.agentpodDesc")}
            </p>
          </Link>
          <Link
            href="/docs/features/mesh"
            className="border border-border rounded-lg p-4 hover:border-primary transition-colors"
          >
            <h3 className="font-medium mb-1">
              {t("docs.gettingStarted.nextSteps.mesh")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.gettingStarted.nextSteps.meshDesc")}
            </p>
          </Link>
        </div>
      </section>

      <DocNavigation />
    </div>
  );
}
