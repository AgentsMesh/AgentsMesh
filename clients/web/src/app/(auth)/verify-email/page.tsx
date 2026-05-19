"use client";

import { Suspense, useState, useEffect, useCallback } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import {
  lightVerifyEmail,
  lightResendVerification,
  resolvePostLoginUrlLight,
} from "@/lib/light-auth";
import { useTranslations } from "next-intl";
import { VerifyingScreen, SuccessScreen, LogoHeader } from "./VerifyStateScreens";

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

  const handleVerifyToken = useCallback(async (verificationToken: string) => {
    setVerifyState("verifying");
    setError("");
    setMessage("");

    try {
      await lightVerifyEmail(verificationToken);
      setVerifyState("success");
      setMessage(t("auth.verifyEmailPage.verificationSuccess"));
      const url = await resolvePostLoginUrlLight({ redirectParam: null });
      router.push(url);
    } catch (err) {
      setVerifyState("error");
      const errorMessage = err instanceof Error ? err.message : String(err);
      if (errorMessage.includes("already verified")) {
        setError(t("auth.verifyEmailPage.alreadyVerifiedError"));
      } else if (errorMessage.includes("expired") || errorMessage.includes("invalid")) {
        setError(t("auth.verifyEmailPage.invalidToken"));
      } else {
        setError(t("auth.verifyEmailPage.verificationFailed"));
      }
    }
  }, [router, t]);

  useEffect(() => {
    if (token && verifyState === "idle") {
      handleVerifyToken(token);
    }
  }, [token, verifyState, handleVerifyToken]);

  const handleResend = async () => {
    if (!email) { setError(t("auth.verifyEmailPage.emailMissing")); return; }
    setLoading(true);
    setError("");
    setMessage("");
    try {
      await lightResendVerification(email);
      setMessage(t("auth.verifyEmailPage.emailSent"));
    } catch {
      setError(t("auth.verifyEmailPage.resendFailed"));
    } finally {
      setLoading(false);
    }
  };

  if (verifyState === "verifying") return <VerifyingScreen t={t} />;
  if (verifyState === "success") return <SuccessScreen t={t} />;

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm space-y-6 text-center">
        <LogoHeader />

        <div className="flex justify-center">
          <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
            <svg className="w-8 h-8 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
                d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
            </svg>
          </div>
        </div>

        <div className="space-y-2">
          <h1 className="text-2xl font-semibold text-foreground">{t("auth.verifyEmailPage.title")}</h1>
          <p className="text-sm text-muted-foreground">
            {email ? t("auth.verifyEmailPage.subtitle", { email }) : t("auth.verifyEmailPage.subtitleDefault")}
            <br />{t("auth.verifyEmailPage.clickLink")}
          </p>
        </div>

        {message && (
          <div className="p-3 text-sm text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/30 rounded-md">{message}</div>
        )}
        {error && (
          <div className="p-3 text-sm text-destructive bg-destructive/10 rounded-md">{error}</div>
        )}

        <div className="space-y-3">
          <Button variant="outline" className="w-full" onClick={handleResend} disabled={loading || !email}>
            {loading ? t("auth.verifyEmailPage.sending") : t("auth.verifyEmailPage.resendEmail")}
          </Button>
          <p className="text-sm text-muted-foreground">
            {t("auth.verifyEmailPage.wrongEmail")}{" "}
            <Link href="/register" className="text-primary hover:underline">{t("auth.verifyEmailPage.signUpDifferent")}</Link>
          </p>
        </div>

        <div className="pt-4 border-t border-border">
          <p className="text-sm text-muted-foreground">
            {t("auth.verifyEmailPage.alreadyVerified")}{" "}
            <Link href="/login" className="text-primary hover:underline">{t("auth.verifyEmailPage.signIn")}</Link>
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
