import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/ui/select";
import type { SubscriptionPlan } from "@/lib/api/admin";
import { statusVariant } from "./subscription-utils";

export function PlanStatusPanel({
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
