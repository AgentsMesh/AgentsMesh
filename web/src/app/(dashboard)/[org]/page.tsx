"use client";

import { useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import { getDefaultRoute } from "@/lib/default-route";

export default function OrganizationPage() {
  const router = useRouter();
  const params = useParams();
  const orgSlug = params.org as string;

  useEffect(() => {
    router.replace(getDefaultRoute(orgSlug));
  }, [orgSlug, router]);

  return null;
}
