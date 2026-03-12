"use client";

import { useState, useCallback, useRef, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuthStore } from "@/stores/auth";
import { authApi, organizationApi } from "@/lib/api";
import { ApiError } from "@/lib/api/base";
import { ssoApi, getSSOAuthURL } from "@/lib/api/sso";
import type { SSOConfig } from "@/lib/api/sso";
import { useTranslations } from "next-intl";
import { getOAuthBaseUrl } from "@/lib/env";
import { Logo } from "@/components/common";

export default function LoginPage() {
  const router = useRouter();
  const t = useTranslations();
  const { setAuth, setOrganizations } = useAuthStore();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [loading, setLoading] = useState(false);
  const [ldapLoading, setLdapLoading] = useState(false);
  const [ldapExpanded, setLdapExpanded] = useState(false);
  const [error, setError] = useState("");

  // SSO state
  const [ssoConfigs, setSsoConfigs] = useState<SSOConfig[]>([]);
  const [ssoLoading, setSsoLoading] = useState(false);
  const [ldapUsername, setLdapUsername] = useState("");
  const [ldapPassword, setLdapPassword] = useState("");
  const debounceRef = useRef<ReturnType<typeof setTimeout>>(null);

  const enforceSso = ssoConfigs.some((c) => c.enforce_sso);
  const ldapConfig = ssoConfigs.find((c) => c.protocol === "ldap");
  const redirectConfigs = ssoConfigs.filter(
    (c) => c.protocol === "oidc" || c.protocol === "saml"
  );
  const hasSSO = ssoConfigs.length > 0;

  // SSO discovery on email blur with debounce + race condition guard
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
      // Only apply result if this is still the latest request
      if (requestId === discoverRequestRef.current) {
        setSsoConfigs(response.configs || []);
      }
    } finally {
      if (requestId === discoverRequestRef.current) {
        setSsoLoading(false);
      }
    }
  }, []);

  const handleEmailBlur = useCallback(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => discoverSSO(email), 500);
  }, [email, discoverSSO]);

  // Clean up debounce on unmount
  useEffect(() => {
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
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
    } catch {
      router.push("/onboarding");
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    // When SSO is enforced and LDAP is configured, Enter key in LDAP fields
    // triggers form submit. Delegate to LDAP handler instead of password login.
    if (enforceSso && ldapConfig) {
      handleLdapSubmit(e);
      return;
    }
    setLoading(true);
    setError("");

    try {
      const response = await authApi.login(email, password);
      setAuth(response.token, response.user, response.refresh_token);
      await navigateAfterLogin();
    } catch (err) {
      if (err instanceof ApiError && err.hasCode("SSO_REQUIRED")) {
        setError(t("auth.sso.ssoRequired"));
        // Trigger SSO discovery so user sees available SSO options
        discoverSSO(email);
      } else {
        setError(t("auth.loginPage.invalidCredentials"));
      }
    } finally {
      setLoading(false);
    }
  };

  const handleLdapSubmit = async (e: React.FormEvent | React.MouseEvent) => {
    e.preventDefault();
    if (!ldapConfig) return;
    if (!ldapUsername.trim() || !ldapPassword) {
      setError(t("auth.loginPage.invalidCredentials"));
      return;
    }
    setLdapLoading(true);
    setError("");

    try {
      const response = await ssoApi.ldapAuth(ldapConfig.domain, ldapUsername, ldapPassword);
      setAuth(response.token, response.user, response.refresh_token);
      await navigateAfterLogin();
    } catch (err) {
      if (err instanceof ApiError && err.status >= 500) {
        setError(t("common.error"));
      } else {
        setError(t("auth.loginPage.invalidCredentials"));
      }
    } finally {
      setLdapLoading(false);
    }
  };

  const handleSSORedirect = (config: SSOConfig) => {
    // Don't pass redirect — let the backend use its configured FrontendURL.
    // This avoids mismatch when window.location.origin differs from PRIMARY_DOMAIN
    // (e.g., dev environment with separate frontend port).
    window.location.href = getSSOAuthURL(config.domain, config.protocol);
  };

  // Shared divider component
  const Divider = ({ text }: { text: string }) => (
    <div className="relative">
      <div className="absolute inset-0 flex items-center">
        <div className="w-full border-t border-border" />
      </div>
      <div className="relative flex justify-center text-xs uppercase">
        <span className="bg-background px-2 text-muted-foreground">{text}</span>
      </div>
    </div>
  );

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm space-y-6">
        {/* Header */}
        <div className="text-center">
          <Link href="/" className="inline-flex items-center gap-2">
            <div className="w-10 h-10 rounded-lg overflow-hidden">
              <Logo />
            </div>
            <span className="text-2xl font-bold text-foreground">AgentsMesh</span>
          </Link>
          <h1 className="mt-6 text-2xl font-semibold text-foreground">
            {t("auth.loginPage.title")}
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">
            {t("auth.loginPage.subtitle")}
          </p>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <div className="p-3 text-sm text-destructive bg-destructive/10 rounded-md">
              {error}
            </div>
          )}

          {/* Email field */}
          <div className="space-y-2">
            <label htmlFor="email" className="text-sm font-medium text-foreground">
              {t("auth.loginPage.emailLabel")}
            </label>
            <Input
              id="email"
              type="email"
              placeholder={t("auth.loginPage.emailPlaceholder")}
              value={email}
              onChange={(e) => {
                setEmail(e.target.value);
                if (debounceRef.current) clearTimeout(debounceRef.current);
                if (ssoConfigs.length > 0) {
                  setSsoConfigs([]);
                }
              }}
              onBlur={handleEmailBlur}
              required
            />
          </div>

          {/* SSO discovery loading */}
          {ssoLoading && (
            <div className="flex items-center justify-center py-2">
              <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />
              <span className="ml-2 text-xs text-muted-foreground">
                {t("auth.sso.checking")}
              </span>
            </div>
          )}

          {/* SSO section — all SSO options grouped in one card */}
          {hasSSO && (
            <div className="rounded-lg border border-border bg-muted/30 p-4 space-y-3">
              <p className="text-xs font-medium text-muted-foreground text-center uppercase tracking-wide">
                {t("auth.sso.orSignInWithSSO")}
              </p>

              {/* OIDC / SAML redirect buttons */}
              {redirectConfigs.map((config) => (
                <Button
                  key={`${config.domain}-${config.protocol}`}
                  type="button"
                  variant="outline"
                  className="w-full"
                  onClick={() => handleSSORedirect(config)}
                >
                  {t("auth.sso.signInWith", { name: config.name })}
                </Button>
              ))}

              {/* LDAP form — collapsible panel */}
              {ldapConfig && (
                <>
                  {redirectConfigs.length > 0 && (
                    <div className="border-t border-border" />
                  )}
                  <button
                    type="button"
                    className="flex w-full items-center justify-between rounded-md px-2 py-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
                    onClick={() => setLdapExpanded((v) => !v)}
                  >
                    <span>{t("auth.sso.signInWith", { name: ldapConfig.name })}</span>
                    <svg
                      className={`h-4 w-4 transition-transform ${ldapExpanded ? "rotate-180" : ""}`}
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                      strokeWidth={2}
                    >
                      <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
                    </svg>
                  </button>
                  {ldapExpanded && (
                    <div className="space-y-3">
                      <Input
                        id="ldap-username"
                        type="text"
                        placeholder={t("auth.sso.ldapUsernamePlaceholder")}
                        value={ldapUsername}
                        onChange={(e) => setLdapUsername(e.target.value)}
                      />
                      <Input
                        id="ldap-password"
                        type="password"
                        placeholder={t("auth.loginPage.passwordPlaceholder")}
                        value={ldapPassword}
                        onChange={(e) => setLdapPassword(e.target.value)}
                      />
                      <Button
                        type="button"
                        variant="outline"
                        className="w-full"
                        loading={ldapLoading}
                        onClick={handleLdapSubmit}
                      >
                        {t("auth.sso.signInWith", { name: ldapConfig.name })}
                      </Button>
                    </div>
                  )}
                </>
              )}
            </div>
          )}

          {/* Password field and submit — hidden when SSO is enforced */}
          {!enforceSso && (
            <>
              {hasSSO && <Divider text={t("auth.sso.orUsePassword")} />}

              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <label htmlFor="password" className="text-sm font-medium text-foreground">
                    {t("auth.loginPage.passwordLabel")}
                  </label>
                  <Link
                    href="/forgot-password"
                    className="text-sm text-primary hover:underline"
                  >
                    {t("auth.forgotPassword")}
                  </Link>
                </div>
                <Input
                  id="password"
                  type="password"
                  placeholder={t("auth.loginPage.passwordPlaceholder")}
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                />
              </div>

              <Button type="submit" className="w-full" loading={loading}>
                {t("auth.loginPage.signIn")}
              </Button>
            </>
          )}
        </form>

        {/* OAuth section — hidden when SSO is enforced */}
        {!enforceSso && (
          <>
            <Divider text={t("auth.loginPage.orContinueWith")} />
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
                onClick={() => {
                  const oauthUrl = getOAuthBaseUrl();
                  const redirectUrl = encodeURIComponent(window.location.origin + "/auth/callback");
                  window.location.href = `${oauthUrl}/api/v1/auth/oauth/google?redirect=${redirectUrl}`;
                }}
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
          </>
        )}

        {/* Register link */}
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
