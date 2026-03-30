"use client";

import { use, useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { ArrowLeft } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import {
  getOrganization,
  getOrganizationMembers,
  deleteOrganization,
  listRunners,
  Organization,
  OrganizationMember,
  Runner,
} from "@/lib/api/admin";
import type { PaginatedResponse } from "@/lib/api/base";
import { OrgDetailContent } from "./_components/org-detail-content";

export default function OrganizationDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const orgId = parseInt(id, 10);
  const router = useRouter();

  const [org, setOrg] = useState<Organization | null>(null);
  const [orgLoading, setOrgLoading] = useState(true);
  const [membersData, setMembersData] = useState<{ organization: Organization; members: OrganizationMember[] } | null>(null);
  const [membersLoading, setMembersLoading] = useState(true);
  const [runnersData, setRunnersData] = useState<PaginatedResponse<Runner> | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const fetchOrg = useCallback(async () => {
    try {
      const result = await getOrganization(orgId);
      setOrg(result);
    } catch {
      // Keep null on error
    } finally {
      setOrgLoading(false);
    }
  }, [orgId]);

  const fetchMembers = useCallback(async () => {
    try {
      const result = await getOrganizationMembers(orgId);
      setMembersData(result);
    } catch {
      // Keep null on error
    } finally {
      setMembersLoading(false);
    }
  }, [orgId]);

  const fetchRunners = useCallback(async () => {
    try {
      const result = await listRunners({ org_id: orgId, page_size: 100 });
      setRunnersData(result);
    } catch {
      // Keep null on error
    }
  }, [orgId]);

  useEffect(() => {
    fetchOrg();
    fetchMembers();
    fetchRunners();
  }, [fetchOrg, fetchMembers, fetchRunners]);

  const handleDelete = async () => {
    if (
      !org ||
      !confirm(
        `Are you sure you want to delete "${org.name}"? This action cannot be undone.`
      )
    ) {
      return;
    }
    setIsDeleting(true);
    try {
      await deleteOrganization(orgId);
      toast.success("Organization deleted successfully");
      router.push("/organizations");
    } catch (err: unknown) {
      const message = (err as { error?: string })?.error || "Failed to delete organization";
      toast.error(message);
    } finally {
      setIsDeleting(false);
    }
  };

  if (orgLoading) {
    return (
      <div className="space-y-6">
        <div className="h-8 w-48 animate-pulse rounded bg-muted" />
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 3 }).map((_, i) => (
            <Card key={i} className="animate-pulse">
              <CardHeader className="pb-2">
                <div className="h-4 w-24 rounded bg-muted" />
              </CardHeader>
              <CardContent>
                <div className="h-8 w-16 rounded bg-muted" />
              </CardContent>
            </Card>
          ))}
        </div>
      </div>
    );
  }

  if (!org) {
    return (
      <div className="flex h-64 flex-col items-center justify-center gap-4">
        <p className="text-muted-foreground">Organization not found</p>
        <Link href="/organizations">
          <Button variant="outline">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Organizations
          </Button>
        </Link>
      </div>
    );
  }

  return (
    <OrgDetailContent
      org={org}
      orgId={orgId}
      members={membersData?.members || []}
      membersLoading={membersLoading}
      runners={runnersData?.data || []}
      isDeleting={isDeleting}
      onDelete={handleDelete}
    />
  );
}
