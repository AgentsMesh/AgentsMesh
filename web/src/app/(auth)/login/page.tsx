"use client";

import { useState, useCallback, useRef, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuthStore } from "@/stores/auth";
import { authApi, organizationApi } from "@/lib/api";
import { ApiError } from "@/lib/api/base";
import { ssoApi } from "@/lib/api/sso";
import type { SSOConfig } from "@/lib/api/sso";
import { useTranslations } from "next-intl";
import { Logo } from "@/components/common";
import { OAuthButtons } from "./OAuthButtons";
import { SSOSection } from "./SSOSection";
import { Divider } from "./Divider";

export default function LoginPage() {
  const router = useRouter();
  const t = useTranslations();
  const { setAuth, setOrganizations } = useAuthStore();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [ldapLoading, setLdapLoading] = useState(false);
  const [error, setError] = useState("");

  const [ssoConfigs, setSsoConfigs] = useState<SSOConfig[]>([]);
  const [ssoLoading, setSsoLoading] = useState(false);
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
    setSsoLoading(true);
    try {
      const response = await ssoApi.discover(emailValue);
      if (requestId === discoverRequestRef.current) {
        setSsoConfigs(response.configs || []);
      }
    } finally {
      if (requestId === discoverRequestRef.current) setSsoLoading(false);
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
    try {
      const orgsResponse = await organizationApi.list();
      if (orgsResponse.organizations && orgsResponse.organizations.length > 0) {
        setOrganizations(orgsResponse.organizations);
        router.push(`/${orgsResponse.organizations[0].slug}/workspace`);
      } else {
        router.push("/onboarding");
      }
    } catch { router.push("/onboarding"); }
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
      const response = await ssoApi.ldapAuth(ldapConfig.domain, username, pwd);
      setAuth(response.token, response.user, response.refresh_token);
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
      const response = await authApi.login(email, password);
      setAuth(response.token, response.user, response.refresh_token);
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

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm space-y-6">
        <div className="text-center">
          <Link href="/" className="inline-flex items-center gap-2">
            <div className="w-10 h-10 rounded-lg overflow-hidden"><Logo /></div>
            <span className="text-2xl font-bold text-foreground">AgentsMesh</span>
          </Link>
          <h1 className="mt-6 text-2xl font-semibold text-foreground">
            {t("auth.loginPage.title")}
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">
            {t("auth.loginPage.subtitle")}
          </p>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div className="p-3 text-sm text-destructive bg-destructive/10 rounded-md">
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

          {ssoLoading && (
            <div className="flex items-center justify-center py-2">
              <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />
              <span className="ml-2 text-xs text-muted-foreground">{t("auth.sso.checking")}</span>
            </div>
          )}

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
                  <Link href="/forgot-password" className="text-sm text-primary hover:underline">
                    {t("auth.forgotPassword")}
                  </Link>
                </div>
                <Input id="password" type="password"
                  placeholder={t("auth.loginPage.passwordPlaceholder")}
                  value={password} onChange={(e) => setPassword(e.target.value)} required />
              </div>
              <Button type="submit" className="w-full" loading={loading}>
                {t("auth.loginPage.signIn")}
              </Button>
            </>
          )}
        </form>

        {!enforceSso && (
          <>
            <Divider text={t("auth.loginPage.orContinueWith")} />
            <OAuthButtons />
          </>
        )}

        <p className="text-center text-sm text-muted-foreground">
          {t("auth.loginPage.dontHaveAccount")}{" "}
          <Link href="/register" className="text-primary hover:underline">
            {t("auth.loginPage.signUp")}
          </Link>
        </p>
      </div>
    </div>
  );
}
