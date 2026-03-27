import { Clock, Activity, ArrowRightLeft, RefreshCw } from "lucide-react";
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { formatRelativeTime } from "@/lib/utils";
import type { ActiveSession } from "@/lib/api/admin";
import type { RelaySessionsTableProps } from "./relay-detail-types";

export function RelaySessionsTable({
  sessions,
  healthyRelays,
  migratingPod,
  targetRelay,
  isMigratingSession,
  onSetTargetRelay,
  onMigrate,
  onCancelMigrate,
}: RelaySessionsTableProps) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Active Sessions ({sessions?.length || 0})</CardTitle>
      </CardHeader>
      <CardContent>
        {!sessions || sessions.length === 0 ? (
          <div className="py-8 text-center text-muted-foreground">
            <Activity className="mx-auto mb-2 h-8 w-8" />
            <p>No active sessions on this relay</p>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Pod Key</TableHead>
                <TableHead>Session ID</TableHead>
                <TableHead>Created</TableHead>
                <TableHead>Expires</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {sessions.map((session) => (
                <SessionRow
                  key={session.pod_key}
                  session={session}
                  healthyRelays={healthyRelays}
                  isMigrating={migratingPod === session.pod_key}
                  targetRelay={targetRelay}
                  isMigratingSession={isMigratingSession}
                  onSetTargetRelay={onSetTargetRelay}
                  onMigrate={onMigrate}
                  onCancelMigrate={onCancelMigrate}
                />
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}

function SessionRow({
  session,
  healthyRelays,
  isMigrating,
  targetRelay,
  isMigratingSession,
  onSetTargetRelay,
  onMigrate,
  onCancelMigrate,
}: {
  session: ActiveSession;
  healthyRelays: RelaySessionsTableProps["healthyRelays"];
  isMigrating: boolean;
  targetRelay: string;
  isMigratingSession: boolean;
  onSetTargetRelay: (v: string) => void;
  onMigrate: (session: ActiveSession) => void;
  onCancelMigrate: () => void;
}) {
  return (
    <TableRow>
      <TableCell className="font-mono text-sm">{session.pod_key}</TableCell>
      <TableCell className="font-mono text-sm">{session.session_id}</TableCell>
      <TableCell className="text-muted-foreground">
        <div className="flex items-center gap-1">
          <Clock className="h-3 w-3" />
          {formatRelativeTime(session.created_at)}
        </div>
      </TableCell>
      <TableCell className="text-muted-foreground">
        {formatRelativeTime(session.expire_at)}
      </TableCell>
      <TableCell className="text-right">
        {isMigrating ? (
          <div className="flex items-center justify-end gap-2">
            <Select value={targetRelay} onValueChange={onSetTargetRelay}>
              <SelectTrigger className="w-40">
                <SelectValue placeholder="Target relay" />
              </SelectTrigger>
              <SelectContent>
                {healthyRelays.map((r) => (
                  <SelectItem key={r.id} value={r.id}>
                    {r.id}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button
              size="sm"
              onClick={() => onMigrate(session)}
              disabled={isMigratingSession || !targetRelay}
            >
              {isMigratingSession ? (
                <RefreshCw className="h-4 w-4 animate-spin" />
              ) : (
                "Confirm"
              )}
            </Button>
            <Button size="sm" variant="ghost" onClick={onCancelMigrate}>
              Cancel
            </Button>
          </div>
        ) : (
          <Button
            size="sm"
            variant="ghost"
            onClick={() => onMigrate(session)}
            disabled={healthyRelays.length === 0}
            title={
              healthyRelays.length === 0
                ? "No other healthy relays available"
                : "Migrate to another relay"
            }
          >
            <ArrowRightLeft className="h-4 w-4" />
          </Button>
        )}
      </TableCell>
    </TableRow>
  );
}
