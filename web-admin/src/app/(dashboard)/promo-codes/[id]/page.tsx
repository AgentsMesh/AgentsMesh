"use client";

import { use, useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  getPromoCode,
  activatePromoCode,
  deactivatePromoCode,
  deletePromoCode,
  listPromoCodeRedemptions,
  PromoCode,
  PromoCodeRedemption,
} from "@/lib/api/admin";
import type { PaginatedResponse } from "@/lib/api/base";
import { PromoCodeHeader } from "./_components/promo-code-header";
import { PromoCodeStats } from "./_components/promo-code-stats";
import { RedemptionsTable } from "./_components/redemptions-table";

export default function PromoCodeDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const promoCodeId = parseInt(id, 10);
  const router = useRouter();

  const [promoCode, setPromoCode] = useState<PromoCode | null>(null);
  const [codeLoading, setCodeLoading] = useState(true);
  const [redemptionsData, setRedemptionsData] = useState<PaginatedResponse<PromoCodeRedemption> | null>(null);
  const [redemptionsLoading, setRedemptionsLoading] = useState(true);
  const [isDeleting, setIsDeleting] = useState(false);

  const fetchPromoCode = useCallback(async () => {
    try {
      const result = await getPromoCode(promoCodeId);
      setPromoCode(result);
    } catch {
      // Keep null on error
    } finally {
      setCodeLoading(false);
    }
  }, [promoCodeId]);

  const fetchRedemptions = useCallback(async () => {
    try {
      const result = await listPromoCodeRedemptions(promoCodeId, { page_size: 50 });
      setRedemptionsData(result);
    } catch {
      // Keep null on error
    } finally {
      setRedemptionsLoading(false);
    }
  }, [promoCodeId]);

  useEffect(() => { fetchPromoCode(); }, [fetchPromoCode]);
  useEffect(() => { if (promoCode) fetchRedemptions(); }, [promoCode, fetchRedemptions]);

  const handleActivate = async () => {
    try {
      await activatePromoCode(promoCodeId);
      toast.success("Promo code activated");
      await fetchPromoCode();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to activate promo code");
    }
  };

  const handleDeactivate = async () => {
    try {
      await deactivatePromoCode(promoCodeId);
      toast.success("Promo code deactivated");
      await fetchPromoCode();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to deactivate promo code");
    }
  };

  const handleDelete = async () => {
    if (!promoCode || !confirm(`Are you sure you want to delete "${promoCode.code}"? This action cannot be undone.`)) return;
    setIsDeleting(true);
    try {
      await deletePromoCode(promoCodeId);
      toast.success("Promo code deleted");
      router.push("/promo-codes");
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to delete promo code");
    } finally {
      setIsDeleting(false);
    }
  };

  if (codeLoading) {
    return (
      <div className="space-y-6">
        <div className="h-8 w-48 animate-pulse rounded bg-muted" />
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Card key={i} className="animate-pulse">
              <CardHeader className="pb-2"><div className="h-4 w-24 rounded bg-muted" /></CardHeader>
              <CardContent><div className="h-8 w-16 rounded bg-muted" /></CardContent>
            </Card>
          ))}
        </div>
      </div>
    );
  }

  if (!promoCode) {
    return (
      <div className="flex h-64 flex-col items-center justify-center gap-4">
        <p className="text-muted-foreground">Promo code not found</p>
        <Link href="/promo-codes">
          <Button variant="outline"><ArrowLeft className="mr-2 h-4 w-4" />Back to Promo Codes</Button>
        </Link>
      </div>
    );
  }

  const redemptions = redemptionsData?.data || [];
  const isExpired = !!promoCode.expires_at && new Date(promoCode.expires_at) < new Date();

  return (
    <div className="space-y-6">
      <PromoCodeHeader
        promoCode={promoCode}
        isExpired={isExpired}
        isDeleting={isDeleting}
        onActivate={handleActivate}
        onDeactivate={handleDeactivate}
        onDelete={handleDelete}
      />
      <PromoCodeStats promoCode={promoCode} isExpired={isExpired} />

      {promoCode.description && (
        <Card>
          <CardHeader><CardTitle>Description</CardTitle></CardHeader>
          <CardContent><p className="text-muted-foreground">{promoCode.description}</p></CardContent>
        </Card>
      )}

      <Card>
        <CardHeader><CardTitle>Usage Limits</CardTitle></CardHeader>
        <CardContent>
          <div className="grid gap-4 sm:grid-cols-2">
            <div>
              <p className="text-sm text-muted-foreground">Max Total Uses</p>
              <p className="font-medium">{promoCode.max_uses === null ? "Unlimited" : promoCode.max_uses}</p>
            </div>
            <div>
              <p className="text-sm text-muted-foreground">Max Uses per Organization</p>
              <p className="font-medium">{promoCode.max_uses_per_org}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      <RedemptionsTable redemptions={redemptions} isLoading={redemptionsLoading} />
    </div>
  );
}
