"use client";

import { useState, useCallback, useRef, useEffect } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ApiError } from "@/lib/api/api-types";
import type { SSOConfig } from "@/lib/api/ssoTypes";
import {
  lightLogin,
  lightLdapAuth,
  lightDiscoverSSO,
  resolvePostLoginUrlLight,
} from "@/lib/light-auth";
import { useRedirectIfAuthenticated } from "@/hooks/useRedirectIfAuthenticated";
import { useTranslations } from "next-intl";
import { AuthShell } from "@/components/auth/AuthShell";
import { OAuthButtons } from "./OAuthButtons";
import { SSOSection } from "./SSOSection";
import { Divider } from "./Divider";

export default function LoginPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const t = useTranslations();
  useRedirectIfAuthenticated();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [ldapLoading, setLdapLoading] = useState(false);
  const [error, setError] = useState("");

  const [ssoConfigs, setSsoConfigs] = useState<SSOConfig[]>([]);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(null);

  const enforceSso = ssoConfigs.some((c) => c.enforce_sso);
  const hasSSO = ssoConfigs.length > 0;

  const discoverRequestRef = useRef(0);
  const discoverSSO = useCallback(async (emailValue: string) => {
    if (!emailValue || !emailValue.includes("@")) {
      setSsoConfigs([]);
      return;
    }
    const requestId = ++discoverRequestRef.current;
    try {
      const configs = await lightDiscoverSSO(emailValue);
      if (requestId === discoverRequestRef.current) setSsoConfigs(configs);
    } catch {
      // Silent — SSO discovery failures shouldn't disrupt the password flow.
    }
  }, []);

  const handleEmailBlur = useCallback(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => discoverSSO(email), 500);
  }, [email, discoverSSO]);

  useEffect(() => {
    return () => { if (debounceRef.current) clearTimeout(debounceRef.current); };
  }, []);

  const navigateAfterLogin = async () => {
    const url = await resolvePostLoginUrlLight({
      redirectParam: searchParams.get("redirect"),
    });
    router.push(url);
  };

  const handleLdapSubmit = async (username: string, pwd: string) => {
    const ldapConfig = ssoConfigs.find((c) => c.protocol === "ldap");
    if (!ldapConfig || !username.trim() || !pwd) {
      setError(t("auth.loginPage.invalidCredentials"));
      return;
    }
    setLdapLoading(true);
    setError("");
    try {
      await lightLdapAuth({ domain: ldapConfig.domain, username, password: pwd });
      await navigateAfterLogin();
    } catch (err) {
      setError(err instanceof ApiError && err.status >= 500
        ? t("common.error") : t("auth.loginPage.invalidCredentials"));
    } finally { setLdapLoading(false); }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (enforceSso && ssoConfigs.find((c) => c.protocol === "ldap")) return;
    setLoading(true);
    setError("");
    try {
      await lightLogin({ email, password });
      await navigateAfterLogin();
    } catch (err) {
      if (err instanceof ApiError && err.hasCode("SSO_REQUIRED")) {
        setError(t("auth.sso.ssoRequired"));
        discoverSSO(email);
      } else {
        setError(t("auth.loginPage.invalidCredentials"));
      }
    } finally { setLoading(false); }
  };

  const redirectParam = searchParams.get("redirect");
  const registerHref = redirectParam
    ? `/register?redirect=${encodeURIComponent(redirectParam)}`
    : "/register";

  return (
    <AuthShell
      title={t("auth.loginPage.title")}
      subtitle={t("auth.loginPage.subtitle")}
      footer={
        <>
          {t("auth.loginPage.dontHaveAccount")}{" "}
          <Link href={registerHref} className="text-[var(--azure-cyan)] hover:underline">
            {t("auth.loginPage.signUp")}
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
          <label htmlFor="email" className="text-sm font-medium text-foreground">
            {t("auth.loginPage.emailLabel")}
          </label>
          <Input id="email" type="email" placeholder={t("auth.loginPage.emailPlaceholder")}
            value={email}
            onChange={(e) => {
              setEmail(e.target.value);
              if (debounceRef.current) clearTimeout(debounceRef.current);
              if (ssoConfigs.length > 0) setSsoConfigs([]);
            }}
            onBlur={handleEmailBlur} required />
        </div>

        {hasSSO && (
          <SSOSection ssoConfigs={ssoConfigs} onLdapSubmit={handleLdapSubmit}
            ldapLoading={ldapLoading} />
        )}

        {!enforceSso && (
          <>
            {hasSSO && <Divider text={t("auth.sso.orUsePassword")} />}
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <label htmlFor="password" className="text-sm font-medium text-foreground">
                  {t("auth.loginPage.passwordLabel")}
                </label>
                <Link href="/forgot-password" className="text-sm text-[var(--azure-cyan)] hover:underline">
                  {t("auth.forgotPassword")}
                </Link>
              </div>
              <Input id="password" type="password"
                placeholder={t("auth.loginPage.passwordPlaceholder")}
                value={password} onChange={(e) => setPassword(e.target.value)} required />
            </div>
            <Button type="submit" className="w-full azure-gradient-bg hover:opacity-90 font-headline font-bold uppercase tracking-wider" loading={loading}>
              {t("auth.loginPage.signIn")}
            </Button>
          </>
        )}
      </form>

      {!enforceSso && (
        <div className="mt-6 space-y-4">
          <Divider text={t("auth.loginPage.orContinueWith")} />
          <OAuthButtons />
        </div>
      )}
    </AuthShell>
  );
}
