"use client";

import React, { useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import { useCurrentOrg, useAuthOrganizations, useAuthStore } from "@/stores/auth";
import { Spinner } from "@/components/ui/spinner";
import { getDefaultRoute } from "@/lib/default-route";

/**
 * Org-scoped layout. Single owner of the URL → auth-state sync, and
 * gate that blocks child routes from mounting until `currentOrg.slug`
 * matches the URL slug — so every page's mount-time fetch sees the
 * correct slug.
 */
export default function OrgLayout({ children }: { children: React.ReactNode }) {
  const params = useParams();
  const router = useRouter();
  const organizations = useAuthOrganizations();
  const currentOrg = useCurrentOrg();
  const setCurrentOrg = useAuthStore((s) => s.setCurrentOrg);
  const _hasHydrated = useAuthStore((s) => s._hasHydrated);

  const orgSlug = params.org as string;

  useEffect(() => {
    if (!_hasHydrated || !orgSlug) return;

    // Find the organization matching the URL slug
    const targetOrg = organizations.find((org) => org.slug === orgSlug);

    if (targetOrg) {
      if (currentOrg?.slug !== targetOrg.slug) {
        void setCurrentOrg(targetOrg);
      }
    } else if (organizations.length > 0) {
      console.warn(`Organization "${orgSlug}" not found, redirecting...`);
      router.replace(getDefaultRoute(organizations[0].slug));
    }
  }, [orgSlug, organizations, currentOrg, setCurrentOrg, router, _hasHydrated]);

  // Don't render children until we have the correct org context
  if (!_hasHydrated) {
    return null;
  }

  const orgExists = organizations.some((org) => org.slug === orgSlug);
  if (!orgExists && organizations.length > 0) {
    return (
      <div className="flex h-full items-center justify-center">
        <Spinner />
      </div>
    );
  }

  if (orgExists && currentOrg?.slug !== orgSlug) {
    return (
      <div className="flex h-full items-center justify-center">
        <Spinner />
      </div>
    );
  }

  return <>{children}</>;
}
