import { useState } from "react";
import { useTranslations } from "next-intl";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogBody, DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  getConfig, getPresets, saveConfig, isValidServerUrl, type ServerKind,
} from "../../../lib/server-config";
import { useAuthStore } from "@/stores/auth";

export function ServerSettingsModal({ open, onOpenChange }: {
  open: boolean;
  onOpenChange: (next: boolean) => void;
}) {
  const t = useTranslations();
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("auth.loginPage.serverSettingsTitle")}</DialogTitle>
        </DialogHeader>
        {open && <ServerSettingsForm onClose={() => onOpenChange(false)} />}
      </DialogContent>
    </Dialog>
  );
}

function ServerSettingsForm({ onClose }: { onClose: () => void }) {
  const t = useTranslations();
  const presets = getPresets();
  const [draft, setDraft] = useState(getConfig);
  const [error, setError] = useState("");

  const setKind = (kind: ServerKind) => setDraft((d) => ({ ...d, kind }));

  const handleConnect = async () => {
    if (draft.kind === "custom") {
      if (!draft.customLabel.trim()) {
        setError(t("auth.loginPage.serverLabelRequired"));
        return;
      }
      if (!isValidServerUrl(draft.customUrl)) {
        setError(t("auth.loginPage.serverInvalidUrl"));
        return;
      }
    }
    // Switching server == logging out. Logout MUST be fail-soft: switch reason is often
    // that the old server is unreachable; Rust core wipes tokens during AppState rebuild
    // regardless, and mainWindow.reload() starts a clean session.
    const wasAuthed = useAuthStore.getState().isAuthenticated();
    if (wasAuthed) {
      const ok = window.confirm(
        "切换服务器将登出当前账号，是否继续？\n\nSwitching server will log you out of the current account."
      );
      if (!ok) return;
      await useAuthStore.getState().logout().catch((err) => {
        console.warn("[server-switch] logout failed (proceeding anyway):", err);
      });
    }
    try {
      await saveConfig(draft);
    } catch (e) {
      // Main rejects when activeUrl() throws (malformed custom URL past renderer pre-check).
      setError((e as Error).message || t("auth.loginPage.serverInvalidUrl"));
      return;
    }
    onClose();
  };

  return (
    <>
      <DialogBody className="space-y-3">
        <p className="text-sm text-muted-foreground">{t("auth.loginPage.serverSettingsDesc")}</p>

        {presets.map((p) => (
          <PresetRow
            key={p.kind}
            label={p.label}
            url={p.url}
            checked={draft.kind === p.kind}
            onSelect={() => setKind(p.kind)}
          />
        ))}

        <label className={`flex items-center gap-3 rounded-md border p-3 cursor-pointer transition-colors ${
          draft.kind === "custom" ? "border-primary bg-primary/5" : "border-border hover:bg-muted/40"
        }`}>
          <input type="radio" name="kind" className="h-4 w-4"
            checked={draft.kind === "custom"} onChange={() => setKind("custom")} />
          <div className="flex-1 min-w-0">
            <div className="text-sm font-medium text-foreground">
              {t("auth.loginPage.serverCustom")}
            </div>
            <div className="truncate text-xs text-muted-foreground">
              {t("auth.loginPage.serverCustomDesc")}
            </div>
          </div>
        </label>

        {draft.kind === "custom" && (
          <div className="space-y-2 rounded-md border border-dashed border-border p-3">
            <div className="space-y-1">
              <label className="text-xs font-medium text-foreground">
                {t("auth.loginPage.serverLabel")}
              </label>
              <Input
                placeholder={t("auth.loginPage.serverLabelPlaceholder")}
                value={draft.customLabel}
                onChange={(e) => setDraft({ ...draft, customLabel: e.target.value })}
              />
            </div>
            <div className="space-y-1">
              <label className="text-xs font-medium text-foreground">
                {t("auth.loginPage.serverUrl")}
              </label>
              <Input
                placeholder={t("auth.loginPage.serverUrlPlaceholder")}
                value={draft.customUrl}
                onChange={(e) => setDraft({ ...draft, customUrl: e.target.value })}
              />
            </div>
            {error && <p className="text-xs text-destructive">{error}</p>}
          </div>
        )}
      </DialogBody>
      <DialogFooter>
        <Button variant="ghost" onClick={onClose}>
          {t("common.cancel")}
        </Button>
        <Button onClick={handleConnect}>{t("auth.loginPage.serverConnect")}</Button>
      </DialogFooter>
    </>
  );
}

function PresetRow({ label, url, checked, onSelect }: {
  label: string;
  url: string;
  checked: boolean;
  onSelect: () => void;
}) {
  return (
    <label className={`flex items-center gap-3 rounded-md border p-3 cursor-pointer transition-colors ${
      checked ? "border-primary bg-primary/5" : "border-border hover:bg-muted/40"
    }`}>
      <input type="radio" name="kind" className="h-4 w-4"
        checked={checked} onChange={onSelect} />
      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium text-foreground">{label}</div>
        <div className="truncate text-xs text-muted-foreground">{url}</div>
      </div>
    </label>
  );
}
