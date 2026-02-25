"use client";

import { useEffect } from "react";
import { useParams, useRouter } from "next/navigation";

/**
 * Organization root page - redirects to workspace
 */
export default function OrganizationPage() {
  const router = useRouter();
  const params = useParams();
  const orgSlug = params.org as string;

  useEffect(() => {
    router.replace(`/${orgSlug}/workspace`);
  }, [orgSlug, router]);

  return null;
}
