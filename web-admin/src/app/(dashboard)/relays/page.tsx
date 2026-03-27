"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { ArrowRightLeft, RefreshCw } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  listRelays,
  getRelayStats,
  forceUnregisterRelay,
  bulkMigrateSessions,
  RelayInfo,
  RelayStats,
  RelayListResponse,
} from "@/lib/api/admin";
import { RelayStatsCards } from "./relay-stats-cards";
import { RelayListCard } from "./relay-list-card";

export default function RelaysPage() {
  const router = useRouter();
  const [selectedSource, setSelectedSource] = useState<string>("");
  const [selectedTarget, setSelectedTarget] = useState<string>("");
  const [isMigrating, setIsMigrating] = useState(false);

  const [relaysData, setRelaysData] = useState<RelayListResponse | null>(null);
  const [stats, setStats] = useState<RelayStats | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const fetchRelays = useCallback(async () => {
    try {
      const result = await listRelays();
      setRelaysData(result);
    } catch {
      // Keep previous data on error
    } finally {
      setIsLoading(false);
    }
  }, []);

  const fetchStats = useCallback(async () => {
    try {
      const result = await getRelayStats();
      setStats(result);
    } catch {
      // Non-critical
    }
  }, []);

  useEffect(() => {
    fetchRelays();
    fetchStats();
    const interval = setInterval(() => { fetchRelays(); fetchStats(); }, 10000);
    return () => clearInterval(interval);
  }, [fetchRelays, fetchStats]);

  const handleUnregister = async (relay: RelayInfo, migrate: boolean) => {
    const msg = migrate
      ? `Unregister relay "${relay.id}" and migrate all sessions to another relay?`
      : `Unregister relay "${relay.id}"? ${relay.connections} active connections will be affected.`;
    if (!confirm(msg)) return;
    try {
      const data = await forceUnregisterRelay(relay.id, migrate);
      toast.success(`Relay unregistered. ${data.affected_sessions} sessions affected.`);
      await fetchRelays();
      await fetchStats();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to unregister relay");
    }
  };

  const handleBulkMigrate = async () => {
    if (!selectedSource || !selectedTarget) {
      toast.error("Please select source and target relays");
      return;
    }
    if (selectedSource === selectedTarget) {
      toast.error("Source and target cannot be the same");
      return;
    }
    if (!confirm(`Migrate all sessions from "${selectedSource}" to "${selectedTarget}"?`)) return;
    setIsMigrating(true);
    try {
      const data = await bulkMigrateSessions(selectedSource, selectedTarget);
      toast.success(`Migration completed: ${data.migrated}/${data.total} sessions migrated`);
      setSelectedSource("");
      setSelectedTarget("");
      await fetchRelays();
      await fetchStats();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to migrate sessions");
    } finally {
      setIsMigrating(false);
    }
  };

  const healthyRelays = relaysData?.data.filter((r) => r.healthy) || [];

  return (
    <div className="space-y-4">
      <RelayStatsCards stats={stats} />

      {healthyRelays.length >= 2 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Bulk Session Migration</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
              <div className="flex-1">
                <Select value={selectedSource} onValueChange={setSelectedSource}>
                  <SelectTrigger>
                    <SelectValue placeholder="Source Relay" />
                  </SelectTrigger>
                  <SelectContent>
                    {relaysData?.data.map((relay) => (
                      <SelectItem key={relay.id} value={relay.id}>
                        {relay.id} ({relay.connections} connections)
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <ArrowRightLeft className="h-4 w-4 text-muted-foreground" />
              <div className="flex-1">
                <Select value={selectedTarget} onValueChange={setSelectedTarget}>
                  <SelectTrigger>
                    <SelectValue placeholder="Target Relay" />
                  </SelectTrigger>
                  <SelectContent>
                    {healthyRelays
                      .filter((r) => r.id !== selectedSource)
                      .map((relay) => (
                        <SelectItem key={relay.id} value={relay.id}>
                          {relay.id} ({relay.region})
                        </SelectItem>
                      ))}
                  </SelectContent>
                </Select>
              </div>
              <Button
                onClick={handleBulkMigrate}
                disabled={!selectedSource || !selectedTarget || isMigrating}
              >
                {isMigrating ? (
                  <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <ArrowRightLeft className="mr-2 h-4 w-4" />
                )}
                Migrate
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      <RelayListCard
        relaysData={relaysData}
        isLoading={isLoading}
        onRelayClick={(id) => router.push(`/relays/${encodeURIComponent(id)}`)}
        onUnregister={handleUnregister}
      />
    </div>
  );
}
