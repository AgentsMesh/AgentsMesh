"use client";

import { useState, useEffect, useCallback } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { useAuthStore } from "@/stores/auth";
import { runnerApi, RunnerData } from "@/lib/api/runner";
import { isApiErrorCode } from "@/lib/api/errors";
import { useServerUrl } from "@/hooks/useServerUrl";
import { useTranslations } from "next-intl";
import { Logo } from "@/components/common";
import { SuccessState } from "./components/SuccessState";
import { SetupSteps } from "./components/SetupSteps";

export default function LocalRunnerSetupPage() {
  const router = useRouter();
  const t = useTranslations();
  const { currentOrg } = useAuthStore();
  const serverUrl = useServerUrl();
  const [token, setToken] = useState<string | null>(null);
  const [tokenCopied, setTokenCopied] = useState(false);
  const [registerMethod, setRegisterMethod] = useState<"token" | "login">("token");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [connectionStatus, setConnectionStatus] = useState<"waiting" | "connected" | "timeout">("waiting");
  const [connectedRunner, setConnectedRunner] = useState<RunnerData | null>(null);
  const [waitTime, setWaitTime] = useState(0);

  useEffect(() => {
    const generateToken = async () => {
      try {
        setLoading(true);
        const { token: newToken } = await runnerApi.createToken();
        setToken(newToken);
      } catch (err) {
        if (isApiErrorCode(err, "ADMIN_REQUIRED") || isApiErrorCode(err, "INSUFFICIENT_PERMISSIONS")) {
          setError(t("apiErrors.INSUFFICIENT_PERMISSIONS"));
        } else {
          setError(t("auth.onboarding.localRunner.tokenGenerationFailed"));
        }
      } finally { setLoading(false); }
    };
    generateToken();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const checkRunnerConnection = useCallback(async () => {
    try {
      const { runners } = await runnerApi.list();
      const onlineRunner = runners.find((r) => r.status === "online");
      if (onlineRunner) {
        setConnectionStatus("connected");
        setConnectedRunner(onlineRunner);
        return true;
      }
    } catch { /* Ignore errors during polling */ }
    return false;
  }, []);

  useEffect(() => {
    if (!token || connectionStatus === "connected") return;
    const interval = setInterval(async () => {
      setWaitTime((prev) => prev + 3);
      const connected = await checkRunnerConnection();
      if (connected) clearInterval(interval);
    }, 3000);
    const timeout = setTimeout(() => {
      setConnectionStatus("timeout");
      clearInterval(interval);
    }, 10 * 60 * 1000);
    return () => { clearInterval(interval); clearTimeout(timeout); };
  }, [token, connectionStatus, checkRunnerConnection]);

  const handleCopyToken = async () => {
    if (!token) return;
    try {
      await navigator.clipboard.writeText(token);
      setTokenCopied(true);
      setTimeout(() => setTokenCopied(false), 2000);
    } catch {
      const textarea = document.createElement("textarea");
      textarea.value = token;
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand("copy");
      document.body.removeChild(textarea);
      setTokenCopied(true);
      setTimeout(() => setTokenCopied(false), 2000);
    }
  };

  const handleComplete = () => {
    if (currentOrg) { router.push(`/${currentOrg.slug}/workspace`); }
    else { router.push("/"); }
  };

  if (connectionStatus === "connected" && connectedRunner) {
    return <SuccessState runner={connectedRunner} t={t} onComplete={handleComplete} />;
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-lg space-y-8">
        <div className="text-center">
          <Link href="/" className="inline-flex items-center gap-2">
            <div className="w-10 h-10 rounded-lg overflow-hidden"><Logo /></div>
            <span className="text-2xl font-bold text-foreground">AgentsMesh</span>
          </Link>
          <h1 className="mt-6 text-2xl font-semibold text-foreground">{t("auth.onboarding.localRunner.title")}</h1>
          <p className="mt-2 text-sm text-muted-foreground">{t("auth.onboarding.localRunner.subtitle")}</p>
        </div>

        {error && (
          <div className="p-3 text-sm text-destructive bg-destructive/10 rounded-md">{error}</div>
        )}

        <SetupSteps token={token} tokenCopied={tokenCopied} loading={loading}
          serverUrl={serverUrl} registerMethod={registerMethod} setRegisterMethod={setRegisterMethod}
          connectionStatus={connectionStatus} waitTime={waitTime} t={t}
          onCopyToken={handleCopyToken} onRetry={() => { setConnectionStatus("waiting"); setWaitTime(0); }} />

        <div className="flex items-center justify-between pt-4 border-t border-border">
          <Link href="/onboarding/setup-runner" className="text-sm text-muted-foreground hover:text-foreground">
            {t("auth.onboarding.localRunner.back")}
          </Link>
          <Button variant="ghost" onClick={handleComplete}>
            {t("auth.onboarding.localRunner.skipForNow")}
          </Button>
        </div>
      </div>
    </div>
  );
}
