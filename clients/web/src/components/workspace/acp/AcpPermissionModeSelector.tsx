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

const MODE_VALUES = ["bypassPermissions", "acceptEdits", "default", "dontAsk"] as const;
type ModeValue = (typeof MODE_VALUES)[number];

// modeKey maps the wire value (kept stable across runner/backend) to the
// i18n key under acp.modeSelector. Diverged purely so the JSON keys read
// naturally (e.g. "bypass" instead of "bypassPermissions").
const modeKey: Record<ModeValue, string> = {
  bypassPermissions: "bypass",
  acceptEdits: "acceptEdits",
  default: "default",
  dontAsk: "dontAsk",
};

export function AcpPermissionModeSelector({ podKey }: { podKey: string }) {
  const t = useTranslations("acp.modeSelector");
  const mode = useAcpSessionField(podKey, (s) => s.configuration.permissionMode);

  const handleSelect = useCallback((value: string) => {
    if (!relayPool.isConnected(podKey)) return;
    relayPool.sendAcpCommand(podKey, { type: "set_permission_mode", mode: value });
  }, [podKey]);

  const currentKey = (MODE_VALUES as readonly string[]).includes(mode) ? modeKey[mode as ModeValue] : "unknown";
  const currentLabel = t(`${currentKey}.label`);
  const currentDesc = t(`${currentKey}.desc`);

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className="flex items-center gap-1 px-2 py-1 text-xs rounded hover:bg-muted transition-colors outline-none focus:bg-muted"
        title={currentDesc}
      >
        <Shield className="h-3 w-3 text-muted-foreground" />
        <span className="text-muted-foreground">{currentLabel}</span>
        <ChevronDown className="h-3 w-3 text-muted-foreground" />
      </DropdownMenuTrigger>
      <DropdownMenuContent side="top" align="start" className="w-48">
        {MODE_VALUES.map((value) => (
          <DropdownMenuItem
            key={value}
            onSelect={() => handleSelect(value)}
            className={mode === value ? "bg-muted font-medium" : ""}
          >
            <div className="flex flex-col gap-0.5">
              <div className="text-xs">{t(`${modeKey[value]}.label`)}</div>
              <div className="text-muted-foreground text-[10px]">{t(`${modeKey[value]}.desc`)}</div>
            </div>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
