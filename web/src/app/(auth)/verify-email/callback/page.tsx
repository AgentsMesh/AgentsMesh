"use client";

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { useAuthStore } from "@/stores/auth";
import { authApi } from "@/lib/api";

function VerifyEmailCallbackContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token");
  const { setAuth } = useAuthStore();

  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");
  const [error, setError] = useState("");

  useEffect(() => {
    const verifyEmail = async () => {
      if (!token) {
        setStatus("error");
        setError("Verification token is missing");
        return;
      }

      try {
        const response = await authApi.verifyEmail(token);

        // Store auth tokens
        setAuth(response.token, {
          id: response.user.id,
          email: response.user.email,
          username: response.user.username,
          name: response.user.name,
        });

        setStatus("success");

        // Redirect to onboarding after a brief delay
        setTimeout(() => {
          router.push("/onboarding");
        }, 2000);
      } catch (err: unknown) {
        setStatus("error");
        if (err instanceof Error && err.message.includes("already verified")) {
          setError("This email has already been verified. Please sign in.");
        } else {
          setError("The verification link is invalid or has expired.");
        }
      }
    };

    verifyEmail();
  }, [token, setAuth, router]);

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
                Verifying your email...
              </h1>
              <p className="text-sm text-muted-foreground">
                Please wait while we verify your email address.
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
                Email verified!
              </h1>
              <p className="text-sm text-muted-foreground">
                Your email has been verified successfully.
                <br />
                Redirecting you to continue setup...
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
                Verification failed
              </h1>
              <p className="text-sm text-muted-foreground">{error}</p>
            </div>
            <div className="space-y-3">
              <Link href="/login">
                <Button className="w-full">Sign in</Button>
              </Link>
              <Link href="/register">
                <Button variant="outline" className="w-full">Sign up again</Button>
              </Link>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

export default function VerifyEmailCallbackPage() {
  return (
    <Suspense fallback={
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
    }>
      <VerifyEmailCallbackContent />
    </Suspense>
  );
}
