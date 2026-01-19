"use client";

import { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import {
  billingApi,
  SubscriptionPlan,
  BillingCycle,
  OrderType,
  CheckoutResponse,
  DeploymentInfo,
} from "@/lib/api/billing";

export interface CheckoutFlowProps {
  // Plan to checkout (for subscription/upgrade)
  plan?: SubscriptionPlan;
  // Order type
  orderType: OrderType;
  // Billing cycle (for subscription)
  billingCycle?: BillingCycle;
  // Number of seats (for seat purchase)
  seats?: number;
  // Current URL for success/cancel redirect
  currentUrl: string;
  // Deployment info for showing available providers
  deploymentInfo?: DeploymentInfo;
  // Translation function
  t: (key: string, params?: Record<string, string | number>) => string;
  // Callback when checkout is created
  onCheckoutCreated?: (response: CheckoutResponse) => void;
  // Callback for errors
  onError?: (error: string) => void;
  // Callback when user cancels
  onCancel?: () => void;
}

export function CheckoutFlow({
  plan,
  orderType,
  billingCycle = "monthly",
  seats = 1,
  currentUrl,
  deploymentInfo,
  t,
  onCheckoutCreated,
  onError,
  onCancel,
}: CheckoutFlowProps) {
  const [loading, setLoading] = useState(false);
  const [selectedCycle, setSelectedCycle] = useState<BillingCycle>(billingCycle);
  const [checkoutResponse, setCheckoutResponse] = useState<CheckoutResponse | null>(null);

  // Calculate price based on billing cycle
  const getPrice = useCallback(() => {
    if (!plan) return 0;
    const basePrice =
      selectedCycle === "yearly"
        ? plan.price_per_seat_yearly
        : plan.price_per_seat_monthly;
    return basePrice * seats;
  }, [plan, selectedCycle, seats]);

  // Calculate annual savings
  const getAnnualSavings = useCallback(() => {
    if (!plan) return 0;
    const monthlyTotal = plan.price_per_seat_monthly * 12 * seats;
    const yearlyTotal = plan.price_per_seat_yearly * seats;
    return monthlyTotal - yearlyTotal;
  }, [plan, seats]);

  const handleCheckout = async () => {
    setLoading(true);
    try {
      const successUrl = `${currentUrl}?payment=success`;
      const cancelUrl = `${currentUrl}?payment=cancelled`;

      const response = await billingApi.createCheckout({
        order_type: orderType,
        plan_name: plan?.name,
        billing_cycle: selectedCycle,
        seats: orderType === "seat_purchase" ? seats : undefined,
        success_url: successUrl,
        cancel_url: cancelUrl,
      });

      setCheckoutResponse(response);
      onCheckoutCreated?.(response);

      // For Stripe, redirect to checkout page
      if (response.session_url) {
        window.location.href = response.session_url;
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : "Checkout failed";
      onError?.(errorMessage);
    } finally {
      setLoading(false);
    }
  };

  // If we have a QR code response (for Alipay/WeChat), show it
  if (checkoutResponse?.qr_code_url) {
    return (
      <QRCodeCheckout
        response={checkoutResponse}
        t={t}
        onCancel={() => {
          setCheckoutResponse(null);
          onCancel?.();
        }}
      />
    );
  }

  return (
    <div className="space-y-6">
      {/* Order Summary */}
      <div className="border border-border rounded-lg p-6">
        <h3 className="text-lg font-semibold mb-4">
          {t("billing.checkout.orderSummary")}
        </h3>

        {plan && (
          <div className="space-y-4">
            <div className="flex justify-between">
              <span>{t("billing.checkout.plan")}</span>
              <span className="font-medium">{plan.display_name}</span>
            </div>

            {/* Billing Cycle Selector */}
            {(orderType === "subscription" || orderType === "plan_upgrade") && (
              <div className="space-y-3">
                <span className="text-sm text-muted-foreground">
                  {t("billing.checkout.billingCycle")}
                </span>
                <div className="grid grid-cols-2 gap-3">
                  <button
                    type="button"
                    className={`p-4 border rounded-lg text-left transition-colors ${
                      selectedCycle === "monthly"
                        ? "border-primary bg-primary/5"
                        : "border-border hover:border-muted-foreground"
                    }`}
                    onClick={() => setSelectedCycle("monthly")}
                  >
                    <div className="font-medium">
                      {t("billing.checkout.monthly")}
                    </div>
                    <div className="text-sm text-muted-foreground">
                      ${plan.price_per_seat_monthly}/seat/month
                    </div>
                  </button>
                  <button
                    type="button"
                    className={`p-4 border rounded-lg text-left transition-colors ${
                      selectedCycle === "yearly"
                        ? "border-primary bg-primary/5"
                        : "border-border hover:border-muted-foreground"
                    }`}
                    onClick={() => setSelectedCycle("yearly")}
                  >
                    <div className="font-medium flex items-center gap-2">
                      {t("billing.checkout.yearly")}
                      {getAnnualSavings() > 0 && (
                        <span className="text-xs bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400 px-2 py-0.5 rounded">
                          {t("billing.checkout.save", {
                            amount: getAnnualSavings().toFixed(0),
                          })}
                        </span>
                      )}
                    </div>
                    <div className="text-sm text-muted-foreground">
                      ${plan.price_per_seat_yearly}/seat/year
                    </div>
                  </button>
                </div>
              </div>
            )}

            <div className="flex justify-between">
              <span>{t("billing.checkout.seats")}</span>
              <span className="font-medium">{seats}</span>
            </div>

            <div className="border-t border-border pt-4 flex justify-between text-lg font-semibold">
              <span>{t("billing.checkout.total")}</span>
              <span>
                ${getPrice().toFixed(2)}
                {selectedCycle === "yearly" ? "/year" : "/month"}
              </span>
            </div>
          </div>
        )}

        {orderType === "seat_purchase" && (
          <div className="space-y-4">
            <div className="flex justify-between">
              <span>{t("billing.checkout.additionalSeats")}</span>
              <span className="font-medium">{seats}</span>
            </div>
            <div className="text-sm text-muted-foreground">
              {t("billing.checkout.seatPurchaseNote")}
            </div>
          </div>
        )}
      </div>

      {/* Payment Providers Info */}
      {deploymentInfo && (
        <div className="text-sm text-muted-foreground">
          {deploymentInfo.deployment_type === "global" && (
            <span>{t("billing.checkout.stripePayment")}</span>
          )}
          {deploymentInfo.deployment_type === "cn" && (
            <span>{t("billing.checkout.cnPayment")}</span>
          )}
        </div>
      )}

      {/* Action Buttons */}
      <div className="flex justify-end gap-3">
        <Button variant="outline" onClick={onCancel} disabled={loading}>
          {t("billing.checkout.cancel")}
        </Button>
        <Button onClick={handleCheckout} loading={loading}>
          {loading
            ? t("billing.checkout.processing")
            : t("billing.checkout.proceedToPayment")}
        </Button>
      </div>
    </div>
  );
}

// QR Code checkout component for Alipay/WeChat
interface QRCodeCheckoutProps {
  response: CheckoutResponse;
  t: (key: string, params?: Record<string, string | number>) => string;
  onCancel?: () => void;
}

function QRCodeCheckout({ response, t, onCancel }: QRCodeCheckoutProps) {
  const [checking, setChecking] = useState(false);

  const checkStatus = async () => {
    setChecking(true);
    try {
      const status = await billingApi.getCheckoutStatus(response.order_no);
      if (status.status === "succeeded") {
        // Refresh page to show updated subscription
        window.location.reload();
      }
    } catch (err) {
      console.error("Failed to check status:", err);
    } finally {
      setChecking(false);
    }
  };

  return (
    <div className="space-y-6 text-center">
      <h3 className="text-lg font-semibold">
        {t("billing.checkout.scanToPay")}
      </h3>

      {response.qr_code_url && (
        <div className="flex justify-center">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src={response.qr_code_url}
            alt="Payment QR Code"
            className="w-64 h-64 border border-border rounded-lg"
          />
        </div>
      )}

      <p className="text-sm text-muted-foreground">
        {t("billing.checkout.scanInstructions")}
      </p>

      <div className="text-sm">
        {t("billing.checkout.orderNo")}: {response.order_no}
      </div>

      <div className="flex justify-center gap-3">
        <Button variant="outline" onClick={onCancel}>
          {t("billing.checkout.cancel")}
        </Button>
        <Button onClick={checkStatus} loading={checking}>
          {t("billing.checkout.checkPaymentStatus")}
        </Button>
      </div>
    </div>
  );
}
