"use client";

import { useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { billingApi, SubscriptionPlan, BillingCycle, OrderType, CheckoutResponse, DeploymentInfo } from "@/lib/api/billing";
import { useLemonSqueezy } from "@/hooks/useLemonSqueezy";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { QRCodeCheckout } from "./QRCodeCheckout";
import { BillingCycleSelector, SeatSelector } from "./CheckoutSelectors";

export interface CheckoutFlowProps {
  plan?: SubscriptionPlan;
  orderType: OrderType;
  billingCycle?: BillingCycle;
  seats?: number;
  currentUrl: string;
  deploymentInfo?: DeploymentInfo;
  t: (key: string, params?: Record<string, string | number>) => string;
  onCheckoutCreated?: (response: CheckoutResponse) => void;
  onError?: (error: string) => void;
  onCancel?: () => void;
}

export function CheckoutFlow({
  plan, orderType, billingCycle = "monthly", seats = 1, currentUrl,
  deploymentInfo, t, onCheckoutCreated, onError, onCancel,
}: CheckoutFlowProps) {
  const router = useRouter();
  const [loading, setLoading] = useState(false);
  const [selectedCycle, setSelectedCycle] = useState<BillingCycle>(billingCycle);
  const [selectedSeats, setSelectedSeats] = useState(seats);
  const [checkoutResponse, setCheckoutResponse] = useState<CheckoutResponse | null>(null);

  const { openCheckout: openLemonCheckout } = useLemonSqueezy({
    onCheckoutSuccess: () => { router.refresh(); window.location.href = `${currentUrl}?payment=success`; },
    onCheckoutClose: () => { setLoading(false); },
  });

  const getPrice = useCallback(() => {
    if (!plan) return 0;
    return (selectedCycle === "yearly" ? plan.price_per_seat_yearly : plan.price_per_seat_monthly) * selectedSeats;
  }, [plan, selectedCycle, selectedSeats]);

  const getAnnualSavings = useCallback(() => {
    if (!plan) return 0;
    return plan.price_per_seat_monthly * 12 * selectedSeats - plan.price_per_seat_yearly * selectedSeats;
  }, [plan, selectedSeats]);

  const handleCheckout = async () => {
    setLoading(true);
    try {
      const response = await billingApi.createCheckout({
        order_type: orderType, plan_name: plan?.name, billing_cycle: selectedCycle,
        seats: orderType === "seat_purchase" || orderType === "subscription" ? selectedSeats : undefined,
        success_url: `${currentUrl}?payment=success`, cancel_url: `${currentUrl}?payment=cancelled`,
      });
      setCheckoutResponse(response);
      onCheckoutCreated?.(response);
      if (response.session_url) {
        if (response.provider === "lemonsqueezy") openLemonCheckout(response.session_url);
        else window.location.href = response.session_url;
      }
    } catch (err) {
      onError?.(getLocalizedErrorMessage(err, t, t("billing.checkout.failed") || "Checkout failed"));
      setLoading(false);
    }
  };

  if (checkoutResponse?.qr_code_url) {
    return <QRCodeCheckout response={checkoutResponse} t={t}
      onCancel={() => { setCheckoutResponse(null); onCancel?.(); }} />;
  }

  return (
    <div className="space-y-6">
      <div className="border border-border rounded-lg p-6">
        <h3 className="text-lg font-semibold mb-4">{t("billing.checkout.orderSummary")}</h3>
        {plan && (
          <div className="space-y-4">
            <div className="flex justify-between">
              <span>{t("billing.checkout.plan")}</span>
              <span className="font-medium">{plan.display_name}</span>
            </div>
            {(orderType === "subscription" || orderType === "plan_upgrade") && (
              <BillingCycleSelector plan={plan} selectedCycle={selectedCycle}
                onCycleChange={setSelectedCycle} annualSavings={getAnnualSavings()} t={t} />
            )}
            <SeatSelector seats={selectedSeats} onSeatsChange={setSelectedSeats}
              pricePerSeat={selectedCycle === "yearly" ? plan.price_per_seat_yearly : plan.price_per_seat_monthly}
              isSubscription={orderType === "subscription"} t={t} />
            <div className="border-t border-border pt-4 flex justify-between text-lg font-semibold">
              <span>{t("billing.checkout.total")}</span>
              <span>${getPrice().toFixed(2)}{selectedCycle === "yearly" ? "/year" : "/month"}</span>
            </div>
          </div>
        )}
        {orderType === "seat_purchase" && (
          <div className="space-y-4">
            <div className="flex justify-between">
              <span>{t("billing.checkout.additionalSeats")}</span>
              <span className="font-medium">{selectedSeats}</span>
            </div>
            <div className="text-sm text-muted-foreground">{t("billing.checkout.seatPurchaseNote")}</div>
          </div>
        )}
      </div>
      {deploymentInfo && (
        <div className="text-sm text-muted-foreground">
          {deploymentInfo.deployment_type === "global" && <span>{t("billing.checkout.globalPayment")}</span>}
          {deploymentInfo.deployment_type === "cn" && <span>{t("billing.checkout.cnPayment")}</span>}
        </div>
      )}
      <div className="flex justify-end gap-3">
        <Button variant="outline" onClick={onCancel} disabled={loading}>{t("billing.checkout.cancel")}</Button>
        <Button onClick={handleCheckout} loading={loading}>
          {loading ? t("billing.checkout.processing") : t("billing.checkout.proceedToPayment")}
        </Button>
      </div>
    </div>
  );
}
