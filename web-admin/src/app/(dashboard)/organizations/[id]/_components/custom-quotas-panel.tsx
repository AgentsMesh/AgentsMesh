import { Settings } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/ui/select";
import type { SubscriptionPlan } from "@/lib/api/admin";
import { QUOTA_RESOURCES, getPlanLimit } from "./subscription-utils";

export function CustomQuotasPanel({
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
                      {planLimit === -1 ? "\u221E" : planLimit}
                    </td>
                    <td className="py-2 pr-4">
                      {customVal != null ? (
                        <Badge variant="outline" className="font-mono">
                          {Number(customVal) === -1 ? "\u221E" : String(customVal)}
                        </Badge>
                      ) : (
                        <span className="text-muted-foreground">&mdash;</span>
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
          placeholder="Limit (-1=\u221E)"
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
