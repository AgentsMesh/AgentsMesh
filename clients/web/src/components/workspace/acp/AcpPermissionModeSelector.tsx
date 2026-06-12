"use client";

import { useCallback } from "react";
import { Shield, ChevronDown } from "lucide-react";
import { useTranslations } from "next-intl";
import { relayPool } from "@/stores/relayConnection";
import { useAcpSessionField } from "@/stores/acpSession";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

// The agent advertises its supported permission modes at initialize
// (configuration.supportedPermissionModes); we render those. Agents that
// advertise none (Claude/codex/gemini) fall back to this Claude Code set.
// Wire values double as i18n keys (messages/{en,zh}/app.json acp.modeSelector).
const FALLBACK_MODES = ["bypassPermissions", "acceptEdits", "default", "dontAsk"];

export function AcpPermissionModeSelector({ podKey }: { podKey: string }) {
  const t = useTranslations("acp.modeSelector");
  const mode = useAcpSessionField(podKey, (s) => s.configuration.permissionMode);
  const supported = useAcpSessionField(podKey, (s) => s.configuration.supportedPermissionModes);
  const modeValues = supported.length > 0 ? supported : FALLBACK_MODES;

  // Wire values double as i18n keys. en-fallback (i18n/request.ts) covers known
  // modes across locales; t.has guards an advertised value with no entry at all
  // → show the raw value instead of a key-path or throw.
  const label = (v: string) => (t.has(`${v}.label`) ? t(`${v}.label`) : v);
  const desc = (v: string) => (t.has(`${v}.desc`) ? t(`${v}.desc`) : "");

  const handleSelect = useCallback((value: string) => {
    if (!relayPool.isConnected(podKey)) return;
    relayPool.sendAcpCommand(podKey, { type: "set_permission_mode", mode: value });
  }, [podKey]);

  const currentKey = modeValues.includes(mode) ? mode : "unknown";

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className="flex items-center gap-1 px-2 py-1 text-xs rounded hover:bg-muted transition-colors outline-none focus:bg-muted"
        title={desc(currentKey)}
      >
        <Shield className="h-3 w-3 text-muted-foreground" />
        <span className="text-muted-foreground">{label(currentKey)}</span>
        <ChevronDown className="h-3 w-3 text-muted-foreground" />
      </DropdownMenuTrigger>
      <DropdownMenuContent side="top" align="start" className="w-48">
        {modeValues.map((value) => (
          <DropdownMenuItem
            key={value}
            onSelect={() => handleSelect(value)}
            className={mode === value ? "bg-muted font-medium" : ""}
          >
            <div className="flex flex-col gap-0.5">
              <div className="text-xs">{label(value)}</div>
              <div className="text-muted-foreground text-[10px]">{desc(value)}</div>
            </div>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
