import { Users, User, Building2 } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { PromoCodeRedemption } from "@/lib/api/admin";
import { formatDate } from "@/lib/utils";

export function RedemptionsTable({
  redemptions,
  isLoading,
}: {
  redemptions: PromoCodeRedemption[];
  isLoading: boolean;
}) {
  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Users className="h-5 w-5" />
          Redemptions ({redemptions.length})
        </CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="h-14 animate-pulse rounded-lg bg-muted" />
            ))}
          </div>
        ) : redemptions.length === 0 ? (
          <p className="py-4 text-center text-muted-foreground">No redemptions yet</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>User</TableHead>
                <TableHead>Organization</TableHead>
                <TableHead>Plan</TableHead>
                <TableHead>Duration</TableHead>
                <TableHead>Redeemed At</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {redemptions.map((redemption) => (
                <RedemptionRow key={redemption.id} redemption={redemption} />
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  );
}

function RedemptionRow({ redemption }: { redemption: PromoCodeRedemption }) {
  return (
    <TableRow>
      <TableCell>
        <div className="flex items-center gap-2">
          {redemption.user?.avatar_url ? (
            <img
              src={redemption.user.avatar_url}
              alt={redemption.user.username}
              className="h-8 w-8 rounded-full"
            />
          ) : (
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/20">
              <User className="h-4 w-4 text-primary" />
            </div>
          )}
          <div>
            <p className="font-medium">
              {redemption.user?.name || redemption.user?.username || "Unknown"}
            </p>
            <p className="text-xs text-muted-foreground">{redemption.user?.email}</p>
          </div>
        </div>
      </TableCell>
      <TableCell>
        <div className="flex items-center gap-2">
          <Building2 className="h-4 w-4 text-muted-foreground" />
          {redemption.organization?.name || "Unknown"}
        </div>
      </TableCell>
      <TableCell className="capitalize">{redemption.plan_name}</TableCell>
      <TableCell>{redemption.duration_months} months</TableCell>
      <TableCell>{formatDate(redemption.created_at)}</TableCell>
    </TableRow>
  );
}
