"use client";

import { useEffect } from "react";
import { useParams, useRouter } from "next/navigation";

export default function OrganizationPage() {
  const router = useRouter();
  const params = useParams();
  const orgSlug = params.org as string;

  useEffect(() => {
    const isMobile = window.innerWidth < 768;
    const target = isMobile ? "channels" : "workspace";
    router.replace(`/${orgSlug}/${target}`);
  }, [orgSlug, router]);

  return null;
}
