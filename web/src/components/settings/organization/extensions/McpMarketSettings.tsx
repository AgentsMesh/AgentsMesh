"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { ExternalLink, Search, Loader2 } from "lucide-react";
import { extensionApi, McpMarketItem } from "@/lib/api";
import type { TranslationFn } from "../GeneralSettings";

const PAGE_SIZE = 50;

interface McpMarketSettingsProps {
  t: TranslationFn;
}

export function McpMarketSettings({ t }: McpMarketSettingsProps) {
  const [servers, setServers] = useState<McpMarketItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [search, setSearch] = useState("");
  const offsetRef = useRef(0);

  const loadServers = useCallback(async (query?: string, append = false, mounted?: { current: boolean }) => {
    try {
      if (append) {
        setLoadingMore(true);
      } else {
        setLoading(true);
        offsetRef.current = 0;
      }
      const res = await extensionApi.listMarketMcpServers(
        query,
        undefined,
        PAGE_SIZE,
        offsetRef.current
      );
      if (mounted && !mounted.current) return;
      const items = res.mcp_servers || [];
      if (append) {
        setServers((prev) => [...prev, ...items]);
      } else {
        setServers(items);
      }
      setTotal(res.total);
      offsetRef.current += items.length;
    } catch (error) {
      if (mounted && !mounted.current) return;
      console.error("Failed to load MCP market servers:", error);
    } finally {
      if (!mounted || mounted.current) {
        setLoading(false);
        setLoadingMore(false);
      }
    }
  }, []);

  // Debounced search – also handles initial load (search starts as "")
  useEffect(() => {
    const mounted = { current: true };
    const timer = setTimeout(() => {
      loadServers(search || undefined, false, mounted);
    }, 300);
    return () => {
      mounted.current = false;
      clearTimeout(timer);
    };
  }, [search, loadServers]);

  const handleLoadMore = () => {
    loadServers(search || undefined, true);
  };

  const hasMore = servers.length < total;

  const sourceLabel = (source?: string) => {
    switch (source) {
      case "registry": return t("extensions.mcpMarket.sourceRegistry");
      case "seed": return t("extensions.mcpMarket.sourceBuiltIn");
      case "admin": return t("extensions.mcpMarket.sourceAdmin");
      default: return source || t("extensions.mcpMarket.sourceBuiltIn");
    }
  };

  return (
    <div className="space-y-6">
      <div className="border border-border rounded-lg p-6">
        <div className="mb-4">
          <h2 className="text-lg font-semibold">{t("extensions.mcpMarket.title")}</h2>
          <p className="text-sm text-muted-foreground">
            {t("extensions.mcpMarket.description")}
          </p>
        </div>

        <div className="relative mb-4">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            className="pl-9"
            placeholder={t("extensions.searchMcpServers")}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>

        {loading ? (
          <div className="flex items-center justify-center py-8 text-muted-foreground gap-2">
            <Loader2 className="h-4 w-4 animate-spin" />
            {t("extensions.loading")}
          </div>
        ) : servers.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            {t("extensions.mcpMarket.noServers")}
          </div>
        ) : (
          <>
            <p className="text-xs text-muted-foreground mb-3">
              {total} {t("extensions.mcpMarket.serversFound")}
            </p>
            <div className="space-y-3">
              {servers.map((server) => (
                <div
                  key={server.id}
                  className="border border-border rounded-lg p-4"
                >
                  <div className="flex items-center gap-2 mb-2 flex-wrap">
                    <span className="font-medium">{server.name}</span>
                    <Badge variant="secondary" className="text-xs">
                      {server.transport_type}
                    </Badge>
                    {server.category && (
                      <Badge variant="outline" className="text-xs">
                        {server.category}
                      </Badge>
                    )}
                    <Badge
                      variant={server.source === "registry" ? "default" : "secondary"}
                      className="text-xs"
                    >
                      {sourceLabel(server.source)}
                    </Badge>
                    {server.version && (
                      <span className="text-xs text-muted-foreground">v{server.version}</span>
                    )}
                  </div>
                  {server.description && (
                    <p className="text-sm text-muted-foreground mb-2">{server.description}</p>
                  )}
                  {server.command && (
                    <p className="text-xs font-mono text-muted-foreground mb-2">
                      {server.command}{server.default_args?.length ? " " + server.default_args.join(" ") : ""}
                    </p>
                  )}
                  {server.default_http_url && (
                    <p className="text-xs font-mono text-muted-foreground mb-2">
                      {server.default_http_url}
                    </p>
                  )}
                  <div className="flex items-center gap-3 flex-wrap">
                    {(server.env_var_schema?.length ?? 0) > 0 && (
                      <div>
                        <span className="text-xs font-medium text-muted-foreground">
                          {t("extensions.envVars")}:
                        </span>
                        <div className="flex flex-wrap gap-1 mt-1">
                          {server.env_var_schema!.map((entry) => (
                            <Badge
                              key={entry.name}
                              variant={entry.required ? "default" : "outline"}
                              className="text-xs"
                            >
                              {entry.name}
                              {entry.required && " *"}
                            </Badge>
                          ))}
                        </div>
                      </div>
                    )}
                    {server.repository_url && (
                      <a
                        href={server.repository_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 text-xs text-primary hover:underline mt-1"
                      >
                        <ExternalLink className="w-3 h-3" />
                        {t("extensions.mcpMarket.repository")}
                      </a>
                    )}
                  </div>
                </div>
              ))}
            </div>
            {hasMore && (
              <div className="flex justify-center mt-4">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleLoadMore}
                  disabled={loadingMore}
                >
                  {loadingMore ? (
                    <>
                      <Loader2 className="h-4 w-4 animate-spin mr-2" />
                      {t("extensions.loading")}
                    </>
                  ) : (
                    t("extensions.mcpMarket.loadMore", { current: servers.length, total })
                  )}
                </Button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
