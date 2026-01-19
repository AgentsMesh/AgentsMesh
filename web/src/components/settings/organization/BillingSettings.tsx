"use client";

import { useState, useEffect, useCallback } from "react";
import { useSearchParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { billingApi, BillingOverview, SubscriptionPlan, DeploymentInfo } from "@/lib/api";
import { ApiError } from "@/lib/api/base";
import { PromoCodeInput } from "@/components/promo-code/PromoCodeInput";
import { CheckoutFlow, CancelSubscriptionDialog, SeatManagement, BillingCycleSwitch } from "@/components/billing";
import type { BillingCycle } from "@/lib/api/billing";
import type { TranslationFn } from "./GeneralSettings";

interface BillingSettingsProps {
  t: TranslationFn;
}

export function BillingSettings({ t }: BillingSettingsProps) {
  const searchParams = useSearchParams();
  const [loading, setLoading] = useState(true);
  const [overview, setOverview] = useState<BillingOverview | null>(null);
  const [plans, setPlans] = useState<SubscriptionPlan[]>([]);
  const [deploymentInfo, setDeploymentInfo] = useState<DeploymentInfo | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [showPlansDialog, setShowPlansDialog] = useState(false);
  const [showCheckout, setShowCheckout] = useState(false);
  const [showCancelDialog, setShowCancelDialog] = useState(false);
  const [selectedPlan, setSelectedPlan] = useState<SubscriptionPlan | null>(null);
  const [upgrading, setUpgrading] = useState(false);
  const [reactivating, setReactivating] = useState(false);
  const [paymentMessage, setPaymentMessage] = useState<{ type: "success" | "error"; text: string } | null>(null);

  // Check for payment callback
  useEffect(() => {
    const payment = searchParams.get("payment");
    if (payment === "success") {
      setPaymentMessage({ type: "success", text: t("settings.billingPage.paymentSuccess") });
      // Clear the query param
      window.history.replaceState({}, "", window.location.pathname);
    } else if (payment === "cancelled") {
      setPaymentMessage({ type: "error", text: t("settings.billingPage.paymentCancelled") });
      window.history.replaceState({}, "", window.location.pathname);
    }
  }, [searchParams, t]);

  const loadBillingData = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [overviewRes, plansRes, deploymentRes] = await Promise.all([
        billingApi.getOverview().catch(() => null),
        billingApi.listPlans().catch(() => ({ plans: [] })),
        billingApi.getDeploymentInfo().catch(() => null),
      ]);
      if (overviewRes?.overview) {
        setOverview(overviewRes.overview);
      }
      setPlans(plansRes.plans || []);
      if (deploymentRes) {
        setDeploymentInfo(deploymentRes);
      }
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

  const handleSelectPlan = (planName: string) => {
    const plan = plans.find((p) => p.name === planName);
    if (plan) {
      setSelectedPlan(plan);
      setShowPlansDialog(false);

      // For free plan, no payment needed
      if (plan.price_per_seat_monthly === 0) {
        handleFreePlanSelect(planName);
      } else {
        setShowCheckout(true);
      }
    }
  };

  const handleFreePlanSelect = async (planName: string) => {
    setUpgrading(true);
    setError(null);
    try {
      if (overview) {
        await billingApi.updateSubscription(planName);
      } else {
        await billingApi.createSubscription(planName);
      }
      await loadBillingData();
    } catch (err: unknown) {
      console.error("Failed to select plan:", err);
      // Extract error message from API response
      let errorMessage = "Failed to select plan";
      if (err instanceof ApiError && err.data) {
        // ApiError stores the response body in data property
        errorMessage = (err.data as { error?: string })?.error || err.message;
      } else if (err instanceof Error) {
        errorMessage = err.message;
      }
      setError(errorMessage);
    } finally {
      setUpgrading(false);
    }
  };

  const handleCheckoutComplete = () => {
    setShowCheckout(false);
    setSelectedPlan(null);
    // Billing data will be refreshed after redirect from payment provider
  };

  const handleCancelSubscription = () => {
    setShowCancelDialog(false);
    loadBillingData();
  };

  const handleReactivateSubscription = async () => {
    setReactivating(true);
    try {
      await billingApi.reactivateSubscription();
      await loadBillingData();
      setPaymentMessage({ type: "success", text: t("settings.billingPage.reactivateSuccess") });
    } catch (err) {
      console.error("Failed to reactivate subscription:", err);
      setError("Failed to reactivate subscription");
    } finally {
      setReactivating(false);
    }
  };

  // Get current URL for payment callbacks
  const currentUrl = typeof window !== "undefined" ? window.location.href.split("?")[0] : "";

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

  // Show checkout flow
  if (showCheckout && selectedPlan) {
    return (
      <div className="space-y-6">
        <div className="border border-border rounded-lg p-6">
          <CheckoutFlow
            plan={selectedPlan}
            orderType={overview ? "plan_upgrade" : "subscription"}
            currentUrl={currentUrl}
            deploymentInfo={deploymentInfo || undefined}
            t={(key, params) => t(`settings.${key}`, params)}
            onCheckoutCreated={handleCheckoutComplete}
            onError={(err) => setError(err)}
            onCancel={() => {
              setShowCheckout(false);
              setSelectedPlan(null);
            }}
          />
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
            onSelect={handleSelectPlan}
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
      {/* Error Message */}
      {error && (
        <div className="p-4 rounded-lg bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400 border border-red-200 dark:border-red-800">
          {error}
          <button
            className="ml-4 text-sm underline"
            onClick={() => setError(null)}
          >
            {t("settings.billingPage.dismiss")}
          </button>
        </div>
      )}

      {/* Payment Success/Error Message */}
      {paymentMessage && (
        <div
          className={`p-4 rounded-lg ${
            paymentMessage.type === "success"
              ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400 border border-green-200 dark:border-green-800"
              : "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400 border border-red-200 dark:border-red-800"
          }`}
        >
          {paymentMessage.text}
          <button
            className="ml-4 text-sm underline"
            onClick={() => setPaymentMessage(null)}
          >
            {t("settings.billingPage.dismiss")}
          </button>
        </div>
      )}

      {/* Current Plan */}
      <CurrentPlanCard
        plan={plan}
        status={status}
        billing_cycle={billing_cycle}
        current_period_end={current_period_end}
        cancelAtPeriodEnd={overview.cancel_at_period_end}
        onChangePlan={() => setShowPlansDialog(true)}
        onCancelPlan={() => setShowCancelDialog(true)}
        onReactivate={handleReactivateSubscription}
        reactivating={reactivating}
        t={t}
      />

      {/* Billing Cycle Switch - only show for active paid plans */}
      {status === "active" && plan?.price_per_seat_monthly > 0 && (
        <BillingCycleSwitch
          currentCycle={billing_cycle as BillingCycle}
          nextCycle={overview.downgrade_to_plan ? undefined : undefined}
          t={(key, params) => t(`settings.${key}`, params)}
          onCycleChanged={() => {
            loadBillingData();
            setPaymentMessage({ type: "success", text: t("settings.billing.cycleSwitch.success") });
          }}
          onError={(err) => setError(err)}
        />
      )}

      {/* Seat Management */}
      <SeatManagement t={(key, params) => t(`settings.${key}`, params)} currentUrl={currentUrl} />

      {/* Usage */}
      <UsageCard usage={usage} getUsagePercent={getUsagePercent} formatLimit={formatLimit} t={t} />

      {/* Promo Code */}
      <PromoCodeCard onRedeemSuccess={() => loadBillingData()} t={t} />

      {/* Plans Dialog */}
      {showPlansDialog && (
        <PlansDialog
          plans={plans}
          currentPlan={plan?.name || null}
          onSelect={handleSelectPlan}
          onClose={() => setShowPlansDialog(false)}
          loading={upgrading}
          t={t}
        />
      )}

      {/* Cancel Subscription Dialog */}
      {showCancelDialog && current_period_end && (
        <CancelSubscriptionDialog
          open={showCancelDialog}
          onOpenChange={setShowCancelDialog}
          periodEnd={current_period_end}
          t={(key, params) => t(`settings.${key}`, params)}
          onCancelled={handleCancelSubscription}
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
  cancelAtPeriodEnd,
  onChangePlan,
  onCancelPlan,
  onReactivate,
  reactivating,
  t,
}: {
  plan: BillingOverview["plan"];
  status: string;
  billing_cycle: string;
  current_period_end?: string;
  cancelAtPeriodEnd?: boolean;
  onChangePlan: () => void;
  onCancelPlan: () => void;
  onReactivate: () => void;
  reactivating?: boolean;
  t: TranslationFn;
}) {
  const isPaidPlan = plan?.price_per_seat_monthly && plan.price_per_seat_monthly > 0;
  const isFrozen = status === "frozen";
  const isCanceled = status === "canceled";
  const isInactive = isFrozen || isCanceled;

  return (
    <div className="border border-border rounded-lg p-6">
      <h2 className="text-lg font-semibold mb-4">{t("settings.billingPage.currentPlan")}</h2>
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h3 className="text-2xl font-bold">{plan?.display_name || plan?.name || t("settings.billingPage.plansDialog.free")}</h3>
            <span className={`text-xs px-2 py-0.5 rounded ${
              status === "active" ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400" :
              status === "past_due" ? "bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400" :
              status === "frozen" ? "bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-400" :
              status === "canceled" ? "bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-400" :
              "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400"
            }`}>
              {status === "frozen" ? t("settings.billingPage.frozen") :
               status === "canceled" ? t("settings.billingPage.canceled") :
               status.charAt(0).toUpperCase() + status.slice(1)}
            </span>
            {cancelAtPeriodEnd && !isInactive && (
              <span className="text-xs px-2 py-0.5 rounded bg-orange-100 text-orange-800 dark:bg-orange-900/30 dark:text-orange-400">
                {t("settings.billingPage.cancellingAtPeriodEnd")}
              </span>
            )}
          </div>
          <p className="text-muted-foreground">
            {billing_cycle === "yearly" ? t("settings.billingPage.yearly") : t("settings.billingPage.monthly")} billing
            {current_period_end && !isInactive && (
              <> · {cancelAtPeriodEnd ? t("settings.billingPage.endsOn") : t("settings.billingPage.renews")} {new Date(current_period_end).toLocaleDateString()}</>
            )}
            {isFrozen && current_period_end && (
              <> · {t("settings.billingPage.expiredOn")} {new Date(current_period_end).toLocaleDateString()}</>
            )}
            {isCanceled && current_period_end && (
              <> · {t("settings.billingPage.canceledOn")} {new Date(current_period_end).toLocaleDateString()}</>
            )}
          </p>
          {isPaidPlan && (
            <p className="text-sm text-muted-foreground mt-1">
              ${plan.price_per_seat_monthly}/seat/month
            </p>
          )}
          {isFrozen && (
            <p className="text-sm text-orange-600 dark:text-orange-400 mt-2">
              {t("settings.billingPage.frozenMessage")}
            </p>
          )}
          {isCanceled && (
            <p className="text-sm text-gray-600 dark:text-gray-400 mt-2">
              {t("settings.billingPage.canceledMessage")}
            </p>
          )}
        </div>
        <div className="flex items-center gap-2">
          {/* Frozen or Canceled state: show resubscribe button */}
          {isInactive && isPaidPlan && (
            <Button variant="default" onClick={onChangePlan}>
              {t("settings.billingPage.resubscribe")}
            </Button>
          )}
          {/* Cancel pending: show reactivate button */}
          {!isInactive && isPaidPlan && cancelAtPeriodEnd && (
            <Button variant="default" onClick={onReactivate} disabled={reactivating}>
              {reactivating ? t("settings.billingPage.reactivating") : t("settings.billingPage.reactivate")}
            </Button>
          )}
          {/* Active state: show cancel button */}
          {!isInactive && isPaidPlan && !cancelAtPeriodEnd && (
            <Button variant="outline" onClick={onCancelPlan}>
              {t("settings.billingPage.cancelPlan")}
            </Button>
          )}
          {/* Active state: show change plan button */}
          {!isInactive && (
            <Button onClick={onChangePlan}>
              {plan?.name === "free" ? t("settings.billingPage.upgrade") : t("settings.billingPage.changePlan")}
            </Button>
          )}
        </div>
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
        onRedeemSuccess={() => {
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
                      <span className="text-green-500 dark:text-green-400">✓</span>
                      {formatLimit(plan.included_pod_minutes)} {t("settings.billingPage.plansDialog.podMinutes")}
                    </li>
                    <li className="flex items-center gap-2">
                      <span className="text-green-500 dark:text-green-400">✓</span>
                      {formatLimit(plan.max_users)} {t("settings.billingPage.plansDialog.teamMembers")}
                    </li>
                    <li className="flex items-center gap-2">
                      <span className="text-green-500 dark:text-green-400">✓</span>
                      {formatLimit(plan.max_runners)} {t("settings.billingPage.plansDialog.runners")}
                    </li>
                    <li className="flex items-center gap-2">
                      <span className="text-green-500 dark:text-green-400">✓</span>
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
