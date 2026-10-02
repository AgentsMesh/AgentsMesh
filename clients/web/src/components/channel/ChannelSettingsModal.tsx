"use client";

import { useEffect, useState } from "react";
import { Dialog, DialogContent, DialogFooter } from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Loader2 } from "lucide-react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { channelApi } from "@/lib/api/facade/channel";
import { useChannelStore } from "@/stores/channelStore";
import { useCurrentOrg, useCurrentUser } from "@/stores/auth";
import { cn } from "@/lib/utils";

type Tab = "basic" | "archive" | "danger";

interface SettingsChannel {
  id: number;
  name: string;
  description?: string;
  created_by_user_id?: number;
  is_archived: boolean;
}

interface ChannelSettingsModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  channel: SettingsChannel | null;
}

export function ChannelSettingsModal({ open, onOpenChange, channel }: ChannelSettingsModalProps) {
  const t = useTranslations("channels.settings");
  const [tab, setTab] = useState<Tab>("basic");
  const fetchChannels = useChannelStore((s) => s.fetchChannels);
  const deleteChannel = useChannelStore((s) => s.deleteChannel);
  const currentUser = useCurrentUser();
  const currentOrg = useCurrentOrg();
  const canDelete = !!channel && (
    currentOrg?.role === "owner" ||
    currentOrg?.role === "admin" ||
    channel.created_by_user_id === currentUser?.id
  );

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [saving, setSaving] = useState(false);
  const [archiving, setArchiving] = useState(false);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    if (channel && open) {
      setName(channel.name ?? "");
      setDescription(channel.description ?? "");
      setTab("basic");
    }
  }, [channel, open]);

  const handleSave = async () => {
    if (!channel) return;
    const trimmedName = name.trim();
    if (!trimmedName) return;
    setSaving(true);
    try {
      await channelApi.update(channel.id, { name: trimmedName, description });
      await fetchChannels({ includeArchived: true });
      toast.success(t("saved"));
      onOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("saveFailed"));
    } finally {
      setSaving(false);
    }
  };

  const handleToggleArchive = async () => {
    if (!channel) return;
    setArchiving(true);
    try {
      if (channel.is_archived) {
        await channelApi.unarchive(channel.id);
        toast.success(t("unarchived"));
      } else {
        await channelApi.archive(channel.id);
        toast.success(t("archived"));
      }
      await fetchChannels({ includeArchived: true });
      onOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("archiveFailed"));
    } finally {
      setArchiving(false);
    }
  };

  const handleDelete = async () => {
    if (!channel || !canDelete) return;
    setDeleting(true);
    try {
      await deleteChannel(channel.id);
      toast.success(t("deleted"));
      onOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : t("deleteFailed"));
    } finally {
      setDeleting(false);
    }
  };

  if (!channel) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md" title={t("title")}>
        <div className="flex border-b border-border px-4">
          <TabButton active={tab === "basic"} onClick={() => setTab("basic")}>
            {t("tabs.basic")}
          </TabButton>
          <TabButton active={tab === "archive"} onClick={() => setTab("archive")}>
            {t("tabs.archive")}
          </TabButton>
          {canDelete && (
            <TabButton active={tab === "danger"} onClick={() => setTab("danger")}>
              {t("tabs.danger")}
            </TabButton>
          )}
        </div>

        {tab === "basic" ? (
          <div className="flex flex-col gap-3 px-6 py-4">
            <label className="flex flex-col gap-1 text-sm">
              <span className="text-muted-foreground">{t("name")}</span>
              <Input
                value={name}
                onChange={(e) => setName(e.target.value)}
                data-testid="channel-settings-name"
              />
            </label>
            <label className="flex flex-col gap-1 text-sm">
              <span className="text-muted-foreground">{t("description")}</span>
              <Input
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder={t("descriptionPlaceholder")}
                data-testid="channel-settings-description"
              />
            </label>
          </div>
        ) : tab === "archive" ? (
          <div className="flex flex-col gap-3 px-6 py-4 text-sm">
            <p className="text-muted-foreground">
              {channel.is_archived ? t("unarchiveHint") : t("archiveHint")}
            </p>
            <Button
              variant={channel.is_archived ? "default" : "destructive"}
              onClick={handleToggleArchive}
              disabled={archiving}
              data-testid="channel-settings-archive-toggle"
            >
              {archiving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {channel.is_archived ? t("unarchive") : t("archive")}
            </Button>
          </div>
        ) : (
          <div className="flex flex-col gap-3 px-6 py-4 text-sm">
            <p className="text-muted-foreground">{t("deleteHint")}</p>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleting}
              data-testid="channel-settings-delete"
            >
              {deleting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {t("delete")}
            </Button>
          </div>
        )}

        {tab === "basic" && (
          <DialogFooter>
            <Button variant="ghost" onClick={() => onOpenChange(false)}>{t("cancel")}</Button>
            <Button onClick={handleSave} disabled={saving || !name.trim()} data-testid="channel-settings-save">
              {saving && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              {t("save")}
            </Button>
          </DialogFooter>
        )}
      </DialogContent>
    </Dialog>
  );
}

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "border-b-2 px-3 py-2 text-sm font-medium transition-colors",
        active
          ? "border-primary text-foreground"
          : "border-transparent text-muted-foreground hover:text-foreground",
      )}
    >
      {children}
    </button>
  );
}

export default ChannelSettingsModal;
