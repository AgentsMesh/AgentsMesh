"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ApiError } from "@/lib/api/api-types";
import { lightRegister } from "@/lib/light-auth";
import { useRedirectIfAuthenticated } from "@/hooks/useRedirectIfAuthenticated";
import { useTranslations } from "next-intl";
import { AuthShell } from "@/components/auth/AuthShell";
import { OAuthButtons } from "../login/OAuthButtons";
import { Divider } from "../login/Divider";

export default function RegisterPage() {
  const router = useRouter();
  const t = useTranslations();
  useRedirectIfAuthenticated();

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
    setFormData((prev) => ({ ...prev, [e.target.name]: e.target.value }));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");

    if (formData.password !== formData.confirmPassword) {
      setError(t("auth.registerPage.passwordsNotMatch"));
      return;
    }
    if (formData.password.length < 8) {
      setError(t("auth.registerPage.passwordTooShort"));
      return;
    }

    setLoading(true);
    try {
      await lightRegister({
        email: formData.email,
        username: formData.username,
        password: formData.password,
        name: formData.name,
      });
      router.push(`/verify-email?email=${encodeURIComponent(formData.email)}`);
    } catch (err: unknown) {
      setError(err instanceof ApiError && err.serverMessage
        ? err.serverMessage
        : t("auth.registerPage.registrationFailed"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthShell
      title={t("auth.registerPage.title")}
      subtitle={t("auth.registerPage.subtitle")}
      footer={
        <>
          {t("auth.registerPage.alreadyHaveAccount")}{" "}
          <Link href="/login" className="text-[var(--azure-cyan)] hover:underline">
            {t("auth.registerPage.signIn")}
          </Link>
        </>
      }
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        {error && (
          <div className="p-3 text-sm text-destructive bg-destructive/10 border border-destructive/20 rounded-lg">
            {error}
          </div>
        )}

        <div className="space-y-2">
          <label htmlFor="name" className="text-sm font-medium text-foreground">
            {t("auth.registerPage.fullName")}
          </label>
          <Input id="name" name="name" type="text"
            placeholder={t("auth.registerPage.fullNamePlaceholder")}
            value={formData.name} onChange={handleChange} />
        </div>

        <div className="space-y-2">
          <label htmlFor="email" className="text-sm font-medium text-foreground">
            {t("auth.registerPage.emailLabel")}
          </label>
          <Input id="email" name="email" type="email"
            placeholder={t("auth.registerPage.emailPlaceholder")}
            value={formData.email} onChange={handleChange} required />
        </div>

        <div className="space-y-2">
          <label htmlFor="username" className="text-sm font-medium text-foreground">
            {t("auth.registerPage.usernameLabel")}
          </label>
          <Input id="username" name="username" type="text"
            placeholder={t("auth.registerPage.usernamePlaceholder")}
            value={formData.username} onChange={handleChange}
            pattern="[a-zA-Z0-9_-]+"
            required minLength={3} maxLength={50} />
          <p className="text-xs text-[var(--azure-text-muted)]">
            {t("auth.registerPage.usernameHint")}
          </p>
        </div>

        <div className="space-y-2">
          <label htmlFor="password" className="text-sm font-medium text-foreground">
            {t("auth.registerPage.passwordLabel")}
          </label>
          <Input id="password" name="password" type="password"
            placeholder={t("auth.registerPage.passwordPlaceholder")}
            value={formData.password} onChange={handleChange}
            required minLength={8} />
          <p className="text-xs text-[var(--azure-text-muted)]">
            {t("auth.registerPage.passwordHint")}
          </p>
        </div>

        <div className="space-y-2">
          <label htmlFor="confirmPassword" className="text-sm font-medium text-foreground">
            {t("auth.registerPage.confirmPasswordLabel")}
          </label>
          <Input id="confirmPassword" name="confirmPassword" type="password"
            placeholder={t("auth.registerPage.passwordPlaceholder")}
            value={formData.confirmPassword} onChange={handleChange} required />
        </div>

        <Button type="submit" className="w-full azure-gradient-bg hover:opacity-90 font-headline font-bold uppercase tracking-wider" disabled={loading}>
          {loading ? t("auth.registerPage.creatingAccount") : t("auth.registerPage.createAccount")}
        </Button>
      </form>

      <p className="mt-6 text-center text-xs text-[var(--azure-text-muted)]">
        {t("auth.registerPage.termsText")}{" "}
        <Link href="/terms" className="text-[var(--azure-cyan)] hover:underline">
          {t("auth.registerPage.termsOfService")}
        </Link>{" "}
        {t("auth.registerPage.and")}{" "}
        <Link href="/privacy" className="text-[var(--azure-cyan)] hover:underline">
          {t("auth.registerPage.privacyPolicy")}
        </Link>
      </p>

      <div className="mt-6 space-y-4">
        <Divider text={t("auth.registerPage.orContinueWith")} />
        <OAuthButtons />
      </div>
    </AuthShell>
  );
}
