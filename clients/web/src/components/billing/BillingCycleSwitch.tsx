"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import type { BillingCycle } from "@/lib/api/billing-types";
import { getBillingService } from "@/lib/wasm-core";
import { getLocalizedErrorMessage } from "@/lib/api/errors";

export interface BillingCycleSwitchProps {
  currentCycle: BillingCycle;
  nextCycle?: BillingCycle;
  effectiveDate?: string;
  t: (key: string, params?: Record<string, string | number>) => string;
  onCycleChanged?: (newCycle: BillingCycle, effectiveDate: string) => void;
  onError?: (error: string) => void;
}

export function BillingCycleSwitch({
  currentCycle,
  nextCycle,
  effectiveDate,
  t,
  onCycleChanged,
  onError,
}: BillingCycleSwitchProps) {
  const [loading, setLoading] = useState(false);
  const [showConfirm, setShowConfirm] = useState(false);
  const [targetCycle, setTargetCycle] = useState<BillingCycle | null>(null);

  const handleSwitchCycle = async (newCycle: BillingCycle) => {
    setLoading(true);
    try {
      const result = JSON.parse(await getBillingService().change_cycle(
        JSON.stringify({ billing_cycle: newCycle })
      ));
      onCycleChanged?.(newCycle, result.effective_date);
      setShowConfirm(false);
      setTargetCycle(null);
    } catch (err) {
      onError?.(getLocalizedErrorMessage(err, t, t("billing.cycleSwitch.failed") || "Failed to change billing cycle"));
    } finally {
      setLoading(false);
    }
  };

  const initiateSwitch = (cycle: BillingCycle) => {
    setTargetCycle(cycle);
    setShowConfirm(true);
  };

  if (nextCycle && nextCycle !== currentCycle) {
    return (
      <div className="border border-border rounded-lg p-4">
        <h3 className="text-sm font-medium mb-2">
          {t("billing.cycleSwitch.title")}
        </h3>
        <p className="text-sm text-muted-foreground mb-4">
          {t("billing.cycleSwitch.pendingChange", {
            from: t(`billing.cycleSwitch.${currentCycle}`),
            to: t(`billing.cycleSwitch.${nextCycle}`),
            date: effectiveDate
              ? new Date(effectiveDate).toLocaleDateString()
              : "",
          })}
        </p>
        <Button
          variant="outline"
          size="sm"
          onClick={() => handleSwitchCycle(currentCycle)}
          disabled={loading}
        >
          {loading
            ? t("billing.cycleSwitch.cancelling")
            : t("billing.cycleSwitch.cancelChange")}
        </Button>
      </div>
    );
  }

  if (showConfirm && targetCycle) {
    return (
      <div className="border border-border rounded-lg p-4">
        <h3 className="text-sm font-medium mb-2">
          {t("billing.cycleSwitch.confirmTitle")}
        </h3>
        <p className="text-sm text-muted-foreground mb-4">
          {t("billing.cycleSwitch.confirmDescription", {
            from: t(`billing.cycleSwitch.${currentCycle}`),
            to: t(`billing.cycleSwitch.${targetCycle}`),
          })}
        </p>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              setShowConfirm(false);
              setTargetCycle(null);
            }}
            disabled={loading}
          >
            {t("billing.cycleSwitch.cancel")}
          </Button>
          <Button
            size="sm"
            onClick={() => handleSwitchCycle(targetCycle)}
            disabled={loading}
          >
            {loading
              ? t("billing.cycleSwitch.switching")
              : t("billing.cycleSwitch.confirm")}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="border border-border rounded-lg p-4">
      <h3 className="text-sm font-medium mb-2">
        {t("billing.cycleSwitch.title")}
      </h3>
      <p className="text-sm text-muted-foreground mb-4">
        {t("billing.cycleSwitch.description")}
      </p>
      <div className="flex gap-2">
        <Button
          variant={currentCycle === "monthly" ? "default" : "outline"}
          size="sm"
          disabled={currentCycle === "monthly"}
          onClick={() => initiateSwitch("monthly")}
        >
          {t("billing.cycleSwitch.monthly")}
        </Button>
        <Button
          variant={currentCycle === "yearly" ? "default" : "outline"}
          size="sm"
          disabled={currentCycle === "yearly"}
          onClick={() => initiateSwitch("yearly")}
        >
          {t("billing.cycleSwitch.yearly")}
          {currentCycle === "monthly" && (
            <span className="ml-2 text-xs bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400 px-1.5 py-0.5 rounded">
              {t("billing.cycleSwitch.saveNote")}
            </span>
          )}
        </Button>
      </div>
    </div>
  );
}
