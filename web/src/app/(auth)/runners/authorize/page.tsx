"use client";

import { useEffect, useState, useCallback, useMemo } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { useAuthStore } from "@/stores/auth";
import { runnerAuthApi, RunnerAuthStatus } from "@/lib/api/runner";
import { organizationApi, OrganizationData } from "@/lib/api/organization";
import { useTranslations } from "@/lib/i18n/client";

export default function RunnerAuthorizePage() {
  const rawT = useTranslations();
  // Create scoped translation functions
  const t = useMemo(() => (key: string, params?: Record<string, string | number>) => rawT(`runners.authorize.${key}`, params), [rawT]);
  const tCommon = useMemo(() => (key: string) => rawT(`common.${key}`), [rawT]);
  const searchParams = useSearchParams();
  const authKey = searchParams.get("key");

  const { token: authToken, user, organizations, setOrganizations, _hasHydrated } = useAuthStore();

  const [authStatus, setAuthStatus] = useState<RunnerAuthStatus | null>(null);
  const [selectedOrg, setSelectedOrg] = useState<OrganizationData | null>(null);
  const [nodeIdInput, setNodeIdInput] = useState("");
  const [loading, setLoading] = useState(true);
  const [authorizing, setAuthorizing] = useState(false);
  const [authorized, setAuthorized] = useState(false);
  const [error, setError] = useState("");

  // Fetch auth status
  const fetchAuthStatus = useCallback(async () => {
    if (!authKey) {
      setError(t("missingAuthKey"));
      setLoading(false);
      return;
    }

    try {
      const status = await runnerAuthApi.getAuthStatus(authKey);
      setAuthStatus(status);

      // Pre-fill node_id if available
      if (status.node_id) {
        setNodeIdInput(status.node_id);
      }
    } catch {
      setError(t("invalidAuthKey"));
    } finally {
      setLoading(false);
    }
  }, [authKey, t]);

  // Fetch organizations if authenticated
  const fetchOrganizations = useCallback(async () => {
    if (!authToken) return;

    try {
      const { organizations: orgs } = await organizationApi.list();
      setOrganizations(orgs);

      // Auto-select first org with admin/owner role
      const adminOrg = orgs.find(
        (org) => org.subscription_status === "active" || org.subscription_plan
      );
      if (adminOrg) {
        setSelectedOrg(adminOrg);
      } else if (orgs.length > 0) {
        setSelectedOrg(orgs[0]);
      }
    } catch {
      // Ignore org fetch errors
    }
  }, [authToken, setOrganizations]);

  useEffect(() => {
    fetchAuthStatus();
  }, [fetchAuthStatus]);

  useEffect(() => {
    if (authToken) {
      fetchOrganizations();
    }
  }, [authToken, fetchOrganizations]);

  // Handle authorization
  const handleAuthorize = async () => {
    if (!authKey || !selectedOrg) return;

    setAuthorizing(true);
    setError("");

    try {
      await runnerAuthApi.authorize(
        selectedOrg.slug,
        authKey,
        nodeIdInput || undefined
      );
      setAuthorized(true);
    } catch (err: unknown) {
      if (err && typeof err === "object" && "data" in err) {
        const apiErr = err as { data?: { error?: string; code?: string } };
        if (apiErr.data?.code === "RUNNER_QUOTA_EXCEEDED") {
          setError(t("quotaExceeded"));
        } else {
          setError(apiErr.data?.error || t("authorizeFailed"));
        }
      } else {
        setError(t("authorizeFailed"));
      }
    } finally {
      setAuthorizing(false);
    }
  };

  // Logo component
  const Logo = () => (
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
  );

  // Loading state (also wait for auth store hydration)
  if (loading || !_hasHydrated) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background px-4">
        <div className="w-full max-w-md space-y-6 text-center">
          <div className="flex justify-center">
            <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
          </div>
          <p className="text-sm text-muted-foreground">{tCommon("loading")}</p>
        </div>
      </div>
    );
  }

  // Missing auth key
  if (!authKey || (error && !authStatus)) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background px-4">
        <div className="w-full max-w-md space-y-6 text-center">
          <Logo />

          {/* Error Icon */}
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
            <h1 className="text-2xl font-semibold text-foreground">{t("invalidTitle")}</h1>
            <p className="text-sm text-muted-foreground">{error || t("invalidAuthKey")}</p>
          </div>

          <Link href="/login">
            <Button className="w-full">{t("goToLogin")}</Button>
          </Link>
        </div>
      </div>
    );
  }

  // Expired authorization
  if (authStatus?.status === "expired") {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background px-4">
        <div className="w-full max-w-md space-y-6 text-center">
          <Logo />

          {/* Warning Icon */}
          <div className="flex justify-center">
            <div className="w-16 h-16 rounded-full bg-amber-100 dark:bg-amber-900/30 flex items-center justify-center">
              <svg
                className="w-8 h-8 text-amber-600 dark:text-amber-400"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
            </div>
          </div>

          <div className="space-y-2">
            <h1 className="text-2xl font-semibold text-foreground">{t("expiredTitle")}</h1>
            <p className="text-sm text-muted-foreground">{t("expiredDescription")}</p>
          </div>

          <p className="text-sm text-muted-foreground">{t("rerunCommand")}</p>
        </div>
      </div>
    );
  }

  // Already authorized
  if (authStatus?.status === "authorized" || authorized) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background px-4">
        <div className="w-full max-w-md space-y-6 text-center">
          <Logo />

          {/* Success Icon */}
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
            <h1 className="text-2xl font-semibold text-foreground">{t("successTitle")}</h1>
            <p className="text-sm text-muted-foreground">{t("successDescription")}</p>
          </div>

          <p className="text-sm text-muted-foreground">{t("closeWindow")}</p>
        </div>
      </div>
    );
  }

  // Pending authorization - show authorization form
  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-md space-y-6">
        {/* Logo */}
        <div className="text-center">
          <Logo />
        </div>

        {/* Authorization Card */}
        <div className="p-6 border border-border rounded-lg space-y-4">
          {/* Runner Icon */}
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
                  d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"
                />
              </svg>
            </div>
          </div>

          {/* Title */}
          <div className="text-center space-y-2">
            <h1 className="text-xl font-semibold text-foreground">{t("title")}</h1>
            <p className="text-sm text-muted-foreground">{t("description")}</p>
          </div>

          {/* Error message */}
          {error && (
            <div className="p-3 text-sm text-destructive bg-destructive/10 rounded-md">
              {error}
            </div>
          )}

          {/* Not authenticated - redirect to login */}
          {!authToken || !user ? (
            <div className="space-y-3">
              <p className="text-sm text-center text-muted-foreground">{t("signInRequired")}</p>
              <Link href={`/login?redirect=/runners/authorize?key=${authKey}`}>
                <Button className="w-full">{t("signInToAuthorize")}</Button>
              </Link>
              <p className="text-sm text-center text-muted-foreground">
                {t("noAccount")}{" "}
                <Link
                  href={`/register?redirect=/runners/authorize?key=${authKey}`}
                  className="text-primary hover:underline"
                >
                  {t("signUp")}
                </Link>
              </p>
            </div>
          ) : (
            <div className="space-y-4">
              <p className="text-sm text-center text-muted-foreground">
                {t("signedInAs")} <strong>{user.email}</strong>
              </p>

              {/* Organization selector */}
              {organizations && organizations.length > 0 ? (
                <div className="space-y-2">
                  <label className="text-sm font-medium text-foreground">
                    {t("selectOrganization")}
                  </label>
                  <select
                    className="w-full px-3 py-2 border border-input rounded-md bg-background text-foreground"
                    value={selectedOrg?.id || ""}
                    onChange={(e) => {
                      const org = organizations.find(
                        (o) => o.id === parseInt(e.target.value)
                      );
                      setSelectedOrg(org || null);
                    }}
                  >
                    <option value="" disabled>
                      {t("selectOrgPlaceholder")}
                    </option>
                    {organizations.map((org) => (
                      <option key={org.id} value={org.id}>
                        {org.name}
                      </option>
                    ))}
                  </select>
                </div>
              ) : (
                <div className="p-3 text-sm text-amber-600 dark:text-amber-400 bg-amber-100 dark:bg-amber-900/30 rounded-md">
                  {t("noOrganizations")}
                </div>
              )}

              {/* Node ID input (optional) */}
              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground">
                  {t("nodeIdLabel")} <span className="text-muted-foreground">({tCommon("optional")})</span>
                </label>
                <input
                  type="text"
                  className="w-full px-3 py-2 border border-input rounded-md bg-background text-foreground placeholder:text-muted-foreground"
                  placeholder={t("nodeIdPlaceholder")}
                  value={nodeIdInput}
                  onChange={(e) => setNodeIdInput(e.target.value)}
                />
                <p className="text-xs text-muted-foreground">{t("nodeIdHint")}</p>
              </div>

              {/* Authorize button */}
              <Button
                className="w-full"
                onClick={handleAuthorize}
                disabled={authorizing || !selectedOrg}
              >
                {authorizing ? t("authorizing") : t("authorizeButton")}
              </Button>
            </div>
          )}
        </div>

        {/* Expiration notice */}
        {authStatus?.expires_at && (
          <p className="text-center text-xs text-muted-foreground">
            {t("expiresAt", {
              time: new Date(authStatus.expires_at).toLocaleTimeString(),
            })}
          </p>
        )}
      </div>
    </div>
  );
}
