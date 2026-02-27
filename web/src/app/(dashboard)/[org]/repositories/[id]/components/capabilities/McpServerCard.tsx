"use client";

import { useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { Button } from "@/components/ui/button";
import { ExternalLink, Settings, Trash2 } from "lucide-react";
import type { InstalledMcpServer } from "@/lib/api";

interface McpServerCardProps {
  mcpServer: InstalledMcpServer;
  canManage: boolean;
  onToggle: (mcp: InstalledMcpServer) => void;
  onDelete: (mcp: InstalledMcpServer) => void;
  onEditEnvVars?: (mcp: InstalledMcpServer) => void;
}

export function McpServerCard({ mcpServer, canManage, onToggle, onDelete, onEditEnvVars }: McpServerCardProps) {
  const t = useTranslations();

  const transportLabel = mcpServer.transport_type === "stdio"
    ? "stdio"
    : mcpServer.transport_type === "sse"
      ? "SSE"
      : mcpServer.transport_type === "http"
        ? "HTTP"
        : mcpServer.transport_type;

  // Build a short preview of the command
  const commandPreview = mcpServer.command
    ? `${mcpServer.command}${mcpServer.args?.length ? " " + mcpServer.args.join(" ") : ""}`
    : mcpServer.http_url || "";

  // Determine source URL for external link
  const sourceUrl = mcpServer.market_item?.repository_url || null;

  // Check if there are env vars to edit (has schema or has existing env vars)
  const hasEnvVarsToEdit = (mcpServer.market_item?.env_var_schema?.length ?? 0) > 0 ||
    Object.keys(mcpServer.env_vars || {}).length > 0;

  return (
    <div className="border border-border rounded-lg p-4 flex items-center justify-between gap-4">
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <span className="font-medium truncate">{mcpServer.name || mcpServer.slug}</span>
          <Badge variant="secondary" className="text-xs shrink-0">
            {transportLabel}
          </Badge>
          {mcpServer.market_item && (
            <Badge variant="outline" className="text-xs shrink-0">
              {t("extensions.sourceMarket")}
            </Badge>
          )}
          {sourceUrl && (
            <a
              href={sourceUrl}
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
        {commandPreview && (
          <p className="text-xs text-muted-foreground truncate font-mono">{commandPreview}</p>
        )}
      </div>

      {canManage && (
        <div className="flex items-center gap-3 shrink-0">
          {hasEnvVarsToEdit && onEditEnvVars && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => onEditEnvVars(mcpServer)}
              title={t("extensions.editEnvVars")}
            >
              <Settings className="w-4 h-4" />
            </Button>
          )}
          <Switch
            checked={mcpServer.is_enabled}
            onCheckedChange={() => onToggle(mcpServer)}
            aria-label={t("extensions.toggleEnabled")}
          />
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onDelete(mcpServer)}
            className="text-destructive hover:text-destructive"
          >
            <Trash2 className="w-4 h-4" />
          </Button>
        </div>
      )}
    </div>
  );
}
