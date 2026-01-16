"use client";

import { Suspense, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { authApi } from "@/lib/api/client";
import { useTranslations } from "@/lib/i18n/client";

function VerifyEmailContent() {
  const t = useTranslations();
  const searchParams = useSearchParams();
  const email = searchParams.get("email") || "";
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

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
          <div className="p-3 text-sm text-green-600 bg-green-50 rounded-md">
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
