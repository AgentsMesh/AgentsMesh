"use client";

import Link from "next/link";
import { ArrowLeft, Users, Calendar, Server, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { Organization, OrganizationMember, Runner } from "@/lib/api/admin";
import { formatDate } from "@/lib/utils";
import { SubscriptionSection } from "./subscription-section";
import { MembersSection } from "./members-section";
import { RunnersSection } from "./runners-section";

interface OrgDetailContentProps {
  org: Organization;
  orgId: number;
  members: OrganizationMember[];
  membersLoading: boolean;
  runners: Runner[];
  isDeleting: boolean;
  onDelete: () => void;
}

export function OrgDetailContent({
  org,
  orgId,
  members,
  membersLoading,
  runners,
  isDeleting,
  onDelete,
}: OrgDetailContentProps) {
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-4">
          <Link href="/organizations">
            <Button variant="ghost" size="icon">
              <ArrowLeft className="h-4 w-4" />
            </Button>
          </Link>
          <div className="flex items-center gap-4">
            {org.logo_url ? (
              <img
                src={org.logo_url}
                alt={org.name}
                className="h-12 w-12 rounded-lg object-cover"
              />
            ) : (
              <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-primary/20 text-lg font-medium text-primary">
                {org.name[0].toUpperCase()}
              </div>
            )}
            <div>
              <h1 className="text-2xl font-bold">{org.name}</h1>
              <p className="text-sm text-muted-foreground">{org.slug}</p>
            </div>
          </div>
        </div>
        <Button
          variant="destructive"
          onClick={onDelete}
          disabled={isDeleting}
        >
          <Trash2 className="mr-2 h-4 w-4" />
          Delete Organization
        </Button>
      </div>

      {/* Stats */}
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Members
            </CardTitle>
            <Users className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{members.length}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Runners
            </CardTitle>
            <Server className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{runners.length}</div>
            <p className="text-xs text-muted-foreground">
              {runners.filter((r) => r.status === "online").length} online
            </p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              Created
            </CardTitle>
            <Calendar className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-sm font-medium">{formatDate(org.created_at)}</div>
          </CardContent>
        </Card>
      </div>

      {/* Subscription */}
      <SubscriptionSection orgId={orgId} />

      {/* Members */}
      <MembersSection members={members} isLoading={membersLoading} />

      {/* Runners */}
      <RunnersSection runners={runners} />
    </div>
  );
}
