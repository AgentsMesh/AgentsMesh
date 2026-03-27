import { Wifi, WifiOff } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatRelativeTime } from "@/lib/utils";
import type { RelayInfoCardsProps } from "./relay-detail-types";

export function RelayInfoCards({ relay, sessionCount }: RelayInfoCardsProps) {
  const loadPercent =
    relay.capacity > 0
      ? Math.round((relay.connections / relay.capacity) * 100)
      : 0;

  return (
    <div className="grid gap-4 md:grid-cols-2">
      <ConnectionInfoCard relay={relay} />
      <LoadResourcesCard
        relay={relay}
        loadPercent={loadPercent}
        sessionCount={sessionCount}
      />
    </div>
  );
}

function ConnectionInfoCard({ relay }: { relay: RelayInfoCardsProps["relay"] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Connection Info</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">Status</span>
          <div className="flex items-center gap-2">
            {relay.healthy ? (
              <Wifi className="h-4 w-4 text-green-500" />
            ) : (
              <WifiOff className="h-4 w-4 text-red-500" />
            )}
            <span>{relay.healthy ? "Online" : "Offline"}</span>
          </div>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">URL</span>
          <span className="text-sm font-mono">{relay.url}</span>
        </div>
        {relay.internal_url && (
          <div className="flex items-center justify-between">
            <span className="text-sm text-muted-foreground">Internal URL</span>
            <span className="text-sm font-mono">{relay.internal_url}</span>
          </div>
        )}
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">Region</span>
          <Badge variant="outline">{relay.region || "default"}</Badge>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">Last Heartbeat</span>
          <span className="text-sm">
            {relay.last_heartbeat
              ? formatRelativeTime(relay.last_heartbeat)
              : "Never"}
          </span>
        </div>
      </CardContent>
    </Card>
  );
}

function LoadResourcesCard({
  relay,
  loadPercent,
  sessionCount,
}: {
  relay: RelayInfoCardsProps["relay"];
  loadPercent: number;
  sessionCount: number;
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Load & Resources</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div>
          <div className="flex items-center justify-between mb-1">
            <span className="text-sm text-muted-foreground">Connections</span>
            <span className="text-sm">
              {relay.connections} / {relay.capacity}
            </span>
          </div>
          <div className="w-full h-2 bg-secondary rounded-full overflow-hidden">
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
        </div>
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">CPU Usage</span>
          <span className="text-sm">{relay.cpu_usage?.toFixed(1) || 0}%</span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">Memory Usage</span>
          <span className="text-sm">{relay.memory_usage?.toFixed(1) || 0}%</span>
        </div>
        <div className="flex items-center justify-between">
          <span className="text-sm text-muted-foreground">Active Sessions</span>
          <span className="text-sm font-medium">{sessionCount}</span>
        </div>
      </CardContent>
    </Card>
  );
}
