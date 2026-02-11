"use client";

import { useState } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { authApi } from "@/lib/api";
import { useTranslations } from "next-intl";

export default function ForgotPasswordPage() {
  const t = useTranslations();
  const [email, setEmail] = useState("");
  const [loading, setLoading] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [error, setError] = useState("");

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    try {
      await authApi.forgotPassword(email);
      setSubmitted(true);
    } catch {
      setError(t("auth.forgotPasswordPage.sendFailed"));
    } finally {
      setLoading(false);
    }
  };

  if (submitted) {
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
              {t("auth.forgotPasswordPage.checkEmail")}
            </h1>
            <p className="text-sm text-muted-foreground">
              {t("auth.forgotPasswordPage.emailSentDescription", { email })}
            </p>
          </div>

          {/* Actions */}
          <div className="space-y-3 pt-4">
            <Link href="/login">
              <Button variant="outline" className="w-full">
                {t("auth.forgotPasswordPage.backToSignIn")}
              </Button>
            </Link>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm space-y-6">
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
            {t("auth.forgotPasswordPage.title")}
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">
            {t("auth.forgotPasswordPage.subtitle")}
          </p>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div className="p-3 text-sm text-destructive bg-destructive/10 rounded-md">
              {error}
            </div>
          )}

          <div className="space-y-2">
            <label htmlFor="email" className="text-sm font-medium text-foreground">
              {t("auth.forgotPasswordPage.emailLabel")}
            </label>
            <Input
              id="email"
              type="email"
              placeholder={t("auth.forgotPasswordPage.emailPlaceholder")}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </div>

          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? t("auth.forgotPasswordPage.sending") : t("auth.forgotPasswordPage.sendResetLink")}
          </Button>
        </form>

        {/* Back link */}
        <p className="text-center text-sm text-muted-foreground">
          {t("auth.forgotPasswordPage.rememberPassword")}{" "}
          <Link href="/login" className="text-primary hover:underline">
            {t("auth.forgotPasswordPage.signIn")}
          </Link>
        </p>
      </div>
    </div>
  );
}
