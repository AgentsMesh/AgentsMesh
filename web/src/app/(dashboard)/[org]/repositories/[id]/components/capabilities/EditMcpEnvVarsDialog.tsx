"use client";

import { useState, useEffect, useCallback } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { extensionApi, InstalledMcpServer } from "@/lib/api";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogBody, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

interface EditMcpEnvVarsDialogProps {
  repositoryId: number;
  mcpServer: InstalledMcpServer;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onUpdated: () => void;
}

interface EnvVarEntry {
  key: string;
  value: string;
  label?: string;
  required?: boolean;
  sensitive?: boolean;
  placeholder?: string;
  fromSchema?: boolean;
}

export function EditMcpEnvVarsDialog({ repositoryId, mcpServer, open, onOpenChange, onUpdated }: EditMcpEnvVarsDialogProps) {
  const t = useTranslations();
  const [saving, setSaving] = useState(false);
  const [envVars, setEnvVars] = useState<EnvVarEntry[]>([]);

  const hasSchema = (mcpServer.market_item?.env_var_schema?.length ?? 0) > 0;

  useEffect(() => {
    if (!open) return;

    if (hasSchema) {
      // Market-installed: use schema structure, pre-fill with existing values
      const entries: EnvVarEntry[] = mcpServer.market_item!.env_var_schema!.map((entry) => ({
        key: entry.name,
        value: mcpServer.env_vars?.[entry.name] || "",
        label: entry.label,
        required: entry.required,
        sensitive: entry.sensitive,
        placeholder: entry.placeholder,
        fromSchema: true,
      }));
      setEnvVars(entries);
    } else {
      // Custom-installed: show existing key-value pairs
      const entries: EnvVarEntry[] = Object.entries(mcpServer.env_vars || {}).map(([key, value]) => ({
        key,
        value,
      }));
      setEnvVars(entries);
    }
  }, [open, mcpServer, hasSchema]);

  const handleSave = useCallback(async () => {
    setSaving(true);
    try {
      const envRecord: Record<string, string> = {};
      envVars.forEach(({ key, value }) => {
        if (key.trim()) {
          envRecord[key.trim()] = value.trim();
        }
      });
      await extensionApi.updateMcpServer(repositoryId, mcpServer.id, { env_vars: envRecord });
      toast.success(t("extensions.envVarsUpdated"));
      onUpdated();
      onOpenChange(false);
    } catch (error) {
      toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToUpdate")));
    } finally {
      setSaving(false);
    }
  }, [repositoryId, mcpServer.id, envVars, t, onUpdated, onOpenChange]);

  const hasUnfilledRequired = envVars.some((e) => e.required && !e.value.trim());

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("extensions.editEnvVars")}</DialogTitle>
        </DialogHeader>
        <DialogBody>
          <p className="text-sm text-muted-foreground mb-4">
            {mcpServer.name || mcpServer.slug} — {t("extensions.envVarsHint")}
          </p>

          {hasSchema ? (
            // Schema-based form (for market-installed MCP servers)
            <div className="space-y-3">
              {envVars.map((entry, idx) => (
                <div key={entry.key}>
                  <label className="text-sm font-medium mb-1 block">
                    {entry.label || entry.key}
                    {entry.required && <span className="text-destructive ml-1">*</span>}
                  </label>
                  <Input
                    type={entry.sensitive ? "password" : "text"}
                    placeholder={entry.placeholder || entry.key}
                    value={entry.value}
                    onChange={(e) => {
                      setEnvVars((prev) =>
                        prev.map((item, i) => (i === idx ? { ...item, value: e.target.value } : item))
                      );
                    }}
                  />
                </div>
              ))}
            </div>
          ) : (
            // Free-form key-value editor (for custom MCP servers)
            <div className="space-y-2">
              {envVars.length === 0 && (
                <p className="text-sm text-muted-foreground py-2">{t("extensions.noEnvVars")}</p>
              )}
              {envVars.map((entry, idx) => (
                <div key={idx} className="flex gap-2">
                  <Input
                    placeholder="KEY"
                    value={entry.key}
                    onChange={(e) => {
                      setEnvVars((prev) =>
                        prev.map((item, i) => (i === idx ? { ...item, key: e.target.value } : item))
                      );
                    }}
                  />
                  <Input
                    placeholder="value"
                    value={entry.value}
                    onChange={(e) => {
                      setEnvVars((prev) =>
                        prev.map((item, i) => (i === idx ? { ...item, value: e.target.value } : item))
                      );
                    }}
                  />
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setEnvVars((prev) => prev.filter((_, i) => i !== idx))}
                    className="text-destructive shrink-0"
                  >
                    x
                  </Button>
                </div>
              ))}
              <Button
                variant="outline"
                size="sm"
                onClick={() => setEnvVars((prev) => [...prev, { key: "", value: "" }])}
              >
                {t("extensions.addEnvVar")}
              </Button>
            </div>
          )}
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("extensions.cancel")}
          </Button>
          <Button disabled={saving || hasUnfilledRequired} onClick={handleSave}>
            {saving ? t("extensions.saving") : t("extensions.save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
