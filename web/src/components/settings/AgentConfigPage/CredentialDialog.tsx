"use client";

import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { CredentialDialogProps, CredentialFormData, CredentialMethod } from "./types";

/**
 * CredentialDialog - Dialog for adding or editing credential profiles
 *
 * Supports two mutually exclusive authentication methods:
 * - API Key (ANTHROPIC_API_KEY)
 * - Auth Token (ANTHROPIC_AUTH_TOKEN)
 *
 * base_url is optional and works with either method.
 * In edit mode, base_url (type: "text") is echoed back from configured_values;
 * api_key/auth_token (type: "secret") are never echoed.
 */
export function CredentialDialog({
  open,
  onOpenChange,
  editingProfile,
  onSubmit,
  t,
}: CredentialDialogProps) {
  const [formName, setFormName] = useState("");
  const [formDescription, setFormDescription] = useState("");
  const [formBaseUrl, setFormBaseUrl] = useState("");
  const [formApiKey, setFormApiKey] = useState("");
  const [formAuthToken, setFormAuthToken] = useState("");
  const [credentialMethod, setCredentialMethod] = useState<CredentialMethod>("api_key");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Reset form when dialog opens/closes or editing profile changes
  useEffect(() => {
    if (open) {
      if (editingProfile) {
        setFormName(editingProfile.name);
        setFormDescription(editingProfile.description || "");
        // Echo back non-secret values (base_url is type: "text")
        setFormBaseUrl(editingProfile.configured_values?.base_url || "");
        setFormApiKey("");
        setFormAuthToken("");
        // Determine current method from configured_fields
        if (editingProfile.configured_fields?.includes("auth_token")) {
          setCredentialMethod("auth_token");
        } else {
          setCredentialMethod("api_key");
        }
      } else {
        setFormName("");
        setFormDescription("");
        setFormBaseUrl("");
        setFormApiKey("");
        setFormAuthToken("");
        setCredentialMethod("api_key");
      }
      setError(null);
    }
  }, [open, editingProfile]);

  const handleSubmit = async () => {
    if (!formName.trim()) return;

    try {
      setSubmitting(true);
      setError(null);

      const formData: CredentialFormData = {
        name: formName,
        description: formDescription,
        baseUrl: formBaseUrl,
        apiKey: formApiKey,
        authToken: formAuthToken,
        credentialMethod,
      };

      await onSubmit(formData, editingProfile);
      onOpenChange(false);
    } catch (err) {
      console.error("Failed to save profile:", err);
      setError(t("settings.agentCredentials.failedToSave"));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>
            {editingProfile
              ? t("settings.agentCredentials.editProfile")
              : t("settings.agentCredentials.addProfile")}
          </DialogTitle>
          <DialogDescription>
            {t("settings.agentCredentials.customProfileDescription")}
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          {error && (
            <div className="text-sm text-destructive">{error}</div>
          )}

          <div className="grid gap-2">
            <Label htmlFor="name">{t("settings.agentCredentials.name")}</Label>
            <Input
              id="name"
              value={formName}
              onChange={(e) => setFormName(e.target.value)}
              placeholder={t("settings.agentCredentials.namePlaceholder")}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="description">{t("settings.agentCredentials.descriptionLabel")}</Label>
            <Textarea
              id="description"
              value={formDescription}
              onChange={(e) => setFormDescription(e.target.value)}
              placeholder={t("settings.agentCredentials.descriptionPlaceholder")}
              rows={2}
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="base_url">
              {t("settings.agentCredentials.baseUrl")}
              <span className="text-xs text-muted-foreground ml-1">
                ({t("common.optional")})
              </span>
            </Label>
            <Input
              id="base_url"
              value={formBaseUrl}
              onChange={(e) => setFormBaseUrl(e.target.value)}
              placeholder="https://api.anthropic.com"
            />
          </div>

          {/* Credential method toggle */}
          <div className="grid gap-2">
            <Label>{t("settings.agentCredentials.credentialMethod")}</Label>
            <Tabs
              value={credentialMethod}
              onValueChange={(v) => setCredentialMethod(v as CredentialMethod)}
            >
              <TabsList className="w-full">
                <TabsTrigger value="api_key" className="flex-1">
                  {t("settings.agentCredentials.credentialMethodApiKey")}
                </TabsTrigger>
                <TabsTrigger value="auth_token" className="flex-1">
                  {t("settings.agentCredentials.credentialMethodAuthToken")}
                </TabsTrigger>
              </TabsList>
            </Tabs>
          </div>

          {/* API Key input (shown when api_key method selected) */}
          {credentialMethod === "api_key" && (
            <div className="grid gap-2">
              <Label htmlFor="api_key">{t("settings.agentCredentials.apiKey")}</Label>
              <Input
                id="api_key"
                type="password"
                value={formApiKey}
                onChange={(e) => setFormApiKey(e.target.value)}
                placeholder={editingProfile ? t("settings.agentCredentials.apiKeyPlaceholder") : "sk-..."}
              />
              {editingProfile && (
                <p className="text-xs text-muted-foreground">
                  {t("settings.agentCredentials.apiKeyEditHint")}
                </p>
              )}
            </div>
          )}

          {/* Auth Token input (shown when auth_token method selected) */}
          {credentialMethod === "auth_token" && (
            <div className="grid gap-2">
              <Label htmlFor="auth_token">{t("settings.agentCredentials.authToken")}</Label>
              <Input
                id="auth_token"
                type="password"
                value={formAuthToken}
                onChange={(e) => setFormAuthToken(e.target.value)}
                placeholder={editingProfile ? t("settings.agentCredentials.authTokenPlaceholder") : ""}
              />
              {editingProfile && (
                <p className="text-xs text-muted-foreground">
                  {t("settings.agentCredentials.authTokenEditHint")}
                </p>
              )}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button onClick={handleSubmit} disabled={submitting || !formName.trim()}>
            {submitting
              ? t("common.saving")
              : editingProfile
              ? t("common.save")
              : t("common.create")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export default CredentialDialog;
