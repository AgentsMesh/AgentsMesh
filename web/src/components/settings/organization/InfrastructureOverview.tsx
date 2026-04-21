"use client";

import { useRouter } from "next/navigation";
import { useCurrentOrg, useAuthStore } from "@/stores/auth";
import { useTranslations } from "next-intl";
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Server, FolderGit2 } from "lucide-react";

export function InfrastructureOverview() {
  const router = useRouter();
  const currentOrg = useCurrentOrg();
  const t = useTranslations();
  const orgSlug = currentOrg?.slug;

  const go = (sub: string) => {
    if (!orgSlug) return;
    router.push(`/${orgSlug}/settings?scope=organization&tab=infra/${sub}`);
  };

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold tracking-tight">
          {t("settings.org.infrastructure.title")}
        </h2>
        <p className="text-sm text-muted-foreground mt-1">
          {t("settings.org.infrastructure.description")}
        </p>
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
              <Server className="w-5 h-5 text-muted-foreground" />
              <CardTitle>{t("ide.sidebar.settings.tabs.runners")}</CardTitle>
            </div>
            <CardDescription>
              {t("settings.org.infrastructure.runnersDescription")}
            </CardDescription>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground">
            {t("settings.org.infrastructure.runnersHint")}
          </CardContent>
          <CardFooter>
            <Button variant="outline" onClick={() => go("runners")}>
              {t("common.manage")}
            </Button>
          </CardFooter>
        </Card>
        <Card>
          <CardHeader>
            <div className="flex items-center gap-2">
              <FolderGit2 className="w-5 h-5 text-muted-foreground" />
              <CardTitle>{t("ide.sidebar.settings.tabs.repositories")}</CardTitle>
            </div>
            <CardDescription>
              {t("settings.org.infrastructure.repositoriesDescription")}
            </CardDescription>
          </CardHeader>
          <CardContent className="text-sm text-muted-foreground">
            {t("settings.org.infrastructure.repositoriesHint")}
          </CardContent>
          <CardFooter>
            <Button variant="outline" onClick={() => go("repositories")}>
              {t("common.manage")}
            </Button>
          </CardFooter>
        </Card>
      </div>
    </div>
  );
}
