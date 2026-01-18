"use client";

import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { billingApi, SeatUsage } from "@/lib/api/billing";

interface SeatManagementProps {
  t: (key: string, params?: Record<string, string | number>) => string;
  currentUrl: string;
  onPurchaseInitiated?: () => void;
}

export function SeatManagement({
  t,
  currentUrl,
  onPurchaseInitiated,
}: SeatManagementProps) {
  const [loading, setLoading] = useState(true);
  const [seatUsage, setSeatUsage] = useState<SeatUsage | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [showPurchase, setShowPurchase] = useState(false);
  const [seatsToAdd, setSeatsToAdd] = useState(1);
  const [purchasing, setPurchasing] = useState(false);

  useEffect(() => {
    loadSeatUsage();
  }, []);

  const loadSeatUsage = async () => {
    setLoading(true);
    setError(null);
    try {
      const usage = await billingApi.getSeatUsage();
      setSeatUsage(usage);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load seat data");
    } finally {
      setLoading(false);
    }
  };

  const handlePurchaseSeats = async () => {
    if (seatsToAdd < 1) return;

    setPurchasing(true);
    try {
      const response = await billingApi.purchaseSeats(
        seatsToAdd,
        `${currentUrl}?payment=success`,
        `${currentUrl}?payment=cancelled`
      );

      onPurchaseInitiated?.();

      if (response.session_url) {
        window.location.href = response.session_url;
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to initiate purchase");
    } finally {
      setPurchasing(false);
    }
  };

  if (loading) {
    return (
      <div className="border border-border rounded-lg p-6 animate-pulse">
        <div className="h-6 bg-muted rounded w-32 mb-4"></div>
        <div className="h-4 bg-muted rounded w-48"></div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="border border-border rounded-lg p-6">
        <p className="text-destructive">{error}</p>
        <Button variant="outline" className="mt-4" onClick={loadSeatUsage}>
          {t("billing.seats.retry")}
        </Button>
      </div>
    );
  }

  if (!seatUsage) return null;

  const usagePercent =
    seatUsage.max_seats === -1
      ? 0
      : (seatUsage.used_seats / seatUsage.total_seats) * 100;

  return (
    <div className="border border-border rounded-lg p-6">
      <h2 className="text-lg font-semibold mb-4">
        {t("billing.seats.title")}
      </h2>

      {/* Seat Usage Overview */}
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <div className="text-2xl font-bold">
              {seatUsage.used_seats} / {seatUsage.total_seats}
            </div>
            <div className="text-sm text-muted-foreground">
              {t("billing.seats.seatsUsed")}
            </div>
          </div>
          <div className="text-right">
            <div className="text-lg font-medium text-green-600 dark:text-green-400">
              {seatUsage.available_seats}
            </div>
            <div className="text-sm text-muted-foreground">
              {t("billing.seats.available")}
            </div>
          </div>
        </div>

        {/* Progress Bar */}
        <div className="w-full bg-muted rounded-full h-3">
          <div
            className={`h-3 rounded-full transition-all ${
              usagePercent > 90 ? "bg-destructive" : "bg-primary"
            }`}
            style={{ width: `${Math.min(100, usagePercent)}%` }}
          ></div>
        </div>

        {/* Max Seats Info */}
        {seatUsage.max_seats !== -1 && (
          <div className="text-sm text-muted-foreground">
            {t("billing.seats.maxSeats", { max: seatUsage.max_seats })}
          </div>
        )}

        {/* Purchase Seats Section */}
        {seatUsage.can_add_seats ? (
          <>
            {!showPurchase ? (
              <Button
                variant="outline"
                onClick={() => setShowPurchase(true)}
                className="w-full"
              >
                {t("billing.seats.addSeats")}
              </Button>
            ) : (
              <div className="space-y-4 pt-4 border-t border-border">
                <h3 className="font-medium">
                  {t("billing.seats.purchaseSeats")}
                </h3>

                <div className="flex items-center gap-4">
                  <div className="flex-1">
                    <label className="text-sm text-muted-foreground">
                      {t("billing.seats.seatsToAdd")}
                    </label>
                    <Input
                      type="number"
                      min={1}
                      max={
                        seatUsage.max_seats === -1
                          ? 100
                          : seatUsage.max_seats - seatUsage.total_seats
                      }
                      value={seatsToAdd}
                      onChange={(e) =>
                        setSeatsToAdd(Math.max(1, parseInt(e.target.value) || 1))
                      }
                      className="mt-1"
                    />
                  </div>
                </div>

                <p className="text-sm text-muted-foreground">
                  {t("billing.seats.proratedNote")}
                </p>

                <div className="flex gap-3">
                  <Button
                    variant="outline"
                    onClick={() => setShowPurchase(false)}
                    disabled={purchasing}
                  >
                    {t("billing.seats.cancelPurchase")}
                  </Button>
                  <Button
                    onClick={handlePurchaseSeats}
                    loading={purchasing}
                    disabled={seatsToAdd < 1}
                  >
                    {purchasing
                      ? t("billing.seats.processing")
                      : t("billing.seats.proceedToPurchase")}
                  </Button>
                </div>
              </div>
            )}
          </>
        ) : (
          <div className="p-4 bg-muted/50 rounded-lg text-sm text-muted-foreground">
            {t("billing.seats.upgradeToAddSeats")}
          </div>
        )}
      </div>
    </div>
  );
}
