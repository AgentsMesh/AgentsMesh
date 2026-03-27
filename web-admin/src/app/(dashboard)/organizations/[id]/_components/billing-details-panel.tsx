import { Button } from "@/components/ui/button";
import { formatDate } from "@/lib/utils";

export function BillingDetailsPanel({
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
