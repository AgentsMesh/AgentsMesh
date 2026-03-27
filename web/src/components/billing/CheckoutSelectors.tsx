"use client";

import type { SubscriptionPlan, BillingCycle } from "@/lib/api/billing";

interface BillingCycleSelectorProps {
  plan: SubscriptionPlan;
  selectedCycle: BillingCycle;
  onCycleChange: (cycle: BillingCycle) => void;
  annualSavings: number;
  t: (key: string, params?: Record<string, string | number>) => string;
}

export function BillingCycleSelector({ plan, selectedCycle, onCycleChange, annualSavings, t }: BillingCycleSelectorProps) {
  return (
    <div className="space-y-3">
      <span className="text-sm text-muted-foreground">{t("billing.checkout.billingCycle")}</span>
      <div className="grid grid-cols-2 gap-3">
        <button type="button"
          className={`p-4 border rounded-lg text-left transition-colors ${
            selectedCycle === "monthly" ? "border-primary bg-primary/5" : "border-border hover:border-muted-foreground"}`}
          onClick={() => onCycleChange("monthly")}>
          <div className="font-medium">{t("billing.checkout.monthly")}</div>
          <div className="text-sm text-muted-foreground">${plan.price_per_seat_monthly}/seat/month</div>
        </button>
        <button type="button"
          className={`p-4 border rounded-lg text-left transition-colors ${
            selectedCycle === "yearly" ? "border-primary bg-primary/5" : "border-border hover:border-muted-foreground"}`}
          onClick={() => onCycleChange("yearly")}>
          <div className="font-medium flex items-center gap-2">
            {t("billing.checkout.yearly")}
            {annualSavings > 0 && (
              <span className="text-xs bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400 px-2 py-0.5 rounded">
                {t("billing.checkout.save", { amount: annualSavings.toFixed(0) })}
              </span>
            )}
          </div>
          <div className="text-sm text-muted-foreground">${plan.price_per_seat_yearly}/seat/year</div>
        </button>
      </div>
    </div>
  );
}

interface SeatSelectorProps {
  seats: number;
  onSeatsChange: (n: number) => void;
  pricePerSeat: number;
  isSubscription: boolean;
  t: (key: string, params?: Record<string, string | number>) => string;
}

export function SeatSelector({ seats, onSeatsChange, pricePerSeat, isSubscription, t }: SeatSelectorProps) {
  if (!isSubscription) {
    return (
      <div className="flex justify-between">
        <span>{t("billing.checkout.seats")}</span>
        <span className="font-medium">{seats}</span>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <span>{t("billing.checkout.seats")}</span>
        <div className="flex items-center gap-2">
          <button type="button"
            className="w-8 h-8 flex items-center justify-center border border-border rounded hover:bg-muted disabled:opacity-50"
            onClick={() => onSeatsChange(Math.max(1, seats - 1))} disabled={seats <= 1}>-</button>
          <span className="font-medium w-8 text-center">{seats}</span>
          <button type="button"
            className="w-8 h-8 flex items-center justify-center border border-border rounded hover:bg-muted"
            onClick={() => onSeatsChange(seats + 1)}>+</button>
        </div>
      </div>
      {seats > 1 && (
        <div className="text-xs text-muted-foreground">
          ${pricePerSeat}/{t("billing.checkout.perSeat")}
        </div>
      )}
    </div>
  );
}
