"use client";

import { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { billingApi, BillingOverview, SubscriptionPlan, RedeemPromoCodeResponse } from "@/lib/api/client";
import { PromoCodeInput } from "@/components/promo-code/PromoCodeInput";
import type { TranslationFn } from "./GeneralSettings";

interface BillingSettingsProps {
  t: TranslationFn;
}

export function BillingSettings({ t }: BillingSettingsProps) {
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

  const getUsagePercent = (current: number, max: number): number => {
    if (max === -1) return 0;
    if (max === 0) return 100;
    return Math.min(100, (current / max) * 100);
  };

  const formatLimit = (value: number): string => {
    return value === -1 ? t("settings.billingPage.unlimited") : String(value);
  };

  if (loading) {
    return <BillingLoadingSkeleton />;
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
      <CurrentPlanCard
        plan={plan}
        status={status}
        billing_cycle={billing_cycle}
        current_period_end={current_period_end}
        onChangePlan={() => setShowPlansDialog(true)}
        t={t}
      />

      {/* Usage */}
      <UsageCard usage={usage} getUsagePercent={getUsagePercent} formatLimit={formatLimit} t={t} />

      {/* Promo Code */}
      <PromoCodeCard onRedeemSuccess={() => loadBillingData()} t={t} />

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

function BillingLoadingSkeleton() {
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

function CurrentPlanCard({
  plan,
  status,
  billing_cycle,
  current_period_end,
  onChangePlan,
  t,
}: {
  plan: BillingOverview["plan"];
  status: string;
  billing_cycle: string;
  current_period_end?: string;
  onChangePlan: () => void;
  t: TranslationFn;
}) {
  return (
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
        <Button onClick={onChangePlan}>
          {plan?.name === "free" ? t("settings.billingPage.upgrade") : t("settings.billingPage.changePlan")}
        </Button>
      </div>
    </div>
  );
}

function UsageCard({
  usage,
  getUsagePercent,
  formatLimit,
  t,
}: {
  usage: BillingOverview["usage"];
  getUsagePercent: (current: number, max: number) => number;
  formatLimit: (value: number) => string;
  t: TranslationFn;
}) {
  const usageItems = [
    { label: t("settings.billingPage.podMinutes"), current: Math.round(usage.pod_minutes), max: usage.included_pod_minutes },
    { label: t("settings.billingPage.teamMembers"), current: usage.users, max: usage.max_users },
    { label: "Runners", current: usage.runners, max: usage.max_runners },
    { label: t("settings.billingPage.repositories"), current: usage.repositories, max: usage.max_repositories },
  ];

  return (
    <div className="border border-border rounded-lg p-6">
      <h2 className="text-lg font-semibold mb-4">{t("settings.billingPage.usage")}</h2>
      <div className="space-y-4">
        {usageItems.map((item, index) => (
          <div key={index}>
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm">{item.label}</span>
              <span className="text-sm font-medium">
                {item.current} / {formatLimit(item.max)}
              </span>
            </div>
            <div className="w-full bg-muted rounded-full h-2">
              <div
                className={`h-2 rounded-full ${
                  getUsagePercent(item.current, item.max) > 90 ? "bg-destructive" : "bg-primary"
                }`}
                style={{ width: `${getUsagePercent(item.current, item.max)}%` }}
              ></div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function PromoCodeCard({
  onRedeemSuccess,
  t,
}: {
  onRedeemSuccess: () => void;
  t: TranslationFn;
}) {
  return (
    <div className="border border-border rounded-lg p-6">
      <h2 className="text-lg font-semibold mb-2">{t("settings.billingPage.promoCode.title")}</h2>
      <p className="text-sm text-muted-foreground mb-4">
        {t("settings.billingPage.promoCode.description")}
      </p>
      <PromoCodeInput
        onRedeemSuccess={(response: RedeemPromoCodeResponse) => {
          onRedeemSuccess();
        }}
        t={(key: string) => t(`settings.billingPage.promoCode.${key}`)}
      />
    </div>
  );
}

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
