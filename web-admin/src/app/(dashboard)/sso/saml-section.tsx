import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { ProtocolSectionProps } from "./sso-form-types";

export function SAMLSection({ form, update, isEdit }: ProtocolSectionProps) {
  return (
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
  );
}
