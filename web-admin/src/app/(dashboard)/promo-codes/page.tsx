"use client";

import { useState, useEffect, useCallback } from "react";
import Link from "next/link";
import { Search, Plus, ChevronLeft, ChevronRight } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  listPromoCodes,
  activatePromoCode,
  deactivatePromoCode,
  deletePromoCode,
  PromoCode,
  PromoCodeType,
} from "@/lib/api/admin";
import type { PaginatedResponse } from "@/lib/api/base";
import { PromoCodesTable } from "./promo-codes-table";

export default function PromoCodesPage() {
  const [search, setSearch] = useState("");
  const [typeFilter, setTypeFilter] = useState<string>("all");
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [page, setPage] = useState(1);
  const pageSize = 20;

  const [data, setData] = useState<PaginatedResponse<PromoCode> | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  const fetchPromoCodes = useCallback(async () => {
    setIsLoading(true);
    try {
      const result = await listPromoCodes({
        search: search || undefined,
        type: typeFilter !== "all" ? (typeFilter as PromoCodeType) : undefined,
        is_active: statusFilter === "all" ? undefined : statusFilter === "active",
        page,
        page_size: pageSize,
      });
      setData(result);
    } catch {
      // Keep previous data on error
    } finally {
      setIsLoading(false);
    }
  }, [search, typeFilter, statusFilter, page, pageSize]);

  useEffect(() => {
    fetchPromoCodes();
  }, [fetchPromoCodes]);

  const handleActivate = async (id: number) => {
    try {
      await activatePromoCode(id);
      toast.success("Promo code activated");
      await fetchPromoCodes();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to activate promo code");
    }
  };

  const handleDeactivate = async (id: number) => {
    try {
      await deactivatePromoCode(id);
      toast.success("Promo code deactivated");
      await fetchPromoCodes();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to deactivate promo code");
    }
  };

  const handleDelete = async (code: PromoCode) => {
    if (!confirm(`Are you sure you want to delete "${code.code}"? This action cannot be undone.`)) return;
    try {
      await deletePromoCode(code.id);
      toast.success("Promo code deleted");
      await fetchPromoCodes();
    } catch (err: unknown) {
      toast.error((err as { error?: string })?.error || "Failed to delete promo code");
    }
  };

  const total = data?.total || 0;
  const totalPages = data?.total_pages || 1;

  return (
    <div className="space-y-6">
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold">Promo Codes</h1>
          <p className="text-sm text-muted-foreground">
            Manage promotional codes for subscriptions
          </p>
        </div>
        <Link href="/promo-codes/new">
          <Button>
            <Plus className="mr-2 h-4 w-4" />
            Create Promo Code
          </Button>
        </Link>
      </div>

      <div className="flex flex-col gap-4 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            placeholder="Search by code or name..."
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
            className="pl-10"
          />
        </div>
        <Select value={typeFilter} onValueChange={(v) => { setTypeFilter(v); setPage(1); }}>
          <SelectTrigger className="w-40">
            <SelectValue placeholder="All Types" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Types</SelectItem>
            <SelectItem value="media">Media</SelectItem>
            <SelectItem value="partner">Partner</SelectItem>
            <SelectItem value="campaign">Campaign</SelectItem>
            <SelectItem value="internal">Internal</SelectItem>
            <SelectItem value="referral">Referral</SelectItem>
          </SelectContent>
        </Select>
        <Select value={statusFilter} onValueChange={(v) => { setStatusFilter(v); setPage(1); }}>
          <SelectTrigger className="w-40">
            <SelectValue placeholder="All Status" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All Status</SelectItem>
            <SelectItem value="active">Active</SelectItem>
            <SelectItem value="inactive">Inactive</SelectItem>
          </SelectContent>
        </Select>
      </div>

      <PromoCodesTable
        promoCodes={data?.data || []}
        isLoading={isLoading}
        onActivate={handleActivate}
        onDeactivate={handleDeactivate}
        onDelete={handleDelete}
      />

      {totalPages > 1 && (
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <p className="text-sm text-muted-foreground">
            Showing {(page - 1) * pageSize + 1} to{" "}
            {Math.min(page * pageSize, total)} of {total} promo codes
          </p>
          <div className="flex items-center gap-2">
            <Button variant="outline" size="icon" onClick={() => setPage(page - 1)} disabled={page <= 1}>
              <ChevronLeft className="h-4 w-4" />
            </Button>
            <span className="text-sm">Page {page} of {totalPages}</span>
            <Button variant="outline" size="icon" onClick={() => setPage(page + 1)} disabled={page >= totalPages}>
              <ChevronRight className="h-4 w-4" />
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
