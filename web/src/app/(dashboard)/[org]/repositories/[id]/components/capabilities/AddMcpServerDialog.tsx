"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { extensionApi, McpMarketItem } from "@/lib/api";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogBody, DialogFooter } from "@/components/ui/dialog";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { MarketTab } from "./MarketTab";
import { CustomTab } from "./CustomTab";

interface AddMcpServerDialogProps {
  repositoryId: number;
  scope: "org" | "user";
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onInstalled: () => void;
}

export function AddMcpServerDialog({ repositoryId, scope, open, onOpenChange, onInstalled }: AddMcpServerDialogProps) {
  const t = useTranslations();
  const [installing, setInstalling] = useState(false);

  // Market state
  const [marketServers, setMarketServers] = useState<McpMarketItem[]>([]);
  const [marketQuery, setMarketQuery] = useState("");
  const [loadingMarket, setLoadingMarket] = useState(false);
  const [selectedTemplate, setSelectedTemplate] = useState<McpMarketItem | null>(null);
  const [envVars, setEnvVars] = useState<Record<string, string>>({});

  // Custom form state
  const [customName, setCustomName] = useState("");
  const [customSlug, setCustomSlug] = useState("");
  const [customTransport, setCustomTransport] = useState("stdio");
  const [customCommand, setCustomCommand] = useState("");
  const [customArgs, setCustomArgs] = useState("");
  const [customHttpUrl, setCustomHttpUrl] = useState("");
  const [customEnvVars, setCustomEnvVars] = useState<Array<{key: string; value: string}>>([]);

  const resetAllState = useCallback(() => {
    setMarketQuery(""); setSelectedTemplate(null); setEnvVars({});
    setCustomName(""); setCustomSlug(""); setCustomTransport("stdio");
    setCustomCommand(""); setCustomArgs(""); setCustomHttpUrl(""); setCustomEnvVars([]);
  }, []);

  const loadMarketServers = useCallback(async (query?: string) => {
    setLoadingMarket(true);
    try {
      const res = await extensionApi.listMarketMcpServers(query, undefined, 100, 0);
      setMarketServers(res.mcp_servers || []);
    } catch (error) { console.error("Failed to load market MCP servers:", error); }
    finally { setLoadingMarket(false); }
  }, []);

  useEffect(() => { if (open) loadMarketServers(); }, [open, loadMarketServers]);

  const handleSearchMarket = useCallback(() => {
    loadMarketServers(marketQuery || undefined);
  }, [marketQuery, loadMarketServers]);

  const handleSelectTemplate = useCallback((item: McpMarketItem) => {
    setSelectedTemplate(item);
    const defaults: Record<string, string> = {};
    item.env_var_schema?.forEach((entry) => { defaults[entry.name] = ""; });
    setEnvVars(defaults);
  }, []);

  const hasUnfilledRequiredEnvVars = selectedTemplate?.env_var_schema?.some(
    (entry) => entry.required && !envVars[entry.name]?.trim()
  ) ?? false;

  const handleInstallFromMarket = useCallback(async () => {
    if (!selectedTemplate) return;
    setInstalling(true);
    try {
      const filteredEnvVars: Record<string, string> = {};
      Object.entries(envVars).forEach(([key, value]) => { if (value.trim()) filteredEnvVars[key] = value.trim(); });
      await extensionApi.installMcpFromMarket(repositoryId, {
        market_item_id: selectedTemplate.id,
        env_vars: Object.keys(filteredEnvVars).length > 0 ? filteredEnvVars : undefined,
        scope,
      });
      toast.success(t("extensions.installed"));
      onInstalled();
    } catch (error) { toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToInstall"))); }
    finally { setInstalling(false); }
  }, [repositoryId, selectedTemplate, envVars, scope, t, onInstalled]);

  const handleInstallCustom = useCallback(async () => {
    if (!customName.trim() || !customSlug.trim()) return;
    setInstalling(true);
    try {
      const filteredEnvVars: Record<string, string> = Object.fromEntries(
        customEnvVars.filter((e) => e.key.trim()).map((e) => [e.key.trim(), e.value.trim()])
      );
      await extensionApi.installCustomMcpServer(repositoryId, {
        name: customName.trim(), slug: customSlug.trim(), transport_type: customTransport,
        command: customTransport === "stdio" ? customCommand.trim() || undefined : undefined,
        args: customTransport === "stdio" && customArgs.trim() ? customArgs.split(/\s+/).filter(Boolean) : undefined,
        http_url: customTransport !== "stdio" ? customHttpUrl.trim() || undefined : undefined,
        env_vars: Object.keys(filteredEnvVars).length > 0 ? filteredEnvVars : undefined,
        scope,
      });
      toast.success(t("extensions.installed"));
      onInstalled();
    } catch (error) { toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToInstall"))); }
    finally { setInstalling(false); }
  }, [repositoryId, customName, customSlug, customTransport, customCommand, customArgs, customHttpUrl, customEnvVars, scope, t, onInstalled]);

  return (
    <Dialog open={open} onOpenChange={(value) => { if (!value) resetAllState(); onOpenChange(value); }}>
      <DialogContent className="max-w-2xl">
        <DialogHeader><DialogTitle>{t("extensions.addMcpServer")}</DialogTitle></DialogHeader>
        <DialogBody>
          <Tabs defaultValue="market">
            <TabsList className="mb-4">
              <TabsTrigger value="market">{t("extensions.marketTemplates")}</TabsTrigger>
              <TabsTrigger value="custom">{t("extensions.custom")}</TabsTrigger>
            </TabsList>
            <TabsContent value="market">
              <MarketTab marketServers={marketServers} marketQuery={marketQuery}
                setMarketQuery={setMarketQuery} loadingMarket={loadingMarket}
                selectedTemplate={selectedTemplate} envVars={envVars} setEnvVars={setEnvVars}
                hasUnfilledRequiredEnvVars={hasUnfilledRequiredEnvVars} installing={installing}
                t={t} onSearch={handleSearchMarket} onSelectTemplate={handleSelectTemplate}
                onClearTemplate={() => setSelectedTemplate(null)} onInstall={handleInstallFromMarket} />
            </TabsContent>
            <TabsContent value="custom">
              <CustomTab customName={customName} setCustomName={setCustomName}
                customSlug={customSlug} setCustomSlug={setCustomSlug}
                customTransport={customTransport} setCustomTransport={setCustomTransport}
                customCommand={customCommand} setCustomCommand={setCustomCommand}
                customArgs={customArgs} setCustomArgs={setCustomArgs}
                customHttpUrl={customHttpUrl} setCustomHttpUrl={setCustomHttpUrl}
                customEnvVars={customEnvVars} setCustomEnvVars={setCustomEnvVars}
                installing={installing} t={t} onInstall={handleInstallCustom} />
            </TabsContent>
          </Tabs>
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>{t("common.cancel")}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
