"use client";

import { useCallback } from "react";
import { Shield, ChevronDown } from "lucide-react";
import { relayPool } from "@/stores/relayConnection";
import { useAcpSessionField } from "@/stores/acpSession";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

const MODES = [
  { value: "bypassPermissions", label: "Bypass", desc: "Auto-approve all" },
  { value: "acceptEdits", label: "Accept Edits", desc: "Auto-approve file edits" },
  { value: "default", label: "Default", desc: "Approve each tool" },
  { value: "dontAsk", label: "Don't Ask", desc: "Deny unless allowlisted" },
] as const;

const UNKNOWN_MODE = { value: "", label: "—", desc: "Mode not yet reported by runner" } as const;

export function AcpPermissionModeSelector({ podKey }: { podKey: string }) {
  const mode = useAcpSessionField(podKey, (s) => s.configuration.permissionMode);

  const handleSelect = useCallback((value: string) => {
    if (!relayPool.isConnected(podKey)) return;
    relayPool.sendAcpCommand(podKey, { type: "set_permission_mode", mode: value });
  }, [podKey]);

  const current = MODES.find((m) => m.value === mode) ?? UNKNOWN_MODE;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        className="flex items-center gap-1 px-2 py-1 text-xs rounded hover:bg-muted transition-colors outline-none focus:bg-muted"
        title={current.desc}
      >
        <Shield className="h-3 w-3 text-muted-foreground" />
        <span className="text-muted-foreground">{current.label}</span>
        <ChevronDown className="h-3 w-3 text-muted-foreground" />
      </DropdownMenuTrigger>
      <DropdownMenuContent side="top" align="start" className="w-48">
        {MODES.map((m) => (
          <DropdownMenuItem
            key={m.value}
            onSelect={() => handleSelect(m.value)}
            className={mode === m.value ? "bg-muted font-medium" : ""}
          >
            <div className="flex flex-col gap-0.5">
              <div className="text-xs">{m.label}</div>
              <div className="text-muted-foreground text-[10px]">{m.desc}</div>
            </div>
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
