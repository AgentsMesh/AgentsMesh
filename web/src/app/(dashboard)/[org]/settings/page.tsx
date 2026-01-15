"use client";

import { useState, useEffect, useCallback } from "react";
import { useSearchParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useAuthStore } from "@/stores/auth";
import { organizationApi, billingApi, BillingOverview, SubscriptionPlan, RedeemPromoCodeResponse } from "@/lib/api/client";
import { PromoCodeInput } from "@/components/promo-code/PromoCodeInput";
import { useRunnerStore, Runner, getRunnerStatusInfo } from "@/stores/runner";
import { LanguageSettings, NotificationSettings, AgentCredentialsSettings, AgentConfigPage } from "@/components/settings";
import { useTranslations } from "@/lib/i18n/client";
import { GitSettingsContent } from "@/components/settings/GitSettingsContent";

export default function SettingsPage() {
  const searchParams = useSearchParams();
  const scope = searchParams.get("scope") || "personal";
  const activeTab = searchParams.get("tab") || "general";
  const { currentOrg } = useAuthStore();
  const t = useTranslations();

  // Tab content mapping based on scope
  const renderContent = () => {
    // Personal settings
    if (scope === "personal") {
      // Handle agent config pages (agents/{slug})
      if (activeTab.startsWith("agents/")) {
        const agentSlug = activeTab.replace("agents/", "");
        return <AgentConfigPage agentSlug={agentSlug} />;
      }

      switch (activeTab) {
        case "general":
          return <PersonalGeneralSettings t={t} />;
        case "git":
          return <PersonalGitSettings t={t} />;
        case "agent-credentials":
          return <PersonalAgentCredentialsSettings t={t} />;
        case "notifications":
          return <PersonalNotificationsSettings t={t} />;
        default:
          return <PersonalGeneralSettings t={t} />;
      }
    }

    // Organization settings
    switch (activeTab) {
      case "general":
        return <GeneralSettings org={currentOrg} t={t} />;
      case "members":
        return <MembersSettings t={t} />;
      case "runners":
        return <RunnersSettings t={t} />;
      case "billing":
        return <BillingSettings t={t} />;
      default:
        return <GeneralSettings org={currentOrg} t={t} />;
    }
  };

  return (
    <div className="h-full overflow-auto p-6">
      {/* Content - navigation controlled by IDE Sidebar */}
      <div className="max-w-4xl">
        {renderContent()}
      </div>
    </div>
  );
}

// ===== Personal Settings Components =====

function PersonalGeneralSettings({ t }: { t: TranslationFn }) {
  return (
    <div className="space-y-6">
      {/* Language Settings */}
      <LanguageSettings />
    </div>
  );
}

function PersonalGitSettings({ t }: { t: TranslationFn }) {
  return <GitSettingsContent />;
}

function PersonalAgentCredentialsSettings({ t }: { t: TranslationFn }) {
  return (
    <div className="space-y-6">
      <div className="border border-border rounded-lg p-6">
        <AgentCredentialsSettings />
      </div>
    </div>
  );
}

function PersonalNotificationsSettings({ t }: { t: TranslationFn }) {
  return (
    <div className="space-y-6">
      <div className="border border-border rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-4">{t("settings.notifications.title")}</h2>
        <p className="text-sm text-muted-foreground mb-6">
          {t("settings.notifications.description")}
        </p>
        <NotificationSettings />
      </div>
    </div>
  );
}

type TranslationFn = (key: string, params?: Record<string, string | number>) => string;

