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

// Wire values are the single source of truth — used as i18n keys directly,
// so adding a new mode only requires a single line in messages/{en,zh}/app.json.
const MODE_VALUES = ["bypassPermissions", "acceptEdits", "default", "dontAsk"] as const;
type ModeValue = (typeof MODE_VALUES)[number];

export function AcpPermissionModeSelector({ podKey }: { podKey: string }) {
  const t = useTranslations("acp.modeSelector");
  const mode = useAcpSessionField(podKey, (s) => s.configuration.permissionMode);

  const handleSelect = useCallback((value: string) => {
    if (!relayPool.isConnected(podKey)) return;
    relayPool.sendAcpCommand(podKey, { type: "set_permission_mode", mode: value });
  }, [podKey]);

  const currentKey = (MODE_VALUES as readonly string[]).includes(mode) ? mode : "unknown";

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className="flex items-center gap-1 px-2 py-1 text-xs rounded hover:bg-muted transition-colors outline-none focus:bg-muted"
        title={t(`${currentKey}.desc`)}
      >
        <Shield className="h-3 w-3 text-muted-foreground" />
        <span className="text-muted-foreground">{t(`${currentKey}.label`)}</span>
        <ChevronDown className="h-3 w-3 text-muted-foreground" />
      </DropdownMenuTrigger>
      <DropdownMenuContent side="top" align="start" className="w-48">
        {MODE_VALUES.map((value: ModeValue) => (
          <DropdownMenuItem
            key={value}
            onSelect={() => handleSelect(value)}
            className={mode === value ? "bg-muted font-medium" : ""}
          >
            <div className="flex flex-col gap-0.5">
              <div className="text-xs">{t(`${value}.label`)}</div>
              <div className="text-muted-foreground text-[10px]">{t(`${value}.desc`)}</div>
            </div>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
