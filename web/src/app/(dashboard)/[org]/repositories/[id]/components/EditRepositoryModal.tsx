"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { repositoryApi, RepositoryData } from "@/lib/api";
import { useTranslations } from "@/lib/i18n/client";

interface EditRepositoryModalProps {
  repository: RepositoryData;
  onClose: () => void;
  onUpdated: () => void;
}

/**
 * EditRepositoryModal - Modal for editing repository settings
 */
export function EditRepositoryModal({
  repository,
  onClose,
  onUpdated,
}: EditRepositoryModalProps) {
  const t = useTranslations();
  const [name, setName] = useState(repository.name);
  const [defaultBranch, setDefaultBranch] = useState(repository.default_branch);
  const [ticketPrefix, setTicketPrefix] = useState(repository.ticket_prefix || "");
  const [isActive, setIsActive] = useState(repository.is_active);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");

  const handleUpdate = async () => {
    if (!name) {
      setError(t("repositories.edit.nameRequired"));
      return;
    }

    setLoading(true);
    setError("");

    try {
      await repositoryApi.update(repository.id, {
        name,
        default_branch: defaultBranch,
        ticket_prefix: ticketPrefix || undefined,
        is_active: isActive,
      });
      onUpdated();
    } catch (err) {
      console.error("Failed to update repository:", err);
      setError(t("repositories.edit.updateFailed"));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-background border border-border rounded-lg w-full max-w-md p-6">
        <h2 className="text-xl font-semibold mb-4">{t("repositories.edit.title")}</h2>

        {error && (
          <div className="mb-4 p-3 bg-destructive/10 text-destructive text-sm rounded-md">
            {error}
          </div>
        )}

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">
              {t("repositories.edit.name")} <span className="text-destructive">*</span>
            </label>
            <Input value={name} onChange={(e) => setName(e.target.value)} />
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">{t("repositories.edit.defaultBranch")}</label>
            <Input
              value={defaultBranch}
              onChange={(e) => setDefaultBranch(e.target.value)}
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-2">
              {t("repositories.edit.ticketPrefixOptional")}
            </label>
            <Input
              placeholder="PROJ"
              value={ticketPrefix}
              onChange={(e) => setTicketPrefix(e.target.value.toUpperCase())}
            />
          </div>

          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="is-active"
              checked={isActive}
              onChange={(e) => setIsActive(e.target.checked)}
              className="rounded border-border"
            />
            <label htmlFor="is-active" className="text-sm font-medium">
              {t("repositories.edit.active")}
            </label>
          </div>
        </div>

        <div className="flex justify-end gap-3 mt-6">
          <Button variant="outline" onClick={onClose}>
            {t("common.cancel")}
          </Button>
          <Button onClick={handleUpdate} disabled={!name || loading}>
            {loading ? t("repositories.edit.saving") : t("repositories.edit.saveChanges")}
          </Button>
        </div>
      </div>
    </div>
  );
}
