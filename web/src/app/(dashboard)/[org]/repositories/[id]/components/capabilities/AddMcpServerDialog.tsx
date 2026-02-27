"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { extensionApi, McpMarketItem } from "@/lib/api";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogBody, DialogFooter } from "@/components/ui/dialog";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { ExternalLink, Search } from "lucide-react";

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
    setMarketQuery("");
    setSelectedTemplate(null);
    setEnvVars({});
    setCustomName("");
    setCustomSlug("");
    setCustomTransport("stdio");
    setCustomCommand("");
    setCustomArgs("");
    setCustomHttpUrl("");
    setCustomEnvVars([]);
  }, []);

  const loadMarketServers = useCallback(async (query?: string) => {
    setLoadingMarket(true);
    try {
      const res = await extensionApi.listMarketMcpServers(query, undefined, 100, 0);
      setMarketServers(res.mcp_servers || []);
    } catch (error) {
      console.error("Failed to load market MCP servers:", error);
    } finally {
      setLoadingMarket(false);
    }
  }, []);

  useEffect(() => {
    if (open) {
      loadMarketServers();
    }
  }, [open, loadMarketServers]);

  const handleSearchMarket = useCallback(() => {
    loadMarketServers(marketQuery || undefined);
  }, [marketQuery, loadMarketServers]);

  const handleSelectTemplate = useCallback((item: McpMarketItem) => {
    setSelectedTemplate(item);
    // Pre-fill env vars from schema
    const defaults: Record<string, string> = {};
    item.env_var_schema?.forEach((entry) => {
      defaults[entry.name] = "";
    });
    setEnvVars(defaults);
  }, []);

  // Check whether all required env vars have been filled
  const hasUnfilledRequiredEnvVars = selectedTemplate?.env_var_schema?.some(
    (entry) => entry.required && !envVars[entry.name]?.trim()
  ) ?? false;

  const handleInstallFromMarket = useCallback(async () => {
    if (!selectedTemplate) return;
    setInstalling(true);
    try {
      // Filter out empty env var values
      const filteredEnvVars: Record<string, string> = {};
      Object.entries(envVars).forEach(([key, value]) => {
        if (value.trim()) {
          filteredEnvVars[key] = value.trim();
        }
      });

      await extensionApi.installMcpFromMarket(repositoryId, {
        market_item_id: selectedTemplate.id,
        env_vars: Object.keys(filteredEnvVars).length > 0 ? filteredEnvVars : undefined,
        scope,
      });
      toast.success(t("extensions.installed"));
      onInstalled();
    } catch (error) {
      toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToInstall")));
    } finally {
      setInstalling(false);
    }
  }, [repositoryId, selectedTemplate, envVars, scope, t, onInstalled]);

  const handleInstallCustom = useCallback(async () => {
    if (!customName.trim() || !customSlug.trim()) return;
    setInstalling(true);
    try {
      // Convert array to Record, filtering out empty keys
      const filteredEnvVars: Record<string, string> = Object.fromEntries(
        customEnvVars
          .filter((e) => e.key.trim())
          .map((e) => [e.key.trim(), e.value.trim()])
      );

      await extensionApi.installCustomMcpServer(repositoryId, {
        name: customName.trim(),
        slug: customSlug.trim(),
        transport_type: customTransport,
        command: customTransport === "stdio" ? customCommand.trim() || undefined : undefined,
        args: customTransport === "stdio" && customArgs.trim()
          ? customArgs.split(/\s+/).filter(Boolean)
          : undefined,
        http_url: customTransport !== "stdio" ? customHttpUrl.trim() || undefined : undefined,
        env_vars: Object.keys(filteredEnvVars).length > 0 ? filteredEnvVars : undefined,
        scope,
      });
      toast.success(t("extensions.installed"));
      onInstalled();
    } catch (error) {
      toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToInstall")));
    } finally {
      setInstalling(false);
    }
  }, [repositoryId, customName, customSlug, customTransport, customCommand, customArgs, customHttpUrl, customEnvVars, scope, t, onInstalled]);

  return (
    <Dialog open={open} onOpenChange={(value) => { if (!value) resetAllState(); onOpenChange(value); }}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t("extensions.addMcpServer")}</DialogTitle>
        </DialogHeader>
        <DialogBody>
          <Tabs defaultValue="market">
            <TabsList className="mb-4">
              <TabsTrigger value="market">{t("extensions.marketTemplates")}</TabsTrigger>
              <TabsTrigger value="custom">{t("extensions.custom")}</TabsTrigger>
            </TabsList>

            {/* Market templates tab */}
            <TabsContent value="market">
              {selectedTemplate ? (
                // Env var configuration form
                <div className="space-y-4">
                  <div className="flex items-center gap-2 mb-2">
                    <span className="font-medium">{selectedTemplate.name}</span>
                    <Badge variant="secondary" className="text-xs">{selectedTemplate.transport_type}</Badge>
                    {selectedTemplate.repository_url && (
                      <a
                        href={selectedTemplate.repository_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-muted-foreground hover:text-foreground"
                        title={t("extensions.viewSource")}
                      >
                        <ExternalLink className="w-3.5 h-3.5" />
                      </a>
                    )}
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setSelectedTemplate(null)}
                    >
                      {t("extensions.changeTemplate")}
                    </Button>
                  </div>
                  {selectedTemplate.description && (
                    <p className="text-sm text-muted-foreground">{selectedTemplate.description}</p>
                  )}
                  {(selectedTemplate.env_var_schema?.length ?? 0) > 0 && (
                    <div className="space-y-3">
                      <h4 className="text-sm font-medium">{t("extensions.envVars")}</h4>
                      {selectedTemplate.env_var_schema!.map((entry) => (
                        <div key={entry.name}>
                          <label className="text-sm font-medium mb-1 block">
                            {entry.label || entry.name}
                            {entry.required && <span className="text-destructive ml-1">*</span>}
                          </label>
                          <Input
                            type={entry.sensitive ? "password" : "text"}
                            placeholder={entry.placeholder || entry.name}
                            value={envVars[entry.name] || ""}
                            onChange={(e) =>
                              setEnvVars((prev) => ({ ...prev, [entry.name]: e.target.value }))
                            }
                          />
                        </div>
                      ))}
                    </div>
                  )}
                  <Button
                    className="w-full"
                    disabled={installing || hasUnfilledRequiredEnvVars}
                    onClick={handleInstallFromMarket}
                  >
                    {installing ? t("extensions.installing") : t("extensions.install")}
                  </Button>
                </div>
              ) : (
                // Market listing
                <>
                  <div className="flex gap-2 mb-4">
                    <div className="relative flex-1">
                      <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                      <Input
                        className="pl-9"
                        placeholder={t("extensions.searchMcpServers")}
                        value={marketQuery}
                        onChange={(e) => setMarketQuery(e.target.value)}
                        onKeyDown={(e) => e.key === "Enter" && handleSearchMarket()}
                      />
                    </div>
                    <Button variant="outline" onClick={handleSearchMarket}>
                      {t("extensions.search")}
                    </Button>
                  </div>
                  {loadingMarket ? (
                    <div className="py-8 text-center">
                      <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary mx-auto"></div>
                    </div>
                  ) : marketServers.length === 0 ? (
                    <p className="text-sm text-muted-foreground text-center py-8">
                      {t("extensions.noMarketMcpServers")}
                    </p>
                  ) : (
                    <div className="space-y-2 max-h-80 overflow-y-auto">
                      {marketServers.map((item) => (
                        <div
                          key={item.id}
                          className="border border-border rounded-lg p-3 flex items-center justify-between gap-3 cursor-pointer hover:bg-muted/50"
                          onClick={() => handleSelectTemplate(item)}
                        >
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <span className="font-medium">{item.name}</span>
                              <Badge variant="secondary" className="text-xs">{item.transport_type}</Badge>
                              {item.category && (
                                <Badge variant="outline" className="text-xs">{item.category}</Badge>
                              )}
                              {item.repository_url && (
                                <a
                                  href={item.repository_url}
                                  target="_blank"
                                  rel="noopener noreferrer"
                                  className="text-muted-foreground hover:text-foreground shrink-0"
                                  title={t("extensions.viewSource")}
                                  onClick={(e) => e.stopPropagation()}
                                >
                                  <ExternalLink className="w-3.5 h-3.5" />
                                </a>
                              )}
                            </div>
                            {item.description && (
                              <p className="text-xs text-muted-foreground mt-1 line-clamp-2">
                                {item.description}
                              </p>
                            )}
                          </div>
                          <Button size="sm" variant="outline">
                            {t("extensions.select")}
                          </Button>
                        </div>
                      ))}
                    </div>
                  )}
                </>
              )}
            </TabsContent>

            {/* Custom tab */}
            <TabsContent value="custom">
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="text-sm font-medium mb-1 block">
                      {t("extensions.serverName")} <span className="text-destructive">*</span>
                    </label>
                    <Input
                      placeholder={t("extensions.serverNamePlaceholder")}
                      value={customName}
                      onChange={(e) => setCustomName(e.target.value)}
                    />
                  </div>
                  <div>
                    <label className="text-sm font-medium mb-1 block">
                      {t("extensions.slug")} <span className="text-destructive">*</span>
                    </label>
                    <Input
                      placeholder={t("extensions.slugPlaceholder")}
                      value={customSlug}
                      onChange={(e) => setCustomSlug(e.target.value)}
                    />
                  </div>
                </div>

                <div>
                  <label className="text-sm font-medium mb-1 block">
                    {t("extensions.transportType")}
                  </label>
                  <div className="flex gap-2">
                    {["stdio", "sse", "http"].map((tp) => (
                      <Button
                        key={tp}
                        variant={customTransport === tp ? "default" : "outline"}
                        size="sm"
                        onClick={() => setCustomTransport(tp)}
                      >
                        {tp}
                      </Button>
                    ))}
                  </div>
                </div>

                {customTransport === "stdio" ? (
                  <>
                    <div>
                      <label className="text-sm font-medium mb-1 block">
                        {t("extensions.command")}
                      </label>
                      <Input
                        placeholder="npx"
                        value={customCommand}
                        onChange={(e) => setCustomCommand(e.target.value)}
                      />
                    </div>
                    <div>
                      <label className="text-sm font-medium mb-1 block">
                        {t("extensions.args")}
                      </label>
                      <Input
                        placeholder="-y @modelcontextprotocol/server-filesystem /path"
                        value={customArgs}
                        onChange={(e) => setCustomArgs(e.target.value)}
                      />
                      <p className="text-xs text-muted-foreground mt-1">
                        {t("extensions.argsHint")}
                      </p>
                    </div>
                  </>
                ) : (
                  <div>
                    <label className="text-sm font-medium mb-1 block">
                      {t("extensions.httpUrl")}
                    </label>
                    <Input
                      placeholder="http://localhost:3001/mcp"
                      value={customHttpUrl}
                      onChange={(e) => setCustomHttpUrl(e.target.value)}
                    />
                  </div>
                )}

                <div>
                  <label className="text-sm font-medium mb-2 block">
                    {t("extensions.envVars")}
                  </label>
                  {customEnvVars.map((entry, idx) => (
                    <div key={idx} className="flex gap-2 mb-2">
                      <Input
                        placeholder="KEY"
                        value={entry.key}
                        onChange={(e) => {
                          setCustomEnvVars((prev) =>
                            prev.map((item, i) => (i === idx ? { ...item, key: e.target.value } : item))
                          );
                        }}
                      />
                      <Input
                        placeholder="value"
                        value={entry.value}
                        onChange={(e) => {
                          setCustomEnvVars((prev) =>
                            prev.map((item, i) => (i === idx ? { ...item, value: e.target.value } : item))
                          );
                        }}
                      />
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          setCustomEnvVars((prev) => prev.filter((_, i) => i !== idx));
                        }}
                        className="text-destructive"
                      >
                        x
                      </Button>
                    </div>
                  ))}
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setCustomEnvVars((prev) => [...prev, { key: "", value: "" }])}
                  >
                    {t("extensions.addEnvVar")}
                  </Button>
                </div>

                <Button
                  className="w-full"
                  disabled={installing || !customName.trim() || !customSlug.trim()}
                  onClick={handleInstallCustom}
                >
                  {installing ? t("extensions.installing") : t("extensions.install")}
                </Button>
              </div>
            </TabsContent>
          </Tabs>
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
