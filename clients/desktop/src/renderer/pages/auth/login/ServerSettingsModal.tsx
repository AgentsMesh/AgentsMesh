import { useState } from "react";
import { useTranslations } from "next-intl";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogBody, DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  getConfig, getCloudInfo, saveConfig, isValidServerUrl,
} from "../../../lib/server-config";

/**
 * Two-mode picker: AgentsMesh Cloud (built-in) or 自定义服务器 (custom).
 * The custom URL/label inputs only appear when 自定义服务器 is selected,
 * so the dialog stays uncluttered for the 95% of users who just want
 * the cloud option. Saving + Connect reloads the window so env.ts re-
 * resolves the active URL on the next render — there's no clean way
 * to hot-swap the API origin while sockets/fetches are in flight.
 *
 * The form lives in a separate component (ServerSettingsForm) that
 * renders only when `open` is true. State initialisers there read the
 * current config on mount, sidestepping the "setState in useEffect"
 * pattern lint forbids — every open cycle gets a fresh form.
 */
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
  const cloud = getCloudInfo();
  const [draft, setDraft] = useState(getConfig);
  const [error, setError] = useState("");

  const setKind = (kind: "cloud" | "custom") => setDraft((d) => ({ ...d, kind }));

  const handleConnect = () => {
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
    saveConfig(draft);
    onClose();
    window.location.reload();
  };

  return (
    <>
      <DialogBody className="space-y-4">
        <p className="text-sm text-muted-foreground">{t("auth.loginPage.serverSettingsDesc")}</p>

        <label className={`flex items-center gap-3 rounded-md border p-3 cursor-pointer transition-colors ${
          draft.kind === "cloud" ? "border-primary bg-primary/5" : "border-border hover:bg-muted/40"
        }`}>
          <input type="radio" name="kind" className="h-4 w-4"
            checked={draft.kind === "cloud"} onChange={() => setKind("cloud")} />
          <div className="flex-1 min-w-0">
            <div className="text-sm font-medium text-foreground">{cloud.label}</div>
            <div className="truncate text-xs text-muted-foreground">{cloud.url}</div>
          </div>
        </label>

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
