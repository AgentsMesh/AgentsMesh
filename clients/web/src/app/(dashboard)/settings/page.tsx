"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { CenteredSpinner } from "@/components/ui/spinner";

export default function PersonalSettingsPage() {
  const router = useRouter();

  useEffect(() => {
    router.replace("/settings/general");
  }, [router]);

  return <CenteredSpinner />;
}
