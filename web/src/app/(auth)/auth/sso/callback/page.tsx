"use client";

import { Suspense, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { useAuthStore } from "@/stores/auth";
import { userApi, organizationApi } from "@/lib/api";
import { Logo } from "@/components/common";
import { useTranslations } from "next-intl";

function SSOCallbackContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const t = useTranslations();
  const token = searchParams.get("token");
  const refreshToken = searchParams.get("refresh_token");
  const error = searchParams.get("error");
  const { setAuth, setOrganizations } = useAuthStore();

  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");
  const [errorMessage, setErrorMessage] = useState("");
  const redirectTimerRef = useRef<ReturnType<typeof setTimeout>>(null);
  const processedRef = useRef(false);

  useEffect(() => {
    // Guard against duplicate execution (e.g., dependency change triggering re-run)
    if (processedRef.current) return;
    processedRef.current = true;

    // Remove tokens from URL to prevent leaking via browser history/Referer
    if (token || refreshToken || error) {
      window.history.replaceState({}, "", window.location.pathname);
    }

    const scheduleRedirect = (path: string) => {
      redirectTimerRef.current = setTimeout(() => router.push(path), 1500);
    };

    const handleCallback = async () => {
      if (error) {
        setStatus("error");
        // Only display known error codes to avoid leaking internal details
        const knownErrors: Record<string, string> = {
          access_denied: t("auth.sso.callbackAccessDenied"),
          authentication_failed: t("auth.sso.callbackGenericError"),
          missing_state: t("auth.sso.callbackGenericError"),
          invalid_state: t("auth.sso.callbackGenericError"),
        };
        setErrorMessage(
          knownErrors[error] ?? t("auth.sso.callbackGenericError")
        );
        return;
      }

      if (!token) {
        setStatus("error");
        setErrorMessage(t("auth.sso.callbackMissingToken"));
        return;
      }

      try {
        // Set token so subsequent API calls work
        setAuth(token, { id: 0, email: "", username: "" }, refreshToken || undefined);

        // Get user info
        const userResponse = await userApi.getMe();
        setAuth(token, userResponse.user, refreshToken || undefined);

        // Get organizations and redirect
        try {
          const orgsResponse = await organizationApi.list();
          const orgs = orgsResponse.organizations;
          if (orgs && orgs.length > 0) {
            setOrganizations(orgs);
            setStatus("success");
            scheduleRedirect(`/${orgs[0].slug}/workspace`);
          } else {
            setStatus("success");
            scheduleRedirect("/onboarding");
          }
        } catch {
          setStatus("success");
          scheduleRedirect("/onboarding");
        }
      } catch {
        setStatus("error");
        setErrorMessage(t("auth.sso.callbackGenericError"));
      }
    };

    handleCallback();

    return () => {
      if (redirectTimerRef.current) clearTimeout(redirectTimerRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- processedRef ensures single execution
  }, [token, refreshToken, error]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm space-y-6 text-center">
        {/* Logo */}
        <div>
          <Link href="/" className="inline-flex items-center gap-2">
            <div className="w-10 h-10 rounded-lg overflow-hidden">
              <Logo />
            </div>
            <span className="text-2xl font-bold text-foreground">AgentsMesh</span>
          </Link>
        </div>

        {/* Loading State */}
        {status === "loading" && (
          <>
            <div className="flex justify-center">
              <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center animate-pulse">
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
                {t("auth.sso.completingSignIn")}
              </h1>
              <p className="text-sm text-muted-foreground">
                {t("auth.sso.pleaseWait")}
              </p>
            </div>
          </>
        )}

        {/* Success State */}
        {status === "success" && (
          <>
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
                {t("auth.sso.welcome")}
              </h1>
              <p className="text-sm text-muted-foreground">
                {t("auth.sso.signInSuccess")}
                <br />
                {t("auth.sso.redirecting")}
              </p>
            </div>
          </>
        )}

        {/* Error State */}
        {status === "error" && (
          <>
            <div className="flex justify-center">
              <div className="w-16 h-16 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center">
                <svg
                  className="w-8 h-8 text-red-600 dark:text-red-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M6 18L18 6M6 6l12 12"
                  />
                </svg>
              </div>
            </div>
            <div className="space-y-2">
              <h1 className="text-2xl font-semibold text-foreground">
                {t("auth.sso.signInFailed")}
              </h1>
              <p className="text-sm text-muted-foreground">{errorMessage}</p>
            </div>
            <div className="space-y-3">
              <Link href="/login">
                <Button className="w-full">{t("auth.sso.tryAgain")}</Button>
              </Link>
              <Link href="/register">
                <Button variant="outline" className="w-full">
                  {t("auth.sso.createAccount")}
                </Button>
              </Link>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

export default function SSOCallbackPage() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen items-center justify-center bg-background px-4">
          <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center animate-pulse">
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
      }
    >
      <SSOCallbackContent />
    </Suspense>
  );
}
