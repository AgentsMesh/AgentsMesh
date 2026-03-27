"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { billingApi, CheckoutResponse } from "@/lib/api/billing";

interface QRCodeCheckoutProps {
  response: CheckoutResponse;
  t: (key: string, params?: Record<string, string | number>) => string;
  onCancel?: () => void;
}

export function QRCodeCheckout({ response, t, onCancel }: QRCodeCheckoutProps) {
  const [checking, setChecking] = useState(false);

  const checkStatus = async () => {
    setChecking(true);
    try {
      const status = await billingApi.getCheckoutStatus(response.order_no);
      if (status.status === "succeeded") window.location.reload();
    } catch (err) {
      console.error("Failed to check status:", err);
    } finally { setChecking(false); }
  };

  return (
    <div className="space-y-6 text-center">
      <h3 className="text-lg font-semibold">{t("billing.checkout.scanToPay")}</h3>
      {response.qr_code_url && (
        <div className="flex justify-center">
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={response.qr_code_url} alt="Payment QR Code"
            className="w-64 h-64 border border-border rounded-lg" />
        </div>
      )}
      <p className="text-sm text-muted-foreground">{t("billing.checkout.scanInstructions")}</p>
      <div className="text-sm">{t("billing.checkout.orderNo")}: {response.order_no}</div>
      <div className="flex justify-center gap-3">
        <Button variant="outline" onClick={onCancel}>{t("billing.checkout.cancel")}</Button>
        <Button onClick={checkStatus} loading={checking}>{t("billing.checkout.checkPaymentStatus")}</Button>
      </div>
    </div>
  );
}
