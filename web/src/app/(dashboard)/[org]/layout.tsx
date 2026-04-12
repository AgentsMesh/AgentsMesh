"use client";

import React, { useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import { useAuthStore } from "@/stores/auth";
import { Spinner } from "@/components/ui/spinner";
import { getDefaultRoute } from "@/lib/default-route";

/**
 * Organization-scoped layout
 * Ensures the organization from URL matches the current organization in state
 */
export default function OrgLayout({ children }: { children: React.ReactNode }) {
  const params = useParams();
  const router = useRouter();
  const { organizations, currentOrg, setCurrentOrg, _hasHydrated } = useAuthStore();

  const orgSlug = params.org as string;

  useEffect(() => {
    if (!_hasHydrated || !orgSlug) return;

    // Find the organization matching the URL slug
    const targetOrg = organizations.find((org) => org.slug === orgSlug);

    if (targetOrg) {
      // If found and different from current, update currentOrg
      if (currentOrg?.slug !== targetOrg.slug) {
        setCurrentOrg(targetOrg);
      }
    } else if (organizations.length > 0) {
      // Organization not found, redirect to first available org
      console.warn(`Organization "${orgSlug}" not found, redirecting...`);
      router.replace(getDefaultRoute(organizations[0].slug));
    }
  }, [orgSlug, organizations, currentOrg, setCurrentOrg, router, _hasHydrated]);

  // Don't render children until we have the correct org context
  if (!_hasHydrated) {
    return null;
  }

  // If org from URL doesn't match any known org and we have orgs, show loading
  const orgExists = organizations.some((org) => org.slug === orgSlug);
  if (!orgExists && organizations.length > 0) {
    return (
      <div className="flex h-full items-center justify-center">
        <Spinner />
      </div>
    );
  }

  return <>{children}</>;
}
