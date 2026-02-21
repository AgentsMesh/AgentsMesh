"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  CreditCard,
  Snowflake,
  Play,
  XCircle,
  RefreshCw,
  Settings,
} from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/ui/select";
import {
  getOrganizationSubscription,
  getSubscriptionPlans,
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
import type { SubscriptionPlan } from "@/lib/api/admin";
import { formatDate } from "@/lib/utils";

const QUOTA_RESOURCES = ["users", "runners", "concurrent_pods", "repositories", "pod_minutes"];

function statusVariant(status: string) {
  switch (status) {
    case "active":
      return "success" as const;
    case "frozen":
      return "destructive" as const;
    case "trialing":
      return "default" as const;
    case "canceled":
    case "expired":
      return "secondary" as const;
    default:
      return "outline" as const;
  }
}

function getPlanLimit(plan: SubscriptionPlan, resource: string): number {
  switch (resource) {
    case "users": return plan.max_users;
    case "runners": return plan.max_runners;
    case "concurrent_pods": return plan.max_concurrent_pods;
    case "repositories": return plan.max_repositories;
    case "pod_minutes": return plan.included_pod_minutes;
    default: return 0;
  }
}

export function SubscriptionSection({ orgId }: { orgId: number }) {
  const queryClient = useQueryClient();
  const [renewMonths, setRenewMonths] = useState("1");
  const [newSeatCount, setNewSeatCount] = useState("");
  const [quotaResource, setQuotaResource] = useState("users");
  const [quotaLimit, setQuotaLimit] = useState("");

  const subQueryKey = ["organization-subscription", orgId];

  const { data: sub, isLoading: subLoading } = useQuery({
    queryKey: subQueryKey,
    queryFn: () => getOrganizationSubscription(orgId),
  });

  const { data: plansData } = useQuery({
    queryKey: ["subscription-plans", orgId],
    queryFn: () => getSubscriptionPlans(orgId),
  });

  const plans = plansData?.data || [];

  const invalidate = () => {
    queryClient.invalidateQueries({ queryKey: subQueryKey });
    queryClient.invalidateQueries({ queryKey: ["organization", orgId] });
  };

  const changePlanMutation = useMutation({
    mutationFn: (planName: string) => updateSubscriptionPlan(orgId, planName),
    onSuccess: () => { invalidate(); toast.success("Plan updated"); },
    onError: (err: { error: string }) => toast.error(err.error || "Failed to update plan"),
  });

  const changeSeatsMutation = useMutation({
    mutationFn: (count: number) => updateSubscriptionSeats(orgId, count),
    onSuccess: () => { invalidate(); setNewSeatCount(""); toast.success("Seats updated"); },
    onError: (err: { error: string }) => toast.error(err.error || "Failed to update seats"),
  });

  const changeCycleMutation = useMutation({
    mutationFn: (cycle: string) => updateSubscriptionCycle(orgId, cycle),
    onSuccess: () => { invalidate(); toast.success("Billing cycle updated"); },
    onError: (err: { error: string }) => toast.error(err.error || "Failed to update cycle"),
  });

  const freezeMutation = useMutation({
    mutationFn: () => freezeSubscription(orgId),
    onSuccess: () => { invalidate(); toast.success("Subscription frozen"); },
    onError: (err: { error: string }) => toast.error(err.error || "Failed to freeze"),
  });

  const unfreezeMutation = useMutation({
    mutationFn: () => unfreezeSubscription(orgId),
    onSuccess: () => { invalidate(); toast.success("Subscription unfrozen"); },
    onError: (err: { error: string }) => toast.error(err.error || "Failed to unfreeze"),
  });

  const cancelMutation = useMutation({
    mutationFn: () => cancelSubscription(orgId),
    onSuccess: () => { invalidate(); toast.success("Subscription canceled"); },
    onError: (err: { error: string }) => toast.error(err.error || "Failed to cancel"),
  });

  const renewMutation = useMutation({
    mutationFn: (months: number) => renewSubscription(orgId, months),
    onSuccess: () => { invalidate(); toast.success("Subscription renewed"); },
    onError: (err: { error: string }) => toast.error(err.error || "Failed to renew"),
  });

  const autoRenewMutation = useMutation({
    mutationFn: (autoRenew: boolean) => setSubscriptionAutoRenew(orgId, autoRenew),
    onSuccess: () => { invalidate(); toast.success("Auto-renew updated"); },
    onError: (err: { error: string }) => toast.error(err.error || "Failed to update auto-renew"),
  });

  const quotaMutation = useMutation({
    mutationFn: ({ resource, limit }: { resource: string; limit: number }) =>
      setSubscriptionQuota(orgId, resource, limit),
    onSuccess: () => { invalidate(); setQuotaLimit(""); toast.success("Quota updated"); },
    onError: (err: { error: string }) => toast.error(err.error || "Failed to set quota"),
  });

  if (subLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <CreditCard className="h-5 w-5" />
            Subscription
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="h-10 animate-pulse rounded bg-muted" />
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!sub) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <CreditCard className="h-5 w-5" />
            Subscription
          </CardTitle>
        </CardHeader>
        <CardContent>
          <p className="py-4 text-center text-muted-foreground">No subscription found</p>
        </CardContent>
      </Card>
    );
  }

  const seatUsage = sub.seat_usage;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <CreditCard className="h-5 w-5" />
          Subscription
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Plan & Status + Billing Details */}
        <div className="grid gap-4 md:grid-cols-2">
          <PlanStatusPanel
            sub={sub}
            plans={plans}
            onChangePlan={(v) => {
              if (confirm(`Change plan to "${v}"? This takes effect immediately.`)) {
                changePlanMutation.mutate(v);
              }
            }}
          />
          <BillingDetailsPanel
            sub={sub}
            onToggleCycle={() => {
              const newCycle = sub.billing_cycle === "monthly" ? "yearly" : "monthly";
              changeCycleMutation.mutate(newCycle);
            }}
            onToggleAutoRenew={() => autoRenewMutation.mutate(!sub.auto_renew)}
            cyclePending={changeCycleMutation.isPending}
            autoRenewPending={autoRenewMutation.isPending}
          />
        </div>

        {/* Seats + Actions */}
        <div className="grid gap-4 md:grid-cols-2">
          <SeatsPanel
            seatUsage={seatUsage}
            newSeatCount={newSeatCount}
            onNewSeatCountChange={setNewSeatCount}
            onSetSeats={(count) => changeSeatsMutation.mutate(count)}
            seatsPending={changeSeatsMutation.isPending}
          />
          <ActionsPanel
            status={sub.status}
            renewMonths={renewMonths}
            onRenewMonthsChange={setRenewMonths}
            onFreeze={() => {
              if (confirm("Freeze this subscription? Users will lose access to restricted resources.")) {
                freezeMutation.mutate();
              }
            }}
            onUnfreeze={() => unfreezeMutation.mutate()}
            onCancel={() => {
              if (confirm("Cancel this subscription? This will not call external payment APIs.")) {
                cancelMutation.mutate();
              }
            }}
            onRenew={(months) => {
              if (confirm(`Renew subscription for ${months} month(s)?`)) {
                renewMutation.mutate(months);
              }
            }}
            freezePending={freezeMutation.isPending}
            unfreezePending={unfreezeMutation.isPending}
            cancelPending={cancelMutation.isPending}
            renewPending={renewMutation.isPending}
          />
        </div>

        {/* Custom Quotas */}
        <CustomQuotasPanel
          plan={sub.plan}
          customQuotas={sub.custom_quotas}
          quotaResource={quotaResource}
          quotaLimit={quotaLimit}
          onQuotaResourceChange={setQuotaResource}
          onQuotaLimitChange={setQuotaLimit}
          onSetQuota={(resource, limit) => quotaMutation.mutate({ resource, limit })}
          quotaPending={quotaMutation.isPending}
        />

        {/* Payment Provider Info */}
        {(sub.has_stripe || sub.has_alipay || sub.has_wechat || sub.has_lemonsqueezy) && (
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <span>Payment Integrations:</span>
            {sub.has_stripe && <Badge variant="outline" className="text-xs">Stripe</Badge>}
            {sub.has_alipay && <Badge variant="outline" className="text-xs">Alipay</Badge>}
            {sub.has_wechat && <Badge variant="outline" className="text-xs">WeChat</Badge>}
            {sub.has_lemonsqueezy && <Badge variant="outline" className="text-xs">LemonSqueezy</Badge>}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// ─── Sub-panels ──────────────────────────────────────────────────────────────

function PlanStatusPanel({
  sub,
  plans,
  onChangePlan,
}: {
  sub: { plan?: SubscriptionPlan; status: string; payment_provider?: string; downgrade_to_plan?: string };
  plans: SubscriptionPlan[];
  onChangePlan: (planName: string) => void;
}) {
  return (
    <div className="space-y-3 rounded-lg border border-border p-4">
      <h3 className="text-sm font-semibold text-muted-foreground">Plan & Status</h3>
      <div className="flex items-center justify-between">
        <span className="text-sm">Plan</span>
        <div className="flex items-center gap-2">
          <span className="font-medium capitalize">{sub.plan?.display_name || sub.plan?.name || "-"}</span>
          <Select
            value={sub.plan?.name || ""}
            onValueChange={(v) => { if (v !== sub.plan?.name) onChangePlan(v); }}
          >
            <SelectTrigger className="h-7 w-auto min-w-[100px] text-xs">
              <SelectValue placeholder="Change..." />
            </SelectTrigger>
            <SelectContent>
              {plans.map((p) => (
                <SelectItem key={p.name} value={p.name}>
                  {p.display_name || p.name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </div>
      <div className="flex items-center justify-between">
        <span className="text-sm">Status</span>
        <Badge variant={statusVariant(sub.status)}>{sub.status}</Badge>
      </div>
      {sub.payment_provider && (
        <div className="flex items-center justify-between">
          <span className="text-sm">Provider</span>
          <span className="text-sm text-muted-foreground">{sub.payment_provider}</span>
        </div>
      )}
      {sub.downgrade_to_plan && (
        <div className="flex items-center justify-between">
          <span className="text-sm">Pending Downgrade</span>
          <Badge variant="warning">{sub.downgrade_to_plan}</Badge>
        </div>
      )}
    </div>
  );
}

function BillingDetailsPanel({
  sub,
  onToggleCycle,
  onToggleAutoRenew,
  cyclePending,
  autoRenewPending,
}: {
  sub: { billing_cycle: string; auto_renew: boolean; current_period_start: string; current_period_end: string; next_billing_cycle?: string };
  onToggleCycle: () => void;
  onToggleAutoRenew: () => void;
  cyclePending: boolean;
  autoRenewPending: boolean;
}) {
  return (
    <div className="space-y-3 rounded-lg border border-border p-4">
      <h3 className="text-sm font-semibold text-muted-foreground">Billing Details</h3>
      <div className="flex items-center justify-between">
        <span className="text-sm">Cycle</span>
        <div className="flex items-center gap-2">
          <span className="text-sm capitalize">{sub.billing_cycle}</span>
          <Button variant="outline" size="sm" className="h-7 text-xs" disabled={cyclePending} onClick={onToggleCycle}>
            → {sub.billing_cycle === "monthly" ? "Yearly" : "Monthly"}
          </Button>
        </div>
      </div>
      <div className="flex items-center justify-between">
        <span className="text-sm">Auto-Renew</span>
        <Button
          variant={sub.auto_renew ? "default" : "outline"}
          size="sm"
          className="h-7 text-xs"
          disabled={autoRenewPending}
          onClick={onToggleAutoRenew}
        >
          {sub.auto_renew ? "On" : "Off"}
        </Button>
      </div>
      <div className="flex items-center justify-between">
        <span className="text-sm">Period</span>
        <span className="text-xs text-muted-foreground">
          {formatDate(sub.current_period_start)} — {formatDate(sub.current_period_end)}
        </span>
      </div>
      {sub.next_billing_cycle && (
        <div className="flex items-center justify-between">
          <span className="text-sm">Next Cycle</span>
          <span className="text-sm capitalize text-muted-foreground">{sub.next_billing_cycle}</span>
        </div>
      )}
    </div>
  );
}

function SeatsPanel({
  seatUsage,
  newSeatCount,
  onNewSeatCountChange,
  onSetSeats,
  seatsPending,
}: {
  seatUsage?: { total_seats: number; used_seats: number; available_seats: number };
  newSeatCount: string;
  onNewSeatCountChange: (v: string) => void;
  onSetSeats: (count: number) => void;
  seatsPending: boolean;
}) {
  return (
    <div className="space-y-3 rounded-lg border border-border p-4">
      <h3 className="text-sm font-semibold text-muted-foreground">Seats</h3>
      {seatUsage ? (
        <div className="space-y-2">
          <div className="flex items-center justify-between text-sm">
            <span>{seatUsage.used_seats}/{seatUsage.total_seats} used</span>
            <span className="text-muted-foreground">{seatUsage.available_seats} available</span>
          </div>
          <div className="h-2 rounded-full bg-muted">
            <div
              className="h-2 rounded-full bg-primary transition-all"
              style={{ width: `${seatUsage.total_seats > 0 ? Math.min((seatUsage.used_seats / seatUsage.total_seats) * 100, 100) : 0}%` }}
            />
          </div>
          <div className="flex items-center gap-2">
            <Input
              type="number"
              min={1}
              placeholder="New count"
              value={newSeatCount}
              onChange={(e) => onNewSeatCountChange(e.target.value)}
              className="h-8 w-24 text-sm"
            />
            <Button
              variant="outline"
              size="sm"
              className="h-8 text-xs"
              disabled={!newSeatCount || seatsPending}
              onClick={() => {
                const count = parseInt(newSeatCount);
                if (count > 0) onSetSeats(count);
              }}
            >
              Set Seats
            </Button>
          </div>
        </div>
      ) : (
        <p className="text-sm text-muted-foreground">No seat data</p>
      )}
    </div>
  );
}

function ActionsPanel({
  status,
  renewMonths,
  onRenewMonthsChange,
  onFreeze,
  onUnfreeze,
  onCancel,
  onRenew,
  freezePending,
  unfreezePending,
  cancelPending,
  renewPending,
}: {
  status: string;
  renewMonths: string;
  onRenewMonthsChange: (v: string) => void;
  onFreeze: () => void;
  onUnfreeze: () => void;
  onCancel: () => void;
  onRenew: (months: number) => void;
  freezePending: boolean;
  unfreezePending: boolean;
  cancelPending: boolean;
  renewPending: boolean;
}) {
  return (
    <div className="space-y-3 rounded-lg border border-border p-4">
      <h3 className="text-sm font-semibold text-muted-foreground">Actions</h3>
      <div className="flex flex-wrap gap-2">
        {status !== "frozen" ? (
          <Button variant="outline" size="sm" disabled={freezePending} onClick={onFreeze}>
            <Snowflake className="mr-1.5 h-3.5 w-3.5" />
            Freeze
          </Button>
        ) : (
          <Button variant="outline" size="sm" disabled={unfreezePending} onClick={onUnfreeze}>
            <Play className="mr-1.5 h-3.5 w-3.5" />
            Unfreeze
          </Button>
        )}
        {status !== "canceled" && (
          <Button variant="destructive" size="sm" disabled={cancelPending} onClick={onCancel}>
            <XCircle className="mr-1.5 h-3.5 w-3.5" />
            Cancel
          </Button>
        )}
      </div>
      <div className="flex items-center gap-2 pt-1">
        <Input
          type="number"
          min={1}
          max={120}
          value={renewMonths}
          onChange={(e) => onRenewMonthsChange(e.target.value)}
          className="h-8 w-20 text-sm"
        />
        <span className="text-xs text-muted-foreground">months</span>
        <Button
          variant="outline"
          size="sm"
          className="h-8"
          disabled={renewPending}
          onClick={() => {
            const months = parseInt(renewMonths);
            if (months > 0 && months <= 120) onRenew(months);
          }}
        >
          <RefreshCw className="mr-1.5 h-3.5 w-3.5" />
          Renew
        </Button>
      </div>
    </div>
  );
}

function CustomQuotasPanel({
  plan,
  customQuotas,
  quotaResource,
  quotaLimit,
  onQuotaResourceChange,
  onQuotaLimitChange,
  onSetQuota,
  quotaPending,
}: {
  plan?: SubscriptionPlan;
  customQuotas: Record<string, number> | null;
  quotaResource: string;
  quotaLimit: string;
  onQuotaResourceChange: (v: string) => void;
  onQuotaLimitChange: (v: string) => void;
  onSetQuota: (resource: string, limit: number) => void;
  quotaPending: boolean;
}) {
  return (
    <div className="space-y-3 rounded-lg border border-border p-4">
      <h3 className="flex items-center gap-2 text-sm font-semibold text-muted-foreground">
        <Settings className="h-4 w-4" />
        Custom Quotas
      </h3>

      {plan && (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b text-left text-muted-foreground">
                <th className="pb-2 pr-4 font-medium">Resource</th>
                <th className="pb-2 pr-4 font-medium">Plan Limit</th>
                <th className="pb-2 pr-4 font-medium">Custom Override</th>
              </tr>
            </thead>
            <tbody>
              {QUOTA_RESOURCES.map((res) => {
                const planLimit = getPlanLimit(plan, res);
                const customVal = customQuotas?.[res];
                return (
                  <tr key={res} className="border-b border-border/50">
                    <td className="py-2 pr-4 font-mono text-xs">{res}</td>
                    <td className="py-2 pr-4 text-muted-foreground">
                      {planLimit === -1 ? "∞" : planLimit}
                    </td>
                    <td className="py-2 pr-4">
                      {customVal != null ? (
                        <Badge variant="outline" className="font-mono">
                          {Number(customVal) === -1 ? "∞" : String(customVal)}
                        </Badge>
                      ) : (
                        <span className="text-muted-foreground">—</span>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      <div className="flex items-center gap-2 pt-1">
        <Select value={quotaResource} onValueChange={onQuotaResourceChange}>
          <SelectTrigger className="h-8 w-40 text-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {QUOTA_RESOURCES.map((r) => (
              <SelectItem key={r} value={r}>
                {r}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Input
          type="number"
          placeholder="Limit (-1=∞)"
          value={quotaLimit}
          onChange={(e) => onQuotaLimitChange(e.target.value)}
          className="h-8 w-28 text-sm"
        />
        <Button
          variant="outline"
          size="sm"
          className="h-8 text-xs"
          disabled={quotaLimit === "" || quotaPending}
          onClick={() => {
            const limit = parseInt(quotaLimit);
            if (!isNaN(limit)) onSetQuota(quotaResource, limit);
          }}
        >
          Set Override
        </Button>
      </div>
    </div>
  );
}