function GeneralSettings({ org, t }: { org: { name: string; slug: string } | null; t: TranslationFn }) {
  const [name, setName] = useState(org?.name || "");
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    setSaving(true);
    try {
      await organizationApi.update(org!.slug, { name });
    } catch (error) {
      console.error("Failed to save:", error);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-6">
      <div className="border border-border rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-4">{t("settings.organizationDetails.title")}</h2>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">
              {t("settings.organizationDetails.nameLabel")}
            </label>
            <Input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t("settings.organizationDetails.namePlaceholder")}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">
              {t("settings.organizationDetails.slugLabel")}
            </label>
            <Input value={org?.slug || ""} disabled />
            <p className="text-xs text-muted-foreground mt-1">
              {t("settings.organizationDetails.slugHint")}
            </p>
          </div>
        </div>
        <div className="mt-6">
          <Button onClick={handleSave} disabled={saving}>
            {saving ? t("settings.organizationDetails.saving") : t("settings.organizationDetails.saveChanges")}
          </Button>
        </div>
      </div>

      <div className="border border-destructive rounded-lg p-6">
        <h2 className="text-lg font-semibold text-destructive mb-4">
          {t("settings.dangerZone.title")}
        </h2>
        <p className="text-sm text-muted-foreground mb-4">
          {t("settings.dangerZone.description")}
        </p>
        <Button variant="destructive">{t("settings.dangerZone.deleteOrg")}</Button>
      </div>
    </div>
  );
}

