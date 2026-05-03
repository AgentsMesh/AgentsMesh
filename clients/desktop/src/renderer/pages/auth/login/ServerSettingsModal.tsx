import { useState } from "react";
import { useTranslations } from "next-intl";
import {
  Dialog, DialogContent, DialogHeader, DialogTitle, DialogBody, DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  listServers, getSelectedId, selectServer, addServer, updateServer,
  removeServer, isValidServerUrl, type Server,
// Relative path — desktop-only state, see AuthShell.tsx for context.
} from "../../../lib/server-config";

/**
 * Picker + CRUD dialog for the desktop server list. Selecting a
 * different server reloads the window so every in-flight fetch /
 * WebSocket re-resolves through env.ts → server-config — there's no
 * way to swap the API URL of a running session safely while sockets
 * are open and zustand state references the old origin.
 */
export function ServerSettingsModal({ open, onOpenChange }: {
  open: boolean;
  onOpenChange: (next: boolean) => void;
}) {
  const t = useTranslations();
  const [version, setVersion] = useState(0);
  const refresh = () => setVersion((v) => v + 1);
  const servers = listServers();
  void version;
  const selectedId = getSelectedId();

  const [draft, setDraft] = useState<{ id?: string; label: string; url: string } | null>(null);
  const [error, setError] = useState("");

  const startAdd = () => { setDraft({ label: "", url: "https://" }); setError(""); };
  const startEdit = (s: Server) => { setDraft({ id: s.id, label: s.label, url: s.url }); setError(""); };
  const cancelEdit = () => { setDraft(null); setError(""); };

  const handleSelect = (id: string) => { selectServer(id); refresh(); };

  const handleSave = () => {
    if (!draft) return;
    if (!draft.label.trim()) { setError(t("auth.loginPage.serverLabelRequired")); return; }
    if (!isValidServerUrl(draft.url)) { setError(t("auth.loginPage.serverInvalidUrl")); return; }
    if (draft.id) updateServer(draft.id, { label: draft.label, url: draft.url });
    else addServer({ label: draft.label, url: draft.url });
    setDraft(null); setError(""); refresh();
  };

  const handleRemove = (id: string) => { removeServer(id); refresh(); };

  const handleConnect = () => { onOpenChange(false); window.location.reload(); };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("auth.loginPage.serverSettingsTitle")}</DialogTitle>
        </DialogHeader>
        <DialogBody className="space-y-4">
          <p className="text-sm text-muted-foreground">{t("auth.loginPage.serverSettingsDesc")}</p>
          <ul className="space-y-2">
            {servers.map((s) => (
              <li key={s.id} className="flex items-center gap-3 rounded-md border border-border p-3">
                <input
                  type="radio"
                  name="server"
                  className="h-4 w-4"
                  checked={s.id === selectedId}
                  onChange={() => handleSelect(s.id)}
                  aria-label={s.label}
                />
                <div className="flex-1 min-w-0">
                  <div className="truncate text-sm font-medium text-foreground">{s.label}</div>
                  <div className="truncate text-xs text-muted-foreground">{s.url}</div>
                </div>
                {!s.readonly && (
                  <div className="flex gap-1">
                    <Button variant="ghost" size="sm" onClick={() => startEdit(s)}>
                      {t("auth.loginPage.serverEdit")}
                    </Button>
                    <Button variant="ghost" size="sm" onClick={() => handleRemove(s.id)}>
                      {t("auth.loginPage.serverRemove")}
                    </Button>
                  </div>
                )}
              </li>
            ))}
          </ul>
          {draft ? (
            <div className="space-y-2 rounded-md border border-dashed border-border p-3">
              <Input
                placeholder={t("auth.loginPage.serverLabel")}
                value={draft.label}
                onChange={(e) => setDraft({ ...draft, label: e.target.value })}
              />
              <Input
                placeholder={t("auth.loginPage.serverUrl")}
                value={draft.url}
                onChange={(e) => setDraft({ ...draft, url: e.target.value })}
              />
              {error && <p className="text-xs text-destructive">{error}</p>}
              <div className="flex justify-end gap-2">
                <Button variant="ghost" size="sm" onClick={cancelEdit}>
                  {t("common.cancel")}
                </Button>
                <Button size="sm" onClick={handleSave}>
                  {t("common.save")}
                </Button>
              </div>
            </div>
          ) : (
            <Button variant="outline" size="sm" onClick={startAdd}>
              {t("auth.loginPage.serverAdd")}
            </Button>
          )}
        </DialogBody>
        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>
            {t("common.close")}
          </Button>
          <Button onClick={handleConnect}>{t("auth.loginPage.serverConnect")}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
