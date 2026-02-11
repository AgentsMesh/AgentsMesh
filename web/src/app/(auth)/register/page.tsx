"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuthStore } from "@/stores/auth";
import { authApi } from "@/lib/api";
import { useTranslations } from "next-intl";
import { getOAuthBaseUrl } from "@/lib/env";

export default function RegisterPage() {
  const router = useRouter();
  const t = useTranslations();
  const { setAuth } = useAuthStore();
  const [formData, setFormData] = useState({
    email: "",
    username: "",
    password: "",
    confirmPassword: "",
    name: "",
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFormData((prev) => ({
      ...prev,
      [e.target.name]: e.target.value,
    }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError("");

    // Validate passwords match
    if (formData.password !== formData.confirmPassword) {
      setError(t("auth.registerPage.passwordsNotMatch"));
      setLoading(false);
      return;
    }

    // Validate password strength
    if (formData.password.length < 8) {
      setError(t("auth.registerPage.passwordTooShort"));
      setLoading(false);
      return;
    }

    try {
      const response = await authApi.register({
        email: formData.email,
        username: formData.username,
        password: formData.password,
        name: formData.name,
      });
      setAuth(response.token, response.user, response.refresh_token);

      // Redirect to email verification page
      router.push(`/verify-email?email=${encodeURIComponent(formData.email)}`);
    } catch (err: unknown) {
      if (err && typeof err === "object" && "data" in err) {
        const apiErr = err as { data?: { error?: string } };
        if (apiErr.data?.error) {
          setError(apiErr.data.error);
        } else {
          setError(t("auth.registerPage.registrationFailed"));
        }
      } else {
        setError(t("auth.registerPage.registrationFailed"));
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4 py-8">
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
            {t("auth.registerPage.title")}
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">
            {t("auth.registerPage.subtitle")}
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
            <label htmlFor="name" className="text-sm font-medium text-foreground">
              {t("auth.registerPage.fullName")}
            </label>
            <Input
              id="name"
              name="name"
              type="text"
              placeholder={t("auth.registerPage.fullNamePlaceholder")}
              value={formData.name}
              onChange={handleChange}
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="email" className="text-sm font-medium text-foreground">
              {t("auth.registerPage.emailLabel")}
            </label>
            <Input
              id="email"
              name="email"
              type="email"
              placeholder={t("auth.registerPage.emailPlaceholder")}
              value={formData.email}
              onChange={handleChange}
              required
            />
          </div>

          <div className="space-y-2">
            <label htmlFor="username" className="text-sm font-medium text-foreground">
              {t("auth.registerPage.usernameLabel")}
            </label>
            <Input
              id="username"
              name="username"
              type="text"
              placeholder={t("auth.registerPage.usernamePlaceholder")}
              value={formData.username}
              onChange={handleChange}
              required
              minLength={3}
              maxLength={50}
            />
            <p className="text-xs text-muted-foreground">
              {t("auth.registerPage.usernameHint")}
            </p>
          </div>

          <div className="space-y-2">
            <label htmlFor="password" className="text-sm font-medium text-foreground">
              {t("auth.registerPage.passwordLabel")}
            </label>
            <Input
              id="password"
              name="password"
              type="password"
              placeholder={t("auth.registerPage.passwordPlaceholder")}
              value={formData.password}
              onChange={handleChange}
              required
              minLength={8}
            />
            <p className="text-xs text-muted-foreground">
              {t("auth.registerPage.passwordHint")}
            </p>
          </div>

          <div className="space-y-2">
            <label
              htmlFor="confirmPassword"
              className="text-sm font-medium text-foreground"
            >
              {t("auth.registerPage.confirmPasswordLabel")}
            </label>
            <Input
              id="confirmPassword"
              name="confirmPassword"
              type="password"
              placeholder={t("auth.registerPage.passwordPlaceholder")}
              value={formData.confirmPassword}
              onChange={handleChange}
              required
            />
          </div>

          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? t("auth.registerPage.creatingAccount") : t("auth.registerPage.createAccount")}
          </Button>
        </form>

        {/* Terms */}
        <p className="text-center text-xs text-muted-foreground">
          {t("auth.registerPage.termsText")}{" "}
          <Link href="/terms" className="text-primary hover:underline">
            {t("auth.registerPage.termsOfService")}
          </Link>{" "}
          {t("auth.registerPage.and")}{" "}
          <Link href="/privacy" className="text-primary hover:underline">
            {t("auth.registerPage.privacyPolicy")}
          </Link>
        </p>

        {/* Divider */}
        <div className="relative">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-border" />
          </div>
          <div className="relative flex justify-center text-xs uppercase">
            <span className="bg-background px-2 text-muted-foreground">
              {t("auth.registerPage.orContinueWith")}
            </span>
          </div>
        </div>

        {/* OAuth */}
        <div className="grid grid-cols-2 gap-3">
          <Button
            variant="outline"
            type="button"
            onClick={() => {
              const oauthUrl = getOAuthBaseUrl();
              const redirectUrl = encodeURIComponent(window.location.origin + "/auth/callback");
              window.location.href = `${oauthUrl}/api/v1/auth/oauth/github?redirect=${redirectUrl}`;
            }}
          >
            <svg className="w-4 h-4 mr-2" viewBox="0 0 24 24">
              <path
                fill="currentColor"
                d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"
              />
            </svg>
            GitHub
          </Button>
          <Button
            variant="outline"
            type="button"
            onClick={() => (window.location.href = "/api/auth/oauth/google")}
          >
            <svg className="w-4 h-4 mr-2" viewBox="0 0 24 24">
              <path
                fill="currentColor"
                d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
              />
              <path
                fill="currentColor"
                d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
              />
              <path
                fill="currentColor"
                d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
              />
              <path
                fill="currentColor"
                d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
              />
            </svg>
            Google
          </Button>
        </div>

        {/* Login link */}
        <p className="text-center text-sm text-muted-foreground">
          {t("auth.registerPage.alreadyHaveAccount")}{" "}
          <Link href="/login" className="text-primary hover:underline">
            {t("auth.registerPage.signIn")}
          </Link>
        </p>
      </div>
    </div>
  );
}
