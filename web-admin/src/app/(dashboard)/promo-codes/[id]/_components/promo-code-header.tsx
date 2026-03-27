import Link from "next/link";
import { ArrowLeft, Tag, Power, PowerOff, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import type { PromoCode } from "@/lib/api/admin";

export function PromoCodeHeader({
  promoCode,
  isExpired,
  isDeleting,
  onActivate,
  onDeactivate,
  onDelete,
}: {
  promoCode: PromoCode;
  isExpired: boolean;
  isDeleting: boolean;
  onActivate: () => void;
  onDeactivate: () => void;
  onDelete: () => void;
}) {
  return (
    <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex items-center gap-4">
        <Link href="/promo-codes">
          <Button variant="ghost" size="icon">
            <ArrowLeft className="h-4 w-4" />
          </Button>
        </Link>
        <div className="flex items-center gap-4">
          <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/20">
            <Tag className="h-6 w-6 text-primary" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="font-mono text-2xl font-bold">{promoCode.code}</h1>
              {promoCode.is_active && !isExpired ? (
                <Badge variant="success">Active</Badge>
              ) : isExpired ? (
                <Badge variant="destructive">Expired</Badge>
              ) : (
                <Badge variant="secondary">Inactive</Badge>
              )}
            </div>
            <p className="text-sm text-muted-foreground">{promoCode.name}</p>
          </div>
        </div>
      </div>
      <div className="flex items-center gap-2">
        {promoCode.is_active ? (
          <Button variant="outline" onClick={onDeactivate}>
            <PowerOff className="mr-2 h-4 w-4" />
            Deactivate
          </Button>
        ) : (
          <Button variant="outline" onClick={onActivate}>
            <Power className="mr-2 h-4 w-4" />
            Activate
          </Button>
        )}
        <Button variant="destructive" onClick={onDelete} disabled={isDeleting}>
          <Trash2 className="mr-2 h-4 w-4" />
          Delete
        </Button>
      </div>
    </div>
  );
}
