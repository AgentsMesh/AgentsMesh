"use client";

import { Suspense, useState, useEffect, useCallback } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { authApi, organizationApi } from "@/lib/api";
import { useTranslations } from "next-intl";
import { useAuthStore } from "@/stores/auth";

type VerifyState = "idle" | "verifying" | "success" | "error";

function VerifyEmailContent() {
  const t = useTranslations();
  const router = useRouter();
  const searchParams = useSearchParams();
  const email = searchParams.get("email") || "";
  const token = searchParams.get("token") || "";

  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [verifyState, setVerifyState] = useState<VerifyState>("idle");

  const { setAuth, setOrganizations } = useAuthStore();

  // Auto-verify when token is present in URL
  const handleVerifyToken = useCallback(async (verificationToken: string) => {
    setVerifyState("verifying");
    setError("");
    setMessage("");

    try {
      // Call verify email API
      const result = await authApi.verifyEmail(verificationToken);

      // Set auth state with returned tokens
      setAuth(result.token, result.user, result.refresh_token);

      setVerifyState("success");
      setMessage(t("auth.verifyEmailPage.verificationSuccess"));

      // Check if user has organizations and redirect accordingly
      try {
        const orgsResponse = await organizationApi.list();
        if (orgsResponse.organizations && orgsResponse.organizations.length > 0) {
          setOrganizations(orgsResponse.organizations);
          // Redirect to first organization's workspace
          router.push(`/${orgsResponse.organizations[0].slug}/workspace`);
        } else {
          // No organizations, redirect to onboarding
          router.push("/onboarding");
        }
      } catch {
        // If org fetch fails, redirect to onboarding as fallback
        router.push("/onboarding");
      }
    } catch (err) {
      setVerifyState("error");
      // Check for specific error types
      const errorMessage = err instanceof Error ? err.message : String(err);
      if (errorMessage.includes("already verified")) {
        setError(t("auth.verifyEmailPage.alreadyVerifiedError"));
      } else if (errorMessage.includes("expired") || errorMessage.includes("invalid")) {
        setError(t("auth.verifyEmailPage.invalidToken"));
      } else {
        setError(t("auth.verifyEmailPage.verificationFailed"));
      }
    }
  }, [setAuth, setOrganizations, router, t]);

  // Effect to auto-verify when token is in URL
  useEffect(() => {
    if (token && verifyState === "idle") {
      handleVerifyToken(token);
    }
  }, [token, verifyState, handleVerifyToken]);

  const handleResend = async () => {
    if (!email) {
      setError(t("auth.verifyEmailPage.emailMissing"));
      return;
    }

    setLoading(true);
    setError("");
    setMessage("");

    try {
      await authApi.resendVerification(email);
      setMessage(t("auth.verifyEmailPage.emailSent"));
    } catch {
      setError(t("auth.verifyEmailPage.resendFailed"));
    } finally {
      setLoading(false);
    }
  };

  // Show verifying state when token is being processed
  if (verifyState === "verifying") {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background px-4">
        <div className="w-full max-w-sm space-y-6 text-center">
          {/* Logo */}
          <div>
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
          </div>

          {/* Loading spinner */}
          <div className="flex justify-center">
            <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
              <svg
                className="w-8 h-8 text-primary animate-spin"
                fill="none"
                viewBox="0 0 24 24"
              >
                <circle
                  className="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  strokeWidth="4"
                />
                <path
                  className="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                />
              </svg>
            </div>
          </div>

          <div className="space-y-2">
            <h1 className="text-2xl font-semibold text-foreground">
              {t("auth.verifyEmailPage.verifying")}
            </h1>
            <p className="text-sm text-muted-foreground">
              {t("auth.verifyEmailPage.pleaseWait")}
            </p>
          </div>
        </div>
      </div>
    );
  }

  // Show success state briefly before redirect
  if (verifyState === "success") {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background px-4">
        <div className="w-full max-w-sm space-y-6 text-center">
          {/* Logo */}
          <div>
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
          </div>

          {/* Success icon */}
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

          <div className="space-y-2">
            <h1 className="text-2xl font-semibold text-foreground">
              {t("auth.verifyEmailPage.verificationSuccessTitle")}
            </h1>
            <p className="text-sm text-muted-foreground">
              {t("auth.verifyEmailPage.redirecting")}
            </p>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm space-y-6 text-center">
        {/* Logo */}
        <div>
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
        </div>

        {/* Icon */}
        <div className="flex justify-center">
          <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
            <svg
              className="w-8 h-8 text-primary"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
              />
            </svg>
          </div>
        </div>

        {/* Content */}
        <div className="space-y-2">
          <h1 className="text-2xl font-semibold text-foreground">
            {t("auth.verifyEmailPage.title")}
          </h1>
          <p className="text-sm text-muted-foreground">
            {email
              ? t("auth.verifyEmailPage.subtitle", { email })
              : t("auth.verifyEmailPage.subtitleDefault")}
            <br />
            {t("auth.verifyEmailPage.clickLink")}
          </p>
        </div>

        {/* Messages */}
        {message && (
          <div className="p-3 text-sm text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/30 rounded-md">
            {message}
          </div>
        )}
        {error && (
          <div className="p-3 text-sm text-destructive bg-destructive/10 rounded-md">
            {error}
          </div>
        )}

        {/* Actions */}
        <div className="space-y-3">
          <Button
            variant="outline"
            className="w-full"
            onClick={handleResend}
            disabled={loading || !email}
          >
            {loading ? t("auth.verifyEmailPage.sending") : t("auth.verifyEmailPage.resendEmail")}
          </Button>

          <p className="text-sm text-muted-foreground">
            {t("auth.verifyEmailPage.wrongEmail")}{" "}
            <Link href="/register" className="text-primary hover:underline">
              {t("auth.verifyEmailPage.signUpDifferent")}
            </Link>
          </p>
        </div>

        {/* Footer */}
        <div className="pt-4 border-t border-border">
          <p className="text-sm text-muted-foreground">
            {t("auth.verifyEmailPage.alreadyVerified")}{" "}
            <Link href="/login" className="text-primary hover:underline">
              {t("auth.verifyEmailPage.signIn")}
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}

export default function VerifyEmailPage() {
  const t = useTranslations();
  return (
    <Suspense fallback={
      <div className="flex min-h-screen items-center justify-center bg-background px-4">
        <div className="w-8 h-8 text-primary animate-spin">{t("common.loading")}</div>
      </div>
    }>
      <VerifyEmailContent />
    </Suspense>
  );
}
