"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { useAuthStore } from "@/stores/auth";
import { useTranslations } from "next-intl";

export default function SetupRunnerPage() {
  const router = useRouter();
  const t = useTranslations();
  const { currentOrg } = useAuthStore();

  const handleSkip = () => {
    if (currentOrg) {
      router.push(`/${currentOrg.slug}`);
    } else {
      router.push("/");
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-md space-y-8">
        {/* Header */}
        <div className="text-center">
          <Link href="/" className="inline-flex items-center gap-2">
            <div className="w-10 h-10 rounded-lg bg-primary flex items-center justify-center">
              <svg
                className="w-6 h-6 text-primary-foreground"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z"
                />
              </svg>
            </div>
            <span className="text-2xl font-bold text-foreground">AgentsMesh</span>
          </Link>
          <h1 className="mt-6 text-2xl font-semibold text-foreground">
            {t("auth.onboarding.setupRunner.title")}
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">
            {t("auth.onboarding.setupRunner.subtitle")}
          </p>
        </div>

        {/* Options */}
        <div className="space-y-4">
          {/* Local Runner */}
          <div className="p-6 border border-border rounded-lg hover:border-primary/50 transition-colors">
            <div className="flex items-start gap-4">
              <div className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0">
                <svg
                  className="w-6 h-6 text-primary"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
                  />
                </svg>
              </div>
              <div className="flex-1">
                <h3 className="font-semibold text-foreground">{t("auth.onboarding.setupRunner.localRunner")}</h3>
                <p className="mt-1 text-sm text-muted-foreground">
                  {t("auth.onboarding.setupRunner.localRunnerDescription")}
                </p>
                <Link href="/onboarding/setup-runner/local">
                  <Button className="mt-4 w-full">
                    {t("auth.onboarding.setupRunner.setUpLocalRunner")}
                  </Button>
                </Link>
              </div>
            </div>
          </div>

          {/* Cloud Runner */}
          <div className="p-6 border border-border rounded-lg opacity-60">
            <div className="flex items-start gap-4">
              <div className="w-12 h-12 rounded-lg bg-blue-500/10 flex items-center justify-center flex-shrink-0">
                <svg
                  className="w-6 h-6 text-blue-500 dark:text-blue-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M3 15a4 4 0 004 4h9a5 5 0 10-.1-9.999 5.002 5.002 0 10-9.78 2.096A4.001 4.001 0 003 15z"
                  />
                </svg>
              </div>
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <h3 className="font-semibold text-foreground">{t("auth.onboarding.setupRunner.cloudRunner")}</h3>
                  <span className="text-xs px-2 py-0.5 bg-muted rounded-full text-muted-foreground">
                    {t("auth.onboarding.setupRunner.comingSoon")}
                  </span>
                </div>
                <p className="mt-1 text-sm text-muted-foreground">
                  {t("auth.onboarding.setupRunner.cloudRunnerDescription")}
                </p>
                <Button className="mt-4 w-full" variant="outline" disabled>
                  {t("auth.onboarding.setupRunner.comingSoon")}
                </Button>
              </div>
            </div>
          </div>
        </div>

        {/* Skip */}
        <div className="pt-4 border-t border-border text-center">
          <Button variant="ghost" onClick={handleSkip}>
            {t("auth.onboarding.setupRunner.skipForNow")}
          </Button>
        </div>
      </div>
    </div>
  );
}
