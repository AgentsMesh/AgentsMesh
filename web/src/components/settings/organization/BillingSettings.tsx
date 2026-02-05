"use client";

import { useState, useEffect, useCallback } from "react";
import { useSearchParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { billingApi, BillingOverview, SubscriptionPlan, DeploymentInfo } from "@/lib/api";
import { ApiError } from "@/lib/api/base";
import { CheckoutFlow, CancelSubscriptionDialog, SeatManagement, BillingCycleSwitch } from "@/components/billing";
import type { BillingCycle } from "@/lib/api/billing";
import type { TranslationFn } from "./GeneralSettings";
import {
  BillingLoadingSkeleton,
  CurrentPlanCard,
  UsageCard,
  PromoCodeCard,
  PlansDialog,
} from "./billing";

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
