"use client";

import { useEffect, useState, useCallback, useMemo } from "react";
import { useSearchParams } from "next/navigation";
import { useAuthStore } from "@/stores/auth";
import { runnerAuthApi, RunnerAuthStatus } from "@/lib/api/runner";
import { organizationApi, OrganizationData } from "@/lib/api/organization";
import { ApiError } from "@/lib/api/base";
import { isApiErrorCode } from "@/lib/api/errors";
import { useTranslations } from "next-intl";
import { LoadingScreen, ErrorScreen, ExpiredScreen, SuccessScreen, BrandLogo } from "./StatusScreens";
import { AuthForm } from "./AuthForm";

export default function RunnerAuthorizePage() {
  const rawT = useTranslations();
  const t = useMemo(() => (key: string, params?: Record<string, string | number>) =>
    rawT(`runners.authorize.${key}`, params), [rawT]);
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

  const fetchAuthStatus = useCallback(async () => {
    if (!authKey) { setError(t("missingAuthKey")); setLoading(false); return; }
    try {
      const status = await runnerAuthApi.getAuthStatus(authKey);
      setAuthStatus(status);
      if (status.node_id) setNodeIdInput(status.node_id);
    } catch { setError(t("invalidAuthKey")); }
    finally { setLoading(false); }
  }, [authKey, t]);

  const fetchOrganizations = useCallback(async () => {
    if (!authToken) return;
    try {
      const { organizations: orgs } = await organizationApi.list();
      setOrganizations(orgs);
      const adminOrg = orgs.find((org) => org.subscription_status === "active" || org.subscription_plan);
      setSelectedOrg(adminOrg || orgs[0] || null);
    } catch { /* ignore */ }
  }, [authToken, setOrganizations]);

  useEffect(() => { fetchAuthStatus(); }, [fetchAuthStatus]);
  useEffect(() => { if (authToken) fetchOrganizations(); }, [authToken, fetchOrganizations]);

  const handleAuthorize = async () => {
    if (!authKey || !selectedOrg) return;
    setAuthorizing(true);
    setError("");
    try {
      await runnerAuthApi.authorize(selectedOrg.slug, authKey, nodeIdInput || undefined);
      setAuthorized(true);
    } catch (err: unknown) {
      if (isApiErrorCode(err, "RUNNER_QUOTA_EXCEEDED")) setError(t("quotaExceeded"));
      else if (err instanceof ApiError && err.serverMessage) setError(err.serverMessage);
      else setError(t("authorizeFailed"));
    } finally { setAuthorizing(false); }
  };

  if (loading || !_hasHydrated) return <LoadingScreen message={tCommon("loading")} />;

  if (!authKey || (error && !authStatus)) {
    return <ErrorScreen error={error || t("invalidAuthKey")} loginLabel={t("goToLogin")} />;
  }

  if (authStatus?.status === "expired") {
    return <ExpiredScreen title={t("expiredTitle")} description={t("expiredDescription")} hint={t("rerunCommand")} />;
  }

  if (authStatus?.status === "authorized" || authorized) {
    return <SuccessScreen title={t("successTitle")} description={t("successDescription")} hint={t("closeWindow")} />;
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-md space-y-6">
        <div className="text-center"><BrandLogo /></div>
        <AuthForm authToken={authToken} userEmail={user?.email} authKey={authKey!}
          organizations={organizations} selectedOrg={selectedOrg} onSelectOrg={setSelectedOrg}
          nodeIdInput={nodeIdInput} onNodeIdChange={setNodeIdInput} authorizing={authorizing}
          onAuthorize={handleAuthorize} error={error} t={t} tCommon={tCommon} />
        {authStatus?.expires_at && (
          <p className="text-center text-xs text-muted-foreground">
            {t("expiresAt", { time: new Date(authStatus.expires_at).toLocaleTimeString() })}
          </p>
        )}
      </div>
    </div>
  );
}
