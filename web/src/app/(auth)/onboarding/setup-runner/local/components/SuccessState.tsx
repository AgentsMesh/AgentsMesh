import Link from "next/link";
import { Button } from "@/components/ui/button";
import type { RunnerData } from "@/lib/api/runner";
import { Logo } from "@/components/common";

interface SuccessStateProps {
  runner: RunnerData;
  t: (key: string) => string;
  onComplete: () => void;
}

export function SuccessState({ runner, t, onComplete }: SuccessStateProps) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-md space-y-6 text-center">
        <div>
          <Link href="/" className="inline-flex items-center gap-2">
            <div className="w-10 h-10 rounded-lg overflow-hidden"><Logo /></div>
            <span className="text-2xl font-bold text-foreground">AgentsMesh</span>
          </Link>
        </div>

        <div className="flex justify-center">
          <div className="w-16 h-16 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center">
            <svg className="w-8 h-8 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
          </div>
        </div>

        <div className="space-y-2">
          <h1 className="text-2xl font-semibold text-foreground">{t("auth.onboarding.localRunner.runnerConnected")}</h1>
          <p className="text-sm text-muted-foreground">{t("auth.onboarding.localRunner.runnerConnectedDescription")}</p>
        </div>

        <div className="p-4 bg-muted rounded-lg text-left">
          <div className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t("auth.onboarding.localRunner.runnerId")}:</span>
              <span className="font-mono text-foreground">{runner.node_id}</span>
            </div>
            {runner.host_info?.os && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{t("auth.onboarding.localRunner.system")}:</span>
                <span className="text-foreground">{runner.host_info.os} {runner.host_info.arch}</span>
              </div>
            )}
            <div className="flex justify-between">
              <span className="text-muted-foreground">{t("auth.onboarding.localRunner.status")}:</span>
              <span className="text-green-600 dark:text-green-400 font-medium">{t("auth.onboarding.localRunner.online")}</span>
            </div>
          </div>
        </div>

        <Button className="w-full" onClick={onComplete}>
          {t("auth.onboarding.localRunner.goToDashboard")}
        </Button>
      </div>
    </div>
  );
}
