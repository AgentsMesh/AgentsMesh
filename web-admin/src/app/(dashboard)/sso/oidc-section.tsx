import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { ProtocolSectionProps } from "./sso-form-types";

export function OIDCSection({ form, update, isEdit }: ProtocolSectionProps) {
  return (
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
  );
}
