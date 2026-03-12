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

interface SSOFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  config?: SSOConfig | null;
  onSubmit: (data: CreateSSOConfigRequest) => Promise<void>;
}

const defaultForm: CreateSSOConfigRequest = {
  domain: "",
  name: "",
  protocol: "oidc",
  is_enabled: true,
  enforce_sso: false,
  oidc_issuer_url: "",
  oidc_client_id: "",
  oidc_client_secret: "",
  oidc_scopes: "openid profile email",
  saml_idp_metadata_url: "",
  saml_idp_sso_url: "",
  saml_idp_cert: "",
  saml_sp_entity_id: "",
  saml_name_id_format: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
  ldap_host: "",
  ldap_port: 389,
  ldap_use_tls: false,
  ldap_bind_dn: "",
  ldap_bind_password: "",
  ldap_base_dn: "",
  ldap_user_filter: "(uid=%s)",
  ldap_email_attr: "mail",
  ldap_name_attr: "cn",
  ldap_username_attr: "uid",
};

export function SSOFormDialog({ open, onOpenChange, config, onSubmit }: SSOFormDialogProps) {
  const [form, setForm] = useState<CreateSSOConfigRequest>(defaultForm);
  const [saving, setSaving] = useState(false);

  const isEdit = !!config;

  useEffect(() => {
    if (config) {
      setForm({
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
      });
    } else {
      setForm(defaultForm);
    }
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
          {/* Common fields */}
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

          {/* OIDC fields */}
          {form.protocol === "oidc" && (
            <fieldset className="space-y-4 rounded-lg border border-border p-4">
              <legend className="px-2 text-sm font-medium">OIDC Settings</legend>
              <div className="space-y-2">
                <Label htmlFor="oidc_issuer_url">Issuer URL</Label>
                <Input
                  id="oidc_issuer_url"
                  placeholder="https://accounts.google.com"
                  value={form.oidc_issuer_url}
                  onChange={(e) => update("oidc_issuer_url", e.target.value)}
                />
              </div>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="oidc_client_id">Client ID</Label>
                  <Input
                    id="oidc_client_id"
                    value={form.oidc_client_id}
                    onChange={(e) => update("oidc_client_id", e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="oidc_client_secret">
                    Client Secret{isEdit && " (leave blank to keep current)"}
                  </Label>
                  <Input
                    id="oidc_client_secret"
                    type="password"
                    value={form.oidc_client_secret}
                    onChange={(e) => update("oidc_client_secret", e.target.value)}
                  />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="oidc_scopes">Scopes</Label>
                <Input
                  id="oidc_scopes"
                  placeholder="openid profile email"
                  value={form.oidc_scopes}
                  onChange={(e) => update("oidc_scopes", e.target.value)}
                />
              </div>
            </fieldset>
          )}

          {/* SAML fields */}
          {form.protocol === "saml" && (
            <fieldset className="space-y-4 rounded-lg border border-border p-4">
              <legend className="px-2 text-sm font-medium">SAML Settings</legend>
              <div className="space-y-2">
                <Label htmlFor="saml_idp_metadata_url">IdP Metadata URL</Label>
                <Input
                  id="saml_idp_metadata_url"
                  placeholder="https://idp.example.com/metadata"
                  value={form.saml_idp_metadata_url}
                  onChange={(e) => update("saml_idp_metadata_url", e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="saml_idp_sso_url">IdP SSO URL</Label>
                <Input
                  id="saml_idp_sso_url"
                  placeholder="https://idp.example.com/sso"
                  value={form.saml_idp_sso_url}
                  onChange={(e) => update("saml_idp_sso_url", e.target.value)}
                />
              </div>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="saml_sp_entity_id">SP Entity ID</Label>
                  <Input
                    id="saml_sp_entity_id"
                    value={form.saml_sp_entity_id}
                    onChange={(e) => update("saml_sp_entity_id", e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="saml_name_id_format">NameID Format</Label>
                  <Select
                    value={form.saml_name_id_format || ""}
                    onValueChange={(v) => update("saml_name_id_format", v)}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select format" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress">Email</SelectItem>
                      <SelectItem value="urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified">Unspecified</SelectItem>
                      <SelectItem value="urn:oasis:names:tc:SAML:2.0:nameid-format:persistent">Persistent</SelectItem>
                      <SelectItem value="urn:oasis:names:tc:SAML:2.0:nameid-format:transient">Transient</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="saml_idp_cert">
                  IdP Certificate (PEM){isEdit && " (leave blank to keep current)"}
                </Label>
                <textarea
                  id="saml_idp_cert"
                  className="flex min-h-[80px] w-full rounded-md border border-input bg-transparent px-3 py-2 text-sm shadow-sm placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                  placeholder="-----BEGIN CERTIFICATE-----"
                  value={form.saml_idp_cert}
                  onChange={(e) => update("saml_idp_cert", e.target.value)}
                />
              </div>
            </fieldset>
          )}

          {/* LDAP fields */}
          {form.protocol === "ldap" && (
            <fieldset className="space-y-4 rounded-lg border border-border p-4">
              <legend className="px-2 text-sm font-medium">LDAP Settings</legend>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <div className="space-y-2 sm:col-span-2">
                  <Label htmlFor="ldap_host">Host</Label>
                  <Input
                    id="ldap_host"
                    placeholder="ldap.example.com"
                    value={form.ldap_host}
                    onChange={(e) => update("ldap_host", e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="ldap_port">Port</Label>
                  <Input
                    id="ldap_port"
                    type="number"
                    value={form.ldap_port}
                    onChange={(e) => update("ldap_port", parseInt(e.target.value) || 389)}
                  />
                </div>
              </div>
              <div className="flex items-center gap-2">
                <input
                  id="ldap_use_tls"
                  type="checkbox"
                  className="h-4 w-4 rounded border-input"
                  checked={form.ldap_use_tls}
                  onChange={(e) => update("ldap_use_tls", e.target.checked)}
                />
                <Label htmlFor="ldap_use_tls">Use TLS (STARTTLS)</Label>
              </div>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="ldap_bind_dn">Bind DN</Label>
                  <Input
                    id="ldap_bind_dn"
                    placeholder="cn=admin,dc=example,dc=com"
                    value={form.ldap_bind_dn}
                    onChange={(e) => update("ldap_bind_dn", e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="ldap_bind_password">
                    Bind Password{isEdit && " (leave blank to keep current)"}
                  </Label>
                  <Input
                    id="ldap_bind_password"
                    type="password"
                    value={form.ldap_bind_password}
                    onChange={(e) => update("ldap_bind_password", e.target.value)}
                  />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="ldap_base_dn">Base DN</Label>
                <Input
                  id="ldap_base_dn"
                  placeholder="ou=users,dc=example,dc=com"
                  value={form.ldap_base_dn}
                  onChange={(e) => update("ldap_base_dn", e.target.value)}
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="ldap_user_filter">User Filter</Label>
                <Input
                  id="ldap_user_filter"
                  placeholder="(uid=%s)"
                  value={form.ldap_user_filter}
                  onChange={(e) => update("ldap_user_filter", e.target.value)}
                />
              </div>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
                <div className="space-y-2">
                  <Label htmlFor="ldap_email_attr">Email Attribute</Label>
                  <Input
                    id="ldap_email_attr"
                    placeholder="mail"
                    value={form.ldap_email_attr}
                    onChange={(e) => update("ldap_email_attr", e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="ldap_name_attr">Name Attribute</Label>
                  <Input
                    id="ldap_name_attr"
                    placeholder="cn"
                    value={form.ldap_name_attr}
                    onChange={(e) => update("ldap_name_attr", e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="ldap_username_attr">Username Attribute</Label>
                  <Input
                    id="ldap_username_attr"
                    placeholder="uid"
                    value={form.ldap_username_attr}
                    onChange={(e) => update("ldap_username_attr", e.target.value)}
                  />
                </div>
              </div>
            </fieldset>
          )}

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
