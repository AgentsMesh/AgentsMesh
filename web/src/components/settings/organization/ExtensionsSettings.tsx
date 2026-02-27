"use client";

import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { SkillRegistriesSettings, McpMarketSettings } from "./extensions";
import type { TranslationFn } from "./GeneralSettings";

interface ExtensionsSettingsProps {
  t: TranslationFn;
}

export function ExtensionsSettings({ t }: ExtensionsSettingsProps) {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold">{t("extensions.settings.title")}</h2>
        <p className="text-sm text-muted-foreground mt-1">
          {t("extensions.settings.description")}
        </p>
      </div>

      <Tabs defaultValue="skill-registries" className="w-full">
        <TabsList>
          <TabsTrigger value="skill-registries">
            {t("extensions.settings.tabs.skillRegistries")}
          </TabsTrigger>
          <TabsTrigger value="mcp-market">
            {t("extensions.settings.tabs.mcpMarket")}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="skill-registries" className="mt-4">
          <SkillRegistriesSettings t={t} />
        </TabsContent>

        <TabsContent value="mcp-market" className="mt-4">
          <McpMarketSettings t={t} />
        </TabsContent>
      </Tabs>
    </div>
  );
}
