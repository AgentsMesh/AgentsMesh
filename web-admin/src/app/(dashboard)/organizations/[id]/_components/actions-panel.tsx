import { Snowflake, Play, XCircle, RefreshCw } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

export function ActionsPanel({
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
