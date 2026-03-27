"use client";

import { Button } from "@/components/ui/button";
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
import { useBillingSettings } from "./billing/useBillingSettings";

interface BillingSettingsProps {
  t: TranslationFn;
}

export function BillingSettings({ t }: BillingSettingsProps) {
  const state = useBillingSettings(t);
  const currentUrl = typeof window !== "undefined" ? window.location.href.split("?")[0] : "";

  const getUsagePercent = (current: number, max: number): number => {
    if (max === -1) return 0;
    if (max === 0) return 100;
    return Math.min(100, (current / max) * 100);
  };

  const formatLimit = (value: number): string => {
    return value === -1 ? t("settings.billingPage.unlimited") : String(value);
  };

  if (state.loading) return <BillingLoadingSkeleton />;

  if (state.error && !state.overview) {
    return (
      <div className="space-y-6">
        <div className="border border-border rounded-lg p-6">
          <p className="text-destructive">{state.error}</p>
          <Button variant="outline" className="mt-4" onClick={state.loadBillingData}>
            {t("settings.billingPage.retry")}
          </Button>
        </div>
      </div>
    );
  }

  if (state.showCheckout && state.selectedPlan) {
    return (
      <div className="space-y-6">
        <div className="border border-border rounded-lg p-6">
          <CheckoutFlow
            plan={state.selectedPlan}
            orderType={state.overview ? "plan_upgrade" : "subscription"}
            currentUrl={currentUrl}
            deploymentInfo={state.deploymentInfo || undefined}
            t={(key, params) => t(`settings.${key}`, params)}
            onCheckoutCreated={state.handleCheckoutComplete}
            onError={(err) => state.setError(err)}
            onCancel={() => { state.setShowCheckout(false); state.setSelectedPlan(null); }}
          />
        </div>
      </div>
    );
  }

  if (!state.overview) {
    return (
      <div className="space-y-6">
        <div className="border border-border rounded-lg p-6 text-center">
          <h2 className="text-lg font-semibold mb-4">{t("settings.billingPage.noSubscription")}</h2>
          <p className="text-muted-foreground mb-6">{t("settings.billingPage.choosePlan")}</p>
          <Button onClick={() => state.setShowPlansDialog(true)}>{t("settings.billingPage.selectPlan")}</Button>
        </div>
        {state.showPlansDialog && (
          <PlansDialog plans={state.plans} currentPlan={null} onSelect={state.handleSelectPlan}
            onClose={() => state.setShowPlansDialog(false)} loading={state.upgrading} t={t} />
        )}
      </div>
    );
  }

  const { plan, usage, status, billing_cycle, current_period_end } = state.overview;

  return (
    <div className="space-y-6">
      {state.error && (
        <div className="p-4 rounded-lg bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400 border border-red-200 dark:border-red-800">
          {state.error}
          <button className="ml-4 text-sm underline" onClick={() => state.setError(null)}>
            {t("settings.billingPage.dismiss")}
          </button>
        </div>
      )}
      {state.paymentMessage && (
        <div className={`p-4 rounded-lg ${
          state.paymentMessage.type === "success"
            ? "bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400 border border-green-200 dark:border-green-800"
            : "bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400 border border-red-200 dark:border-red-800"
        }`}>
          {state.paymentMessage.text}
          <button className="ml-4 text-sm underline" onClick={() => state.setPaymentMessage(null)}>
            {t("settings.billingPage.dismiss")}
          </button>
        </div>
      )}
      <CurrentPlanCard plan={plan} status={status} billing_cycle={billing_cycle}
        current_period_end={current_period_end} cancelAtPeriodEnd={state.overview.cancel_at_period_end}
        onChangePlan={() => state.setShowPlansDialog(true)} onCancelPlan={() => state.setShowCancelDialog(true)}
        onReactivate={state.handleReactivateSubscription} reactivating={state.reactivating} t={t} />
      {status === "active" && plan?.price_per_seat_monthly > 0 && (
        <BillingCycleSwitch currentCycle={billing_cycle as BillingCycle} nextCycle={undefined}
          t={(key, params) => t(`settings.${key}`, params)}
          onCycleChanged={() => { state.loadBillingData(); state.setPaymentMessage({ type: "success", text: t("settings.billing.cycleSwitch.success") }); }}
          onError={(err) => state.setError(err)} />
      )}
      <SeatManagement t={(key, params) => t(`settings.${key}`, params)} currentUrl={currentUrl} />
      <UsageCard usage={usage} getUsagePercent={getUsagePercent} formatLimit={formatLimit} t={t} />
      <PromoCodeCard onRedeemSuccess={() => state.loadBillingData()} t={t} />
      {state.showPlansDialog && (
        <PlansDialog plans={state.plans} currentPlan={plan?.name || null} onSelect={state.handleSelectPlan}
          onClose={() => state.setShowPlansDialog(false)} loading={state.upgrading} t={t} />
      )}
      {state.showCancelDialog && current_period_end && (
        <CancelSubscriptionDialog open={state.showCancelDialog} onOpenChange={state.setShowCancelDialog}
          periodEnd={current_period_end} t={(key, params) => t(`settings.${key}`, params)}
          onCancelled={state.handleCancelSubscription} />
      )}
    </div>
  );
}