function MembersSettings({ t }: { t: TranslationFn }) {
  const { currentOrg, user } = useAuthStore();
  const [members, setMembers] = useState<Array<{
    id: number;
    user_id: number;
    role: string;
    joined_at: string;
    user?: { id: number; email: string; username: string; name?: string };
  }>>([]);
  const [loading, setLoading] = useState(true);
  const [showInviteDialog, setShowInviteDialog] = useState(false);
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState("member");
  const [inviting, setInviting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadMembers = useCallback(async () => {
    if (!currentOrg) return;
    try {
      setLoading(true);
      const response = await organizationApi.listMembers(currentOrg.slug);
      setMembers(response.members || []);
    } catch (err) {
      console.error("Failed to load members:", err);
      setError("Failed to load members");
    } finally {
      setLoading(false);
    }
  }, [currentOrg]);

  useEffect(() => {
    loadMembers();
  }, [loadMembers]);

  const handleInvite = async () => {
    if (!currentOrg || !inviteEmail) return;
    setInviting(true);
    setError(null);
    try {
      await organizationApi.inviteMember(currentOrg.slug, inviteEmail, inviteRole);
      setShowInviteDialog(false);
      setInviteEmail("");
      setInviteRole("member");
      await loadMembers();
    } catch (err) {
      console.error("Failed to invite member:", err);
      setError("Failed to invite member. Please check the email and try again.");
    } finally {
      setInviting(false);
    }
  };

  const handleRemove = async (userId: number) => {
    if (!currentOrg) return;
    if (!confirm("Are you sure you want to remove this member?")) return;
    try {
      await organizationApi.removeMember(currentOrg.slug, userId);
      await loadMembers();
    } catch (err) {
      console.error("Failed to remove member:", err);
      setError("Failed to remove member");
    }
  };

  const handleRoleChange = async (userId: number, newRole: string) => {
    if (!currentOrg) return;
    try {
      await organizationApi.updateMemberRole(currentOrg.slug, userId, newRole);
      await loadMembers();
    } catch (err) {
      console.error("Failed to update role:", err);
      setError("Failed to update member role");
    }
  };

  const getRoleBadgeColor = (role: string) => {
    switch (role) {
      case "owner": return "bg-purple-100 text-purple-800";
      case "admin": return "bg-blue-100 text-blue-800";
      default: return "bg-gray-100 text-gray-800";
    }
  };

  return (
    <div className="border border-border rounded-lg p-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <h2 className="text-lg font-semibold">{t("settings.members.title")}</h2>
          <p className="text-sm text-muted-foreground">
            {t("settings.members.description")}
          </p>
        </div>
        <Button onClick={() => setShowInviteDialog(true)}>{t("settings.members.inviteMember")}</Button>
      </div>

      {error && (
        <div className="bg-destructive/10 border border-destructive text-destructive px-4 py-3 rounded-lg mb-4">
          {error}
          <button onClick={() => setError(null)} className="ml-4 underline text-sm">
            {t("settings.members.dismiss")}
          </button>
        </div>
      )}

      {loading ? (
        <div className="text-center py-8 text-muted-foreground">{t("settings.members.loading")}</div>
      ) : members.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">
          {t("settings.members.noMembers")}
        </div>
      ) : (
        <div className="space-y-3">
          {members.map((member) => (
            <div
              key={member.id}
              className="flex items-center justify-between p-4 border border-border rounded-lg"
            >
              <div className="flex items-center gap-3">
                <div className="w-10 h-10 rounded-full bg-muted flex items-center justify-center text-sm font-medium">
                  {member.user?.name?.[0] || member.user?.username?.[0] || "?"}
                </div>
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-medium">
                      {member.user?.name || member.user?.username || "Unknown"}
                    </span>
                    <span className={`text-xs px-2 py-0.5 rounded-full ${getRoleBadgeColor(member.role)}`}>
                      {member.role}
                    </span>
                    {member.user_id === user?.id && (
                      <span className="text-xs text-muted-foreground">{t("settings.members.you")}</span>
                    )}
                  </div>
                  <p className="text-sm text-muted-foreground">{member.user?.email}</p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                {member.role !== "owner" && member.user_id !== user?.id && (
                  <>
                    <select
                      value={member.role}
                      onChange={(e) => handleRoleChange(member.user_id, e.target.value)}
                      className="text-sm border border-border rounded px-2 py-1 bg-background"
                    >
                      <option value="member">{t("settings.members.roleMember")}</option>
                      <option value="admin">{t("settings.members.roleAdmin")}</option>
                    </select>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      onClick={() => handleRemove(member.user_id)}
                    >
                      {t("settings.members.remove")}
                    </Button>
                  </>
                )}
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Invite Dialog */}
      {showInviteDialog && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-background border border-border rounded-lg p-6 w-full max-w-md">
            <h3 className="text-lg font-semibold mb-4">{t("settings.members.inviteDialog.title")}</h3>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium mb-2">{t("settings.members.inviteDialog.emailLabel")}</label>
                <Input
                  type="email"
                  value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                  placeholder={t("settings.members.inviteDialog.emailPlaceholder")}
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-2">{t("settings.members.inviteDialog.roleLabel")}</label>
                <select
                  value={inviteRole}
                  onChange={(e) => setInviteRole(e.target.value)}
                  className="w-full border border-border rounded px-3 py-2 bg-background"
                >
                  <option value="member">{t("settings.members.roleMember")}</option>
                  <option value="admin">{t("settings.members.roleAdmin")}</option>
                </select>
              </div>
            </div>
            <div className="flex gap-3 mt-6">
              <Button
                variant="outline"
                className="flex-1"
                onClick={() => {
                  setShowInviteDialog(false);
                  setInviteEmail("");
                  setInviteRole("member");
                }}
              >
                {t("settings.members.inviteDialog.cancel")}
              </Button>
              <Button
                className="flex-1"
                onClick={handleInvite}
                disabled={inviting || !inviteEmail}
              >
                {inviting ? t("settings.members.inviteDialog.inviting") : t("settings.members.inviteDialog.sendInvite")}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}


// NOTE: AgentsSettings has been removed - agent configuration moved to personal settings
// NOTE: GitProvidersSettings has been removed and moved to personal settings (/settings/git)

function BillingSettings({ t }: { t: TranslationFn }) {
  const [loading, setLoading] = useState(true);
  const [overview, setOverview] = useState<BillingOverview | null>(null);
  const [plans, setPlans] = useState<SubscriptionPlan[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [showPlansDialog, setShowPlansDialog] = useState(false);
  const [upgrading, setUpgrading] = useState(false);

  const loadBillingData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [overviewRes, plansRes] = await Promise.all([
        billingApi.getOverview().catch(() => null),
        billingApi.listPlans().catch(() => ({ plans: [] })),
      ]);
      if (overviewRes?.overview) {
        setOverview(overviewRes.overview);
      }
      setPlans(plansRes.plans || []);
    } catch (err) {
      setError("Failed to load billing data");
      console.error("Error loading billing data:", err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadBillingData();
  }, [loadBillingData]);

  const handleUpgrade = async (planName: string) => {
    setUpgrading(true);
    try {
      if (overview) {
        await billingApi.updateSubscription(planName);
      } else {
        await billingApi.createSubscription(planName);
      }
      setShowPlansDialog(false);
      await loadBillingData();
    } catch (err) {
      console.error("Failed to upgrade:", err);
      setError("Failed to upgrade plan");
    } finally {
      setUpgrading(false);
    }
  };

  // Calculate usage percentages
  const getUsagePercent = (current: number, max: number): number => {
    if (max === -1) return 0; // Unlimited
    if (max === 0) return 100;
    return Math.min(100, (current / max) * 100);
  };

  const formatLimit = (value: number): string => {
    return value === -1 ? t("settings.billingPage.unlimited") : String(value);
  };

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="border border-border rounded-lg p-6 animate-pulse">
          <div className="h-6 bg-muted rounded w-32 mb-4"></div>
          <div className="h-8 bg-muted rounded w-48 mb-2"></div>
          <div className="h-4 bg-muted rounded w-64"></div>
        </div>
      </div>
    );
  }

  if (error && !overview) {
    return (
      <div className="space-y-6">
        <div className="border border-border rounded-lg p-6">
          <p className="text-destructive">{error}</p>
          <Button variant="outline" className="mt-4" onClick={loadBillingData}>
            {t("settings.billingPage.retry")}
          </Button>
        </div>
      </div>
    );
  }

  // If no subscription exists, show setup prompt
  if (!overview) {
    return (
      <div className="space-y-6">
        <div className="border border-border rounded-lg p-6 text-center">
          <h2 className="text-lg font-semibold mb-4">{t("settings.billingPage.noSubscription")}</h2>
          <p className="text-muted-foreground mb-6">
            {t("settings.billingPage.choosePlan")}
          </p>
          <Button onClick={() => setShowPlansDialog(true)}>{t("settings.billingPage.selectPlan")}</Button>
        </div>

        {/* Plans Dialog */}
        {showPlansDialog && (
          <PlansDialog
            plans={plans}
            currentPlan={null}
            onSelect={handleUpgrade}
            onClose={() => setShowPlansDialog(false)}
            loading={upgrading}
            t={t}
          />
        )}
      </div>
    );
  }

  const { plan, usage, status, billing_cycle, current_period_end } = overview;

  return (
    <div className="space-y-6">
      {/* Current Plan */}
      <div className="border border-border rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-4">{t("settings.billingPage.currentPlan")}</h2>
        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center gap-3">
              <h3 className="text-2xl font-bold">{plan?.display_name || plan?.name || t("settings.billingPage.plansDialog.free")}</h3>
              <span className={`text-xs px-2 py-0.5 rounded ${
                status === "active" ? "bg-green-100 text-green-800" :
                status === "past_due" ? "bg-yellow-100 text-yellow-800" :
                "bg-red-100 text-red-800"
              }`}>
                {status.charAt(0).toUpperCase() + status.slice(1)}
              </span>
            </div>
            <p className="text-muted-foreground">
              {billing_cycle === "yearly" ? t("settings.billingPage.yearly") : t("settings.billingPage.monthly")} billing
              {current_period_end && (
                <> · {t("settings.billingPage.renews")} {new Date(current_period_end).toLocaleDateString()}</>
              )}
            </p>
            {plan?.price_per_seat_monthly > 0 && (
              <p className="text-sm text-muted-foreground mt-1">
                ${plan.price_per_seat_monthly}/seat/month
              </p>
            )}
          </div>
          <Button onClick={() => setShowPlansDialog(true)}>
            {plan?.name === "free" ? t("settings.billingPage.upgrade") : t("settings.billingPage.changePlan")}
          </Button>
        </div>
      </div>

      {/* Usage */}
      <div className="border border-border rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-4">{t("settings.billingPage.usage")}</h2>
        <div className="space-y-4">
          {/* Pod Minutes */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm">{t("settings.billingPage.podMinutes")}</span>
              <span className="text-sm font-medium">
                {Math.round(usage.pod_minutes)} / {formatLimit(usage.included_pod_minutes)}
              </span>
            </div>
            <div className="w-full bg-muted rounded-full h-2">
              <div
                className={`h-2 rounded-full ${
                  getUsagePercent(usage.pod_minutes, usage.included_pod_minutes) > 90
                    ? "bg-destructive"
                    : "bg-primary"
                }`}
                style={{ width: `${getUsagePercent(usage.pod_minutes, usage.included_pod_minutes)}%` }}
              ></div>
            </div>
          </div>

          {/* Users */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm">{t("settings.billingPage.teamMembers")}</span>
              <span className="text-sm font-medium">
                {usage.users} / {formatLimit(usage.max_users)}
              </span>
            </div>
            <div className="w-full bg-muted rounded-full h-2">
              <div
                className={`h-2 rounded-full ${
                  getUsagePercent(usage.users, usage.max_users) > 90 ? "bg-destructive" : "bg-primary"
                }`}
                style={{ width: `${getUsagePercent(usage.users, usage.max_users)}%` }}
              ></div>
            </div>
          </div>

          {/* Runners */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm">Runners</span>
              <span className="text-sm font-medium">
                {usage.runners} / {formatLimit(usage.max_runners)}
              </span>
            </div>
            <div className="w-full bg-muted rounded-full h-2">
              <div
                className={`h-2 rounded-full ${
                  getUsagePercent(usage.runners, usage.max_runners) > 90 ? "bg-destructive" : "bg-primary"
                }`}
                style={{ width: `${getUsagePercent(usage.runners, usage.max_runners)}%` }}
              ></div>
            </div>
          </div>

          {/* Repositories */}
          <div>
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm">{t("settings.billingPage.repositories")}</span>
              <span className="text-sm font-medium">
                {usage.repositories} / {formatLimit(usage.max_repositories)}
              </span>
            </div>
            <div className="w-full bg-muted rounded-full h-2">
              <div
                className={`h-2 rounded-full ${
                  getUsagePercent(usage.repositories, usage.max_repositories) > 90
                    ? "bg-destructive"
                    : "bg-primary"
                }`}
                style={{ width: `${getUsagePercent(usage.repositories, usage.max_repositories)}%` }}
              ></div>
            </div>
          </div>
        </div>
      </div>

      {/* Promo Code */}
      <div className="border border-border rounded-lg p-6">
        <h2 className="text-lg font-semibold mb-2">{t("settings.billingPage.promoCode.title")}</h2>
        <p className="text-sm text-muted-foreground mb-4">
          {t("settings.billingPage.promoCode.description")}
        </p>
        <PromoCodeInput
          onRedeemSuccess={(response: RedeemPromoCodeResponse) => {
            // Reload billing data after successful redemption
            loadBillingData();
          }}
          t={(key: string) => t(`settings.billingPage.promoCode.${key}`)}
        />
      </div>

      {/* Plans Dialog */}
      {showPlansDialog && (
        <PlansDialog
          plans={plans}
          currentPlan={plan?.name || null}
          onSelect={handleUpgrade}
          onClose={() => setShowPlansDialog(false)}
          loading={upgrading}
          t={t}
        />
      )}
    </div>
  );
}

// Plans selection dialog component
function PlansDialog({
  plans,
  currentPlan,
  onSelect,
  onClose,
  loading,
  t,
}: {
  plans: SubscriptionPlan[];
  currentPlan: string | null;
  onSelect: (planName: string) => void;
  onClose: () => void;
  loading: boolean;
  t: TranslationFn;
}) {
  const formatLimit = (value: number): string => {
    return value === -1 ? t("settings.billingPage.unlimited") : String(value);
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background border border-border rounded-lg p-6 w-full max-w-4xl max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-6">
          <h3 className="text-lg font-semibold">{t("settings.billingPage.plansDialog.title")}</h3>
          <button onClick={onClose} className="text-muted-foreground hover:text-foreground">
            ✕
          </button>
        </div>

        {plans.length === 0 ? (
          <p className="text-center text-muted-foreground py-8">{t("settings.billingPage.plansDialog.noPlans")}</p>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {plans.map((plan) => {
              const isCurrent = plan.name === currentPlan;
              return (
                <div
                  key={plan.id}
                  className={`border rounded-lg p-6 ${
                    isCurrent ? "border-primary bg-primary/5" : "border-border"
                  }`}
                >
                  <div className="mb-4">
                    <h4 className="text-xl font-bold">{plan.display_name}</h4>
                    {plan.price_per_seat_monthly > 0 ? (
                      <p className="text-2xl font-bold mt-2">
                        ${plan.price_per_seat_monthly}
                        <span className="text-sm font-normal text-muted-foreground">/seat/month</span>
                      </p>
                    ) : (
                      <p className="text-2xl font-bold mt-2">{t("settings.billingPage.plansDialog.free")}</p>
                    )}
                  </div>

                  <ul className="space-y-2 mb-6 text-sm">
                    <li className="flex items-center gap-2">
                      <span className="text-green-500">✓</span>
                      {formatLimit(plan.included_pod_minutes)} {t("settings.billingPage.plansDialog.podMinutes")}
                    </li>
                    <li className="flex items-center gap-2">
                      <span className="text-green-500">✓</span>
                      {formatLimit(plan.max_users)} {t("settings.billingPage.plansDialog.teamMembers")}
                    </li>
                    <li className="flex items-center gap-2">
                      <span className="text-green-500">✓</span>
                      {formatLimit(plan.max_runners)} {t("settings.billingPage.plansDialog.runners")}
                    </li>
                    <li className="flex items-center gap-2">
                      <span className="text-green-500">✓</span>
                      {formatLimit(plan.max_repositories)} {t("settings.billingPage.plansDialog.repositories")}
                    </li>
                  </ul>

                  <Button
                    className="w-full"
                    variant={isCurrent ? "outline" : "default"}
                    disabled={isCurrent || loading}
                    onClick={() => onSelect(plan.name)}
                  >
                    {loading ? t("settings.billingPage.plansDialog.processing") : isCurrent ? t("settings.billingPage.plansDialog.currentPlan") : t("settings.billingPage.plansDialog.select")}
                  </Button>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

// ===== Runners Settings =====

function RunnersSettings({ t }: { t: TranslationFn }) {
  const {
    runners,
    loading,
    error,
    fetchRunners,
    updateRunner,
    deleteRunner,
    regenerateAuthToken,
    clearError,
  } = useRunnerStore();

  const [editingRunner, setEditingRunner] = useState<Runner | null>(null);

  useEffect(() => {
    fetchRunners();
  }, [fetchRunners]);

  return (
    <div className="space-y-6">
      {error && (
        <div className="bg-destructive/10 border border-destructive text-destructive px-4 py-3 rounded-lg flex items-center justify-between">
          <span>{error}</span>
          <button onClick={clearError} className="text-sm underline">
            {t("settings.members.dismiss")}
          </button>
        </div>
      )}

      {/* Runners List */}
      <RunnersPanel
        runners={runners}
        loading={loading}
        onEdit={setEditingRunner}
        onDelete={deleteRunner}
        onRegenerateToken={regenerateAuthToken}
        t={t}
      />

      {/* Edit Runner Dialog */}
      {editingRunner && (
        <EditRunnerDialog
          runner={editingRunner}
          onClose={() => setEditingRunner(null)}
          onSave={async (id, data) => {
            await updateRunner(id, data);
            setEditingRunner(null);
          }}
          t={t}
        />
      )}
    </div>
  );
}

// RunnersPanel Component
function RunnersPanel({
  runners,
  loading,
  onEdit,
  onDelete,
  onRegenerateToken,
  t,
}: {
  runners: Runner[];
  loading: boolean;
  onEdit: (runner: Runner) => void;
  onDelete: (id: number) => Promise<void>;
  onRegenerateToken: (id: number) => Promise<string>;
  t: TranslationFn;
}) {
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null);
  const [regeneratedToken, setRegeneratedToken] = useState<{ id: number; token: string } | null>(null);

  const handleDelete = async (id: number) => {
    try {
      await onDelete(id);
      setConfirmDelete(null);
    } catch (err) {
      console.error("Failed to delete runner:", err);
    }
  };

  const handleRegenerateToken = async (id: number) => {
    try {
      const token = await onRegenerateToken(id);
      setRegeneratedToken({ id, token });
    } catch (err) {
      console.error("Failed to regenerate token:", err);
    }
  };

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const formatLastSeen = (dateString?: string) => {
    if (!dateString) return "Never";
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffSec = Math.floor(diffMs / 1000);

    if (diffSec < 60) return t("settings.runnersSection.justNow");
    if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`;
    if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`;
    return date.toLocaleDateString();
  };

  return (
    <div className="border border-border rounded-lg p-6">
      <div className="mb-4">
        <h2 className="text-lg font-semibold">{t("settings.runnersSection.title")}</h2>
        <p className="text-sm text-muted-foreground">
          {t("settings.runnersSection.description")}
        </p>
      </div>

      {loading ? (
        <div className="text-center py-4 text-muted-foreground">{t("settings.runnersSection.loading")}</div>
      ) : runners.length === 0 ? (
        <div className="text-center py-8 text-muted-foreground">
          {t("settings.runnersSection.noRunners")}
        </div>
      ) : (
        <div className="space-y-3">
          {runners.map((runner) => {
            const statusInfo = getRunnerStatusInfo(runner.status as "online" | "offline" | "maintenance" | "busy");
            return (
              <div
                key={runner.id}
                className={`p-4 border rounded-lg ${
                  runner.is_enabled ? "border-border" : "border-border bg-muted/50"
                }`}
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <span className="font-medium">{runner.node_id}</span>
                      <span
                        className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${statusInfo?.color}`}
                      >
                        <span className={`w-1.5 h-1.5 rounded-full ${statusInfo?.dotColor}`} />
                        {statusInfo?.label}
                      </span>
                      {!runner.is_enabled && (
                        <span className="text-xs bg-yellow-100 text-yellow-800 px-2 py-0.5 rounded">
                          {t("settings.runnersSection.disabled")}
                        </span>
                      )}
                    </div>
                    {runner.description && (
                      <p className="text-sm text-muted-foreground mt-1">
                        {runner.description}
                      </p>
                    )}
                    <div className="flex items-center gap-4 text-sm text-muted-foreground mt-2">
                      <span>
                        {t("settings.runnersSection.pods")} {runner.current_pods} / {runner.max_concurrent_pods}
                      </span>
                      {runner.runner_version && <span>v{runner.runner_version}</span>}
                      <span>{t("settings.runnersSection.lastSeen")} {formatLastSeen(runner.last_heartbeat)}</span>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <Button variant="outline" size="sm" onClick={() => onEdit(runner)}>
                      {t("settings.runnersSection.edit")}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleRegenerateToken(runner.id)}
                    >
                      {t("settings.runnersSection.regenerateToken")}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="text-destructive hover:text-destructive"
                      onClick={() => setConfirmDelete(runner.id)}
                    >
                      {t("settings.runnersSection.delete")}
                    </Button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Confirm Delete Dialog */}
      {confirmDelete !== null && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-background border border-border rounded-lg p-6 w-full max-w-sm">
            <h3 className="text-lg font-semibold mb-2">{t("settings.runnersSection.deleteDialog.title")}</h3>
            <p className="text-muted-foreground mb-4">
              {t("settings.runnersSection.deleteDialog.description")}
            </p>
            <div className="flex gap-3">
              <Button variant="outline" className="flex-1" onClick={() => setConfirmDelete(null)}>
                {t("settings.runnersSection.deleteDialog.cancel")}
              </Button>
              <Button
                variant="destructive"
                className="flex-1"
                onClick={() => handleDelete(confirmDelete)}
              >
                {t("settings.runnersSection.deleteDialog.delete")}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Regenerated Token Dialog */}
      {regeneratedToken && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-background border border-border rounded-lg p-6 w-full max-w-md">
            <h3 className="text-lg font-semibold mb-4">{t("settings.runnersSection.tokenDialog.title")}</h3>
            <p className="text-sm text-muted-foreground mb-4">
              {t("settings.runnersSection.tokenDialog.description")}
            </p>
            <div className="bg-muted p-3 rounded-lg mb-4 flex items-center justify-between">
              <code className="text-sm break-all">{regeneratedToken.token}</code>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => copyToClipboard(regeneratedToken.token)}
              >
                {t("settings.runnersSection.tokenDialog.copy")}
              </Button>
            </div>
            <Button className="w-full" onClick={() => setRegeneratedToken(null)}>
              {t("settings.runnersSection.tokenDialog.done")}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

// Edit Runner Dialog Component
function EditRunnerDialog({
  runner,
  onClose,
  onSave,
  t,
}: {
  runner: Runner;
  onClose: () => void;
  onSave: (id: number, data: { description?: string; max_concurrent_pods?: number; is_enabled?: boolean }) => Promise<void>;
  t: TranslationFn;
}) {
  const [description, setDescription] = useState(runner.description || "");
  const [maxPods, setMaxPods] = useState(runner.max_concurrent_pods.toString());
  const [isEnabled, setIsEnabled] = useState(runner.is_enabled);
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    setSaving(true);
    try {
      await onSave(runner.id, {
        description: description || undefined,
        max_concurrent_pods: parseInt(maxPods, 10),
        is_enabled: isEnabled,
      });
    } catch (err) {
      console.error("Failed to save runner:", err);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background border border-border rounded-lg p-6 w-full max-w-md">
        <h3 className="text-lg font-semibold mb-4">{t("settings.runnersSection.editDialog.title")}</h3>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">{t("settings.runnersSection.editDialog.nodeIdLabel")}</label>
            <Input value={runner.node_id} disabled />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">{t("settings.runnersSection.editDialog.descriptionLabel")}</label>
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t("settings.runnersSection.editDialog.descriptionPlaceholder")}
            />
          </div>
          <div>
            <label className="block text-sm font-medium mb-2">
              {t("settings.runnersSection.editDialog.maxPodsLabel")}
            </label>
            <Input
              type="number"
              value={maxPods}
              onChange={(e) => setMaxPods(e.target.value)}
              min="1"
            />
          </div>
          <div className="flex items-center justify-between">
            <label className="text-sm font-medium">{t("settings.runnersSection.editDialog.enabledLabel")}</label>
            <label className="relative inline-flex items-center cursor-pointer">
              <input
                type="checkbox"
                className="sr-only peer"
                checked={isEnabled}
                onChange={(e) => setIsEnabled(e.target.checked)}
              />
              <div className="w-11 h-6 bg-gray-200 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary"></div>
            </label>
          </div>
        </div>
        <div className="flex gap-3 mt-6">
          <Button variant="outline" className="flex-1" onClick={onClose}>
            {t("settings.runnersSection.editDialog.cancel")}
          </Button>
          <Button className="flex-1" onClick={handleSave} disabled={saving}>
            {saving ? t("settings.runnersSection.editDialog.saving") : t("settings.runnersSection.editDialog.saveChanges")}
          </Button>
        </div>
      </div>
    </div>
  );
}

// NOTE: NotificationsSettings has been removed and moved to personal settings (/settings/notifications)
