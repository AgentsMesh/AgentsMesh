"use client";

import { useState } from "react";
import { toast } from "sonner";
import {
  updateSubscriptionPlan,
  updateSubscriptionSeats,
  updateSubscriptionCycle,
  freezeSubscription,
  unfreezeSubscription,
  cancelSubscription,
  renewSubscription,
  setSubscriptionAutoRenew,
  setSubscriptionQuota,
} from "@/lib/api/admin";

interface Setters {
  setNewSeatCount: (v: string) => void;
  setQuotaLimit: (v: string) => void;
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function useSubscriptionActions(orgId: number, sub: any, refreshData: () => Promise<void>, setters: Setters) {
  const [changePlanPending, setChangePlanPending] = useState(false);
  const [changeSeatsPending, setChangeSeatsPending] = useState(false);
  const [changeCyclePending, setChangeCyclePending] = useState(false);
  const [freezePending, setFreezePending] = useState(false);
  const [unfreezePending, setUnfreezePending] = useState(false);
  const [cancelPending, setCancelPending] = useState(false);
  const [renewPending, setRenewPending] = useState(false);
  const [autoRenewPending, setAutoRenewPending] = useState(false);
  const [quotaPending, setQuotaPending] = useState(false);

  const handleChangePlan = async (planName: string) => {
    if (!confirm(`Change plan to "${planName}"? This takes effect immediately.`)) return;
    setChangePlanPending(true);
    try {
      await updateSubscriptionPlan(orgId, planName);
      toast.success("Plan updated");
      await refreshData();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to update plan");
    } finally {
      setChangePlanPending(false);
    }
  };

  const handleChangeSeats = async (count: number) => {
    setChangeSeatsPending(true);
    try {
      await updateSubscriptionSeats(orgId, count);
      setters.setNewSeatCount("");
      toast.success("Seats updated");
      await refreshData();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to update seats");
    } finally {
      setChangeSeatsPending(false);
    }
  };

  const handleChangeCycle = async () => {
    const newCycle = sub.billing_cycle === "monthly" ? "yearly" : "monthly";
    setChangeCyclePending(true);
    try {
      await updateSubscriptionCycle(orgId, newCycle);
      toast.success("Billing cycle updated");
      await refreshData();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to update cycle");
    } finally {
      setChangeCyclePending(false);
    }
  };

  const handleFreeze = async () => {
    if (!confirm("Freeze this subscription? Users will lose access to restricted resources.")) return;
    setFreezePending(true);
    try {
      await freezeSubscription(orgId);
      toast.success("Subscription frozen");
      await refreshData();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to freeze");
    } finally {
      setFreezePending(false);
    }
  };

  const handleUnfreeze = async () => {
    setUnfreezePending(true);
    try {
      await unfreezeSubscription(orgId);
      toast.success("Subscription unfrozen");
      await refreshData();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to unfreeze");
    } finally {
      setUnfreezePending(false);
    }
  };

  const handleCancel = async () => {
    if (!confirm("Cancel this subscription? This will not call external payment APIs.")) return;
    setCancelPending(true);
    try {
      await cancelSubscription(orgId);
      toast.success("Subscription canceled");
      await refreshData();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to cancel");
    } finally {
      setCancelPending(false);
    }
  };

  const handleRenew = async (months: number) => {
    if (!confirm(`Renew subscription for ${months} month(s)?`)) return;
    setRenewPending(true);
    try {
      await renewSubscription(orgId, months);
      toast.success("Subscription renewed");
      await refreshData();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to renew");
    } finally {
      setRenewPending(false);
    }
  };

  const handleToggleAutoRenew = async () => {
    setAutoRenewPending(true);
    try {
      await setSubscriptionAutoRenew(orgId, !sub.auto_renew);
      toast.success("Auto-renew updated");
      await refreshData();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to update auto-renew");
    } finally {
      setAutoRenewPending(false);
    }
  };

  const handleSetQuota = async (resource: string, limit: number) => {
    setQuotaPending(true);
    try {
      await setSubscriptionQuota(orgId, resource, limit);
      setters.setQuotaLimit("");
      toast.success("Quota updated");
      await refreshData();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to set quota");
    } finally {
      setQuotaPending(false);
    }
  };

  return {
    changePlanPending, changeSeatsPending, changeCyclePending,
    freezePending, unfreezePending, cancelPending, renewPending,
    autoRenewPending, quotaPending,
    handleChangePlan, handleChangeSeats, handleChangeCycle,
    handleFreeze, handleUnfreeze, handleCancel, handleRenew,
    handleToggleAutoRenew, handleSetQuota,
  };
}
