import { ArrowLeft, Radio, ArrowRightLeft, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import type { RelayDetailHeaderProps } from "./relay-detail-types";

export function RelayDetailHeader({
  relay,
  healthyRelays,
  isUnregistering,
  onUnregister,
  onBack,
}: RelayDetailHeaderProps) {
  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex items-center gap-4">
        <Button variant="ghost" onClick={onBack}>
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back
        </Button>
        <div className="flex items-center gap-2">
          <Radio className="h-5 w-5" />
          <h1 className="text-xl font-semibold">{relay.id}</h1>
          <Badge variant={relay.healthy ? "success" : "destructive"}>
            {relay.healthy ? "Healthy" : "Unhealthy"}
          </Badge>
        </div>
      </div>
      <div className="flex gap-2">
        <Button
          variant="outline"
          onClick={() => onUnregister(true)}
          disabled={isUnregistering || healthyRelays.length === 0}
        >
          <ArrowRightLeft className="mr-2 h-4 w-4" />
          Unregister & Migrate
        </Button>
        <Button
          variant="destructive"
          onClick={() => onUnregister(false)}
          disabled={isUnregistering}
        >
          <Trash2 className="mr-2 h-4 w-4" />
          Force Unregister
        </Button>
      </div>
    </div>
  );
}
