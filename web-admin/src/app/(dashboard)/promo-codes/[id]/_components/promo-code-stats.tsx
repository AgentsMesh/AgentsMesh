import { Tag, Clock, Users, Calendar } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { PromoCode, PromoCodeType } from "@/lib/api/admin";
import { formatDate } from "@/lib/utils";

const typeLabels: Record<PromoCodeType, string> = {
  media: "Media",
  partner: "Partner",
  campaign: "Campaign",
  internal: "Internal",
  referral: "Referral",
};

export function PromoCodeStats({
  promoCode,
  isExpired,
}: {
  promoCode: PromoCode;
  isExpired: boolean;
}) {
  const remainingUses =
    promoCode.max_uses === null
      ? "Unlimited"
      : `${promoCode.max_uses - promoCode.used_count}`;

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">Type</CardTitle>
          <Tag className="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-lg font-semibold capitalize">{typeLabels[promoCode.type]}</div>
          <p className="text-xs text-muted-foreground">Plan: {promoCode.plan_name}</p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">Duration</CardTitle>
          <Clock className="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{promoCode.duration_months}</div>
          <p className="text-xs text-muted-foreground">months</p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">Uses</CardTitle>
          <Users className="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{promoCode.used_count}</div>
          <p className="text-xs text-muted-foreground">{remainingUses} remaining</p>
        </CardContent>
      </Card>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between pb-2">
          <CardTitle className="text-sm font-medium text-muted-foreground">Expires</CardTitle>
          <Calendar className="h-4 w-4 text-muted-foreground" />
        </CardHeader>
        <CardContent>
          <div className={`text-sm font-medium ${isExpired ? "text-destructive" : ""}`}>
            {promoCode.expires_at ? formatDate(promoCode.expires_at) : "Never"}
          </div>
          <p className="text-xs text-muted-foreground">Created {formatDate(promoCode.created_at)}</p>
        </CardContent>
      </Card>
    </div>
  );
}
