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

  // Generate registration token
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
      } finally {
        setLoading(false);
      }
    };

    generateToken();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []); // t is stable, effect should only run on mount

  // Poll for runner connection
  const checkRunnerConnection = useCallback(async () => {
    try {
      const { runners } = await runnerApi.list();
      const onlineRunner = runners.find((r) => r.status === "online");
      if (onlineRunner) {
        setConnectionStatus("connected");
        setConnectedRunner(onlineRunner);
        return true;
      }
    } catch {
      // Ignore errors during polling
    }
    return false;
  }, []);

  useEffect(() => {
    if (!token || connectionStatus === "connected") return;

    const interval = setInterval(async () => {
      setWaitTime((prev) => prev + 3);

      const connected = await checkRunnerConnection();
      if (connected) {
        clearInterval(interval);
      }
    }, 3000);

    // Timeout after 10 minutes
    const timeout = setTimeout(() => {
      setConnectionStatus("timeout");
      clearInterval(interval);
    }, 10 * 60 * 1000);

    return () => {
      clearInterval(interval);
      clearTimeout(timeout);
    };
  }, [token, connectionStatus, checkRunnerConnection]);

  const handleCopyToken = async () => {
    if (!token) return;

    try {
      await navigator.clipboard.writeText(token);
      setTokenCopied(true);
      setTimeout(() => setTokenCopied(false), 2000);
    } catch {
      // Fallback for older browsers
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
    if (currentOrg) {
      router.push(`/${currentOrg.slug}/workspace`);
    } else {
      router.push("/");
    }
  };

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, "0")}`;
  };

  // Success state
  if (connectionStatus === "connected" && connectedRunner) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background px-4">
        <div className="w-full max-w-md space-y-6 text-center">
          {/* Logo */}
          <div>
            <Link href="/" className="inline-flex items-center gap-2">
              <div className="w-10 h-10 rounded-lg overflow-hidden">
                <Logo />
              </div>
              <span className="text-2xl font-bold text-foreground">AgentsMesh</span>
            </Link>
          </div>

          {/* Success Icon */}
          <div className="flex justify-center">
            <div className="w-16 h-16 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center">
              <svg
                className="w-8 h-8 text-green-600 dark:text-green-400"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M5 13l4 4L19 7"
                />
              </svg>
            </div>
          </div>

          {/* Content */}
          <div className="space-y-2">
            <h1 className="text-2xl font-semibold text-foreground">
              {t("auth.onboarding.localRunner.runnerConnected")}
            </h1>
            <p className="text-sm text-muted-foreground">
              {t("auth.onboarding.localRunner.runnerConnectedDescription")}
            </p>
          </div>

          {/* Runner Info */}
          <div className="p-4 bg-muted rounded-lg text-left">
            <div className="space-y-2 text-sm">
              <div className="flex justify-between">
                <span className="text-muted-foreground">{t("auth.onboarding.localRunner.runnerId")}:</span>
                <span className="font-mono text-foreground">{connectedRunner.node_id}</span>
              </div>
              {connectedRunner.host_info?.os && (
                <div className="flex justify-between">
                  <span className="text-muted-foreground">{t("auth.onboarding.localRunner.system")}:</span>
                  <span className="text-foreground">
                    {connectedRunner.host_info.os} {connectedRunner.host_info.arch}
                  </span>
                </div>
              )}
              <div className="flex justify-between">
                <span className="text-muted-foreground">{t("auth.onboarding.localRunner.status")}:</span>
                <span className="text-green-600 dark:text-green-400 font-medium">{t("auth.onboarding.localRunner.online")}</span>
              </div>
            </div>
          </div>

          <Button className="w-full" onClick={handleComplete}>
            {t("auth.onboarding.localRunner.goToDashboard")}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-lg space-y-8">
        {/* Header */}
        <div className="text-center">
          <Link href="/" className="inline-flex items-center gap-2">
            <div className="w-10 h-10 rounded-lg overflow-hidden">
              <Logo />
            </div>
            <span className="text-2xl font-bold text-foreground">AgentsMesh</span>
          </Link>
          <h1 className="mt-6 text-2xl font-semibold text-foreground">
            {t("auth.onboarding.localRunner.title")}
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">
            {t("auth.onboarding.localRunner.subtitle")}
          </p>
        </div>

        {/* Error */}
        {error && (
          <div className="p-3 text-sm text-destructive bg-destructive/10 rounded-md">
            {error}
          </div>
        )}

        {/* Steps */}
        <div className="space-y-6">
          {/* Step 1: Token */}
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <div className="w-6 h-6 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-medium">
                1
              </div>
              <h3 className="font-medium text-foreground">{t("auth.onboarding.localRunner.step1Title")}</h3>
            </div>
            <div className="ml-8">
              {loading ? (
                <div className="p-3 bg-muted rounded-md text-sm text-muted-foreground">
                  {t("auth.onboarding.localRunner.generatingToken")}
                </div>
              ) : token ? (
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <code className="flex-1 p-3 bg-muted rounded-md text-sm font-mono text-foreground overflow-x-auto">
                      {token}
                    </code>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleCopyToken}
                      className="flex-shrink-0"
                    >
                      {tokenCopied ? (
                        <svg className="w-4 h-4 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
                        </svg>
                      ) : (
                        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                        </svg>
                      )}
                    </Button>
                  </div>
                  <p className="text-xs text-amber-600 dark:text-amber-400">
                    {t("auth.onboarding.localRunner.tokenWarning")}
                  </p>
                </div>
              ) : (
                <div className="p-3 bg-destructive/10 rounded-md text-sm text-destructive">
                  {t("auth.onboarding.localRunner.tokenGenerationFailedShort")}
                </div>
              )}
            </div>
          </div>

          {/* Step 2: Install */}
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <div className="w-6 h-6 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-medium">
                2
              </div>
              <h3 className="font-medium text-foreground">{t("auth.onboarding.localRunner.step2Title")}</h3>
            </div>
            <div className="ml-8 space-y-3">
              <div className="p-4 bg-muted rounded-md">
                <p className="text-xs text-muted-foreground mb-2"># macOS / Linux</p>
                <code className="text-sm font-mono text-foreground block">
                  curl -fsSL {serverUrl}/install.sh | sh
                </code>
              </div>
              <div className="p-4 bg-muted rounded-md">
                <p className="text-xs text-muted-foreground mb-2"># Windows (PowerShell)</p>
                <code className="text-sm font-mono text-foreground block">
                  irm {serverUrl}/install.ps1 | iex
                </code>
              </div>
              {/* Register method tabs */}
              <div className="border border-border rounded-md overflow-hidden">
                <div className="flex border-b border-border">
                  <button
                    type="button"
                    className={`flex-1 px-4 py-2 text-sm font-medium transition-colors ${
                      registerMethod === "token"
                        ? "bg-primary text-primary-foreground"
                        : "bg-muted text-muted-foreground hover:text-foreground"
                    }`}
                    onClick={() => setRegisterMethod("token")}
                  >
                    {t("auth.onboarding.localRunner.methodToken")}
                  </button>
                  <button
                    type="button"
                    className={`flex-1 px-4 py-2 text-sm font-medium transition-colors ${
                      registerMethod === "login"
                        ? "bg-primary text-primary-foreground"
                        : "bg-muted text-muted-foreground hover:text-foreground"
                    }`}
                    onClick={() => setRegisterMethod("login")}
                  >
                    {t("auth.onboarding.localRunner.methodLogin")}
                  </button>
                </div>
                <div className="p-4 bg-muted">
                  {registerMethod === "token" ? (
                    <>
                      <p className="text-xs text-muted-foreground mb-2"># {t("auth.onboarding.localRunner.startRunnerComment")}</p>
                      <code className="text-sm font-mono text-foreground block whitespace-pre-wrap">
{`agentsmesh-runner register --server ${serverUrl} --token ${token || "<your-token>"}
agentsmesh-runner run`}
                      </code>
                    </>
                  ) : (
                    <>
                      <p className="text-xs text-muted-foreground mb-2"># {t("auth.onboarding.localRunner.loginRunnerComment")}</p>
                      <code className="text-sm font-mono text-foreground block whitespace-pre-wrap">
{`agentsmesh-runner login
agentsmesh-runner run`}
                      </code>
                    </>
                  )}
                </div>
              </div>
            </div>
          </div>

          {/* Step 3: Waiting */}
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <div className="w-6 h-6 rounded-full bg-primary text-primary-foreground flex items-center justify-center text-sm font-medium">
                3
              </div>
              <h3 className="font-medium text-foreground">{t("auth.onboarding.localRunner.step3Title")}</h3>
            </div>
            <div className="ml-8">
              {connectionStatus === "waiting" && (
                <div className="p-4 border border-border rounded-md">
                  <div className="flex items-center gap-3">
                    <div className="w-5 h-5 border-2 border-primary border-t-transparent rounded-full animate-spin" />
                    <div>
                      <p className="text-sm text-foreground">{t("auth.onboarding.localRunner.waitingForConnection")}</p>
                      <p className="text-xs text-muted-foreground">{t("auth.onboarding.localRunner.elapsed")}: {formatTime(waitTime)}</p>
                    </div>
                  </div>
                </div>
              )}
              {connectionStatus === "timeout" && (
                <div className="p-4 border border-amber-500/50 bg-amber-50 dark:bg-amber-950/30 rounded-md">
                  <p className="text-sm text-amber-800 dark:text-amber-200">
                    {t("auth.onboarding.localRunner.connectionTimeout")}
                  </p>
                  <Button
                    variant="outline"
                    size="sm"
                    className="mt-2"
                    onClick={() => {
                      setConnectionStatus("waiting");
                      setWaitTime(0);
                    }}
                  >
                    {t("auth.onboarding.localRunner.retry")}
                  </Button>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Footer */}
        <div className="flex items-center justify-between pt-4 border-t border-border">
          <Link
            href="/onboarding/setup-runner"
            className="text-sm text-muted-foreground hover:text-foreground"
          >
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
