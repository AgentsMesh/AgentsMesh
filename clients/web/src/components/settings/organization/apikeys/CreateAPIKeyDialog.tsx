"use client";

import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { ScopeSelector } from "./ScopeSelector";
import type { TranslationFn } from "../GeneralSettings";

const EXPIRATION_OPTIONS = [
  { value: 0, labelKey: "settings.apiKeys.createDialog.expirationNever" },
  { value: 30 * 86400, labelKey: "settings.apiKeys.createDialog.expiration30d" },
  { value: 90 * 86400, labelKey: "settings.apiKeys.createDialog.expiration90d" },
  { value: 365 * 86400, labelKey: "settings.apiKeys.createDialog.expiration1y" },
];

interface CreateAPIKeyDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreate: (data: {
    name: string;
    description?: string;
    scopes: string[];
    expires_in?: number;
  }) => Promise<void>;
  t: TranslationFn;
}

export function CreateAPIKeyDialog({ open, onOpenChange, onCreate, t }: CreateAPIKeyDialogProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [selectedScopes, setSelectedScopes] = useState<Set<string>>(new Set());
  const [expiresIn, setExpiresIn] = useState(0);
  const [creating, setCreating] = useState(false);

  useEffect(() => {
    if (open) {
      setName("");
      setDescription("");
      setSelectedScopes(new Set());
      setExpiresIn(0);
      setCreating(false);
    }
  }, [open]);

  const toggleScope = (scope: string) => {
    const next = new Set(selectedScopes);
    if (next.has(scope)) {
      next.delete(scope);
    } else {
      next.add(scope);
    }
    setSelectedScopes(next);
  };

  const handleCreate = async () => {
    if (!name.trim() || selectedScopes.size === 0) return;
    setCreating(true);
    try {
      await onCreate({
        name: name.trim(),
        description: description.trim() || undefined,
        scopes: Array.from(selectedScopes),
        expires_in: expiresIn > 0 ? expiresIn : undefined,
      });
    } catch (err) {
      console.error("Failed to create API key:", err);
    } finally {
      setCreating(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t("settings.apiKeys.createDialog.title")}</DialogTitle>
          <DialogDescription>{t("settings.apiKeys.description")}</DialogDescription>
        </DialogHeader>

        <div className="px-6 py-4 space-y-4">
          {/* Name */}
          <div>
            <label htmlFor="apikey-name" className="block text-sm font-medium mb-2">
              {t("settings.apiKeys.createDialog.nameLabel")}
            </label>
            <Input
              id="apikey-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder={t("settings.apiKeys.createDialog.namePlaceholder")}
            />
          </div>

          {/* Description */}
          <div>
            <label htmlFor="apikey-description" className="block text-sm font-medium mb-2">
              {t("settings.apiKeys.createDialog.descriptionLabel")}
            </label>
            <Input
              id="apikey-description"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t("settings.apiKeys.createDialog.descriptionPlaceholder")}
            />
          </div>

          {/* Expiration */}
          <div>
            <label htmlFor="apikey-expiration" className="block text-sm font-medium mb-2">
              {t("settings.apiKeys.createDialog.expirationLabel")}
            </label>
            <select
              id="apikey-expiration"
              value={expiresIn}
              onChange={(e) => setExpiresIn(Number(e.target.value))}
              className="w-full h-9 rounded-md border border-input bg-background px-3 text-sm"
            >
              {EXPIRATION_OPTIONS.map((opt) => (
                <option key={opt.value} value={opt.value}>
                  {t(opt.labelKey)}
                </option>
              ))}
            </select>
          </div>

          {/* Scopes */}
          <div>
            <label className="block text-sm font-medium mb-2">
              {t("settings.apiKeys.createDialog.scopesLabel")}
            </label>
            <ScopeSelector
              selectedScopes={selectedScopes}
              onToggle={toggleScope}
              t={t}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("settings.apiKeys.createDialog.cancel")}
          </Button>
          <Button
            onClick={handleCreate}
            disabled={creating || !name.trim() || selectedScopes.size === 0}
          >
            {creating ? t("settings.apiKeys.createDialog.creating") : t("settings.apiKeys.createDialog.create")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
