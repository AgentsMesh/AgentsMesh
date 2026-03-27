import {
  Wifi,
  WifiOff,
  ArrowRightLeft,
  Trash2,
  Radio,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { RelayInfo, RelayListResponse } from "@/lib/api/admin";
import { formatRelativeTime } from "@/lib/utils";

interface RelayListCardProps {
  relaysData: RelayListResponse | null;
  isLoading: boolean;
  onRelayClick: (relayId: string) => void;
  onUnregister: (relay: RelayInfo, migrate: boolean) => void;
}

export function RelayListCard({
  relaysData,
  isLoading,
  onRelayClick,
  onUnregister,
}: RelayListCardProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Relay Servers ({relaysData?.total || 0})</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="h-20 animate-pulse rounded-lg bg-muted" />
            ))}
          </div>
        ) : relaysData?.data.length === 0 ? (
          <div className="py-8 text-center text-muted-foreground">
            <Radio className="mx-auto mb-2 h-8 w-8" />
            <p>No relay servers registered</p>
            <p className="text-sm">
              Relay servers will appear here once they connect to the backend.
            </p>
          </div>
        ) : (
          <div className="space-y-2">
            {relaysData?.data.map((relay) => (
              <RelayRow
                key={relay.id}
                relay={relay}
                onClick={() => onRelayClick(relay.id)}
                onUnregister={(migrate) => onUnregister(relay, migrate)}
              />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function RelayRow({
  relay,
  onClick,
  onUnregister,
}: {
  relay: RelayInfo;
  onClick: () => void;
  onUnregister: (migrate: boolean) => void;
}) {
  const loadPercent =
    relay.capacity > 0
      ? Math.round((relay.connections / relay.capacity) * 100)
      : 0;

  return (
    <div
      className="flex flex-col gap-3 rounded-lg border border-border p-4 cursor-pointer hover:bg-accent/50 transition-colors sm:flex-row sm:items-center sm:justify-between"
      onClick={onClick}
    >
      <div className="flex items-center gap-4">
        <div
          className={`flex h-10 w-10 items-center justify-center rounded-lg ${
            relay.healthy ? "bg-green-100" : "bg-red-100"
          }`}
        >
          {relay.healthy ? (
            <Wifi className="h-5 w-5 text-green-600" />
          ) : (
            <WifiOff className="h-5 w-5 text-red-600" />
          )}
        </div>
        <div>
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-medium">{relay.id}</span>
            <Badge variant={relay.healthy ? "success" : "destructive"}>
              {relay.healthy ? "Healthy" : "Unhealthy"}
            </Badge>
            {relay.region && <Badge variant="outline">{relay.region}</Badge>}
          </div>
          <div className="flex items-center gap-3 text-sm text-muted-foreground">
            <span>{relay.url}</span>
          </div>
        </div>
      </div>
      <div className="flex items-center gap-4 sm:gap-6">
        <div className="text-right">
          <div className="flex items-center gap-2">
            <div className="w-24 h-2 bg-secondary rounded-full overflow-hidden">
              <div
                className={`h-full ${
                  loadPercent > 80
                    ? "bg-red-500"
                    : loadPercent > 50
                    ? "bg-yellow-500"
                    : "bg-green-500"
                }`}
                style={{ width: `${loadPercent}%` }}
              />
            </div>
            <span className="text-sm w-16 text-right">
              {relay.connections}/{relay.capacity}
            </span>
          </div>
          <div className="text-xs text-muted-foreground">
            CPU: {relay.cpu_usage?.toFixed(1) || 0}% | Mem:{" "}
            {relay.memory_usage?.toFixed(1) || 0}%
          </div>
        </div>
        <div className="hidden text-right text-xs text-muted-foreground sm:block">
          {relay.last_heartbeat && (
            <p>Last seen {formatRelativeTime(relay.last_heartbeat)}</p>
          )}
        </div>
        <div className="flex gap-1" onClick={(e) => e.stopPropagation()}>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => onUnregister(true)}
            title="Unregister with migration"
          >
            <ArrowRightLeft className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => onUnregister(false)}
            title="Force unregister"
            className="text-destructive hover:text-destructive"
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
