"use client";

import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type {
  SSOConfig,
  SSOProtocol,
  CreateSSOConfigRequest,
} from "@/lib/api/sso";
import { defaultForm } from "./sso-form-types";
import { OIDCSection } from "./oidc-section";
import { SAMLSection } from "./saml-section";
import { LDAPSection } from "./ldap-section";

interface SSOFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  config?: SSOConfig | null;
  onSubmit: (data: CreateSSOConfigRequest) => Promise<void>;
}

function buildFormFromConfig(config: SSOConfig): CreateSSOConfigRequest {
  return {
    domain: config.domain,
    name: config.name,
    protocol: config.protocol,
    is_enabled: config.is_enabled,
    enforce_sso: config.enforce_sso,
    oidc_issuer_url: config.oidc_issuer_url || "",
    oidc_client_id: config.oidc_client_id || "",
    oidc_client_secret: "",
    oidc_scopes: config.oidc_scopes || "openid profile email",
    saml_idp_metadata_url: config.saml_idp_metadata_url || "",
    saml_idp_sso_url: config.saml_idp_sso_url || "",
    saml_idp_cert: "",
    saml_sp_entity_id: config.saml_sp_entity_id || "",
    saml_name_id_format: config.saml_name_id_format || "",
    ldap_host: config.ldap_host || "",
    ldap_port: config.ldap_port || 389,
    ldap_use_tls: config.ldap_use_tls || false,
    ldap_bind_dn: config.ldap_bind_dn || "",
    ldap_bind_password: "",
    ldap_base_dn: config.ldap_base_dn || "",
    ldap_user_filter: config.ldap_user_filter || "(uid=%s)",
    ldap_email_attr: config.ldap_email_attr || "mail",
    ldap_name_attr: config.ldap_name_attr || "cn",
    ldap_username_attr: config.ldap_username_attr || "uid",
  };
}

export function SSOFormDialog({ open, onOpenChange, config, onSubmit }: SSOFormDialogProps) {
  const [form, setForm] = useState<CreateSSOConfigRequest>(defaultForm);
  const [saving, setSaving] = useState(false);
  const isEdit = !!config;

  useEffect(() => {
    setForm(config ? buildFormFromConfig(config) : defaultForm);
  }, [config, open]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setSaving(true);
    try {
      await onSubmit(form);
      onOpenChange(false);
    } finally {
      setSaving(false);
    }
  };

  const update = (field: keyof CreateSSOConfigRequest, value: unknown) => {
    setForm((prev) => ({ ...prev, [field]: value }));
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] overflow-y-auto sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit SSO Config" : "Create SSO Config"}</DialogTitle>
          <DialogDescription>
            {isEdit
              ? "Update the SSO configuration for this domain."
              : "Configure single sign-on for a domain."}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <CommonFields form={form} update={update} />
          {form.protocol === "oidc" && <OIDCSection form={form} update={update} isEdit={isEdit} />}
          {form.protocol === "saml" && <SAMLSection form={form} update={update} isEdit={isEdit} />}
          {form.protocol === "ldap" && <LDAPSection form={form} update={update} isEdit={isEdit} />}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={saving}>
              {saving ? "Saving..." : isEdit ? "Update" : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function CommonFields({
  form,
  update,
}: {
  form: CreateSSOConfigRequest;
  update: (field: keyof CreateSSOConfigRequest, value: unknown) => void;
}) {
  return (
    <>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="domain">Domain</Label>
          <Input
            id="domain"
            placeholder="example.com"
            value={form.domain}
            onChange={(e) => update("domain", e.target.value)}
            required
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="name">Display Name</Label>
          <Input
            id="name"
            placeholder="Company SSO"
            value={form.name}
            onChange={(e) => update("name", e.target.value)}
            required
          />
        </div>
      </div>
      <div className="space-y-2">
        <Label>Protocol</Label>
        <Select
          value={form.protocol}
          onValueChange={(v) => update("protocol", v as SSOProtocol)}
        >
          <SelectTrigger>
            <SelectValue placeholder="Select protocol" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="oidc">OIDC (OpenID Connect)</SelectItem>
            <SelectItem value="saml">SAML 2.0</SelectItem>
            <SelectItem value="ldap">LDAP</SelectItem>
          </SelectContent>
        </Select>
      </div>
    </>
  );
}
