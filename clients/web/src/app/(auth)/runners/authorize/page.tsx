"use client";

import { useEffect, useState, useCallback, useMemo } from "react";
import { useSearchParams } from "next/navigation";
import {
  lightFetchMe,
  lightGetRunnerAuthStatus,
  lightAuthorizeRunner,
  lightListOrganizations,
  type LightOrganization,
} from "@/lib/light-auth";
import { useLightSession } from "@/hooks/useLightSession";
import type { RunnerAuthStatus } from "@/lib/api/runnerTypes";
import { ApiError } from "@/lib/api/api-types";
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

  const { session, hydrated } = useLightSession();
  const isSignedIn = !!session?.isAuthenticated;

  const [authStatus, setAuthStatus] = useState<RunnerAuthStatus | null>(null);
  const [meEmail, setMeEmail] = useState<string | undefined>(undefined);
  const [organizations, setOrganizations] = useState<LightOrganization[]>([]);
  const [selectedOrg, setSelectedOrg] = useState<LightOrganization | null>(null);
  const [nodeIdInput, setNodeIdInput] = useState("");
  const [loading, setLoading] = useState(true);
  const [authorizing, setAuthorizing] = useState(false);
  const [authorized, setAuthorized] = useState(false);
  const [error, setError] = useState("");

  const fetchAuthStatus = useCallback(async () => {
    if (!authKey) { setError(t("missingAuthKey")); setLoading(false); return; }
    try {
      const status = await lightGetRunnerAuthStatus(authKey);
      setAuthStatus(status);
      if (status.node_id) setNodeIdInput(status.node_id);
    } catch { setError(t("invalidAuthKey")); }
    finally { setLoading(false); }
  }, [authKey, t]);

  const fetchUserAndOrgs = useCallback(async () => {
    if (!isSignedIn) return;
    const [me, orgs] = await Promise.all([
      lightFetchMe(),
      lightListOrganizations().catch(() => []),
    ]);
    setMeEmail(me?.email);
    setOrganizations(orgs);
    const adminOrg = orgs.find((org) => org.subscription_status === "active" || org.subscription_plan);
    setSelectedOrg(adminOrg || orgs[0] || null);
  }, [isSignedIn]);

  useEffect(() => { fetchAuthStatus(); }, [fetchAuthStatus]);
  useEffect(() => { fetchUserAndOrgs(); }, [fetchUserAndOrgs]);

  const handleAuthorize = async () => {
    if (!authKey || !selectedOrg) return;
    setAuthorizing(true);
    setError("");
    try {
      await lightAuthorizeRunner({
        organizationSlug: selectedOrg.slug,
        authKey,
        nodeId: nodeIdInput || undefined,
      });
      setAuthorized(true);
    } catch (err: unknown) {
      if (isApiErrorCode(err, "RUNNER_QUOTA_EXCEEDED")) setError(t("quotaExceeded"));
      else if (err instanceof ApiError && err.serverMessage) setError(err.serverMessage);
      else setError(t("authorizeFailed"));
    } finally { setAuthorizing(false); }
  };

  if (loading || !hydrated) return <LoadingScreen message={tCommon("loading")} />;

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
        <AuthForm isSignedIn={isSignedIn} userEmail={meEmail} authKey={authKey!}
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
