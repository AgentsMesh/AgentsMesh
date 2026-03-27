import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { ProtocolSectionProps } from "./sso-form-types";

export function LDAPSection({ form, update, isEdit }: ProtocolSectionProps) {
  return (
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
      <LDAPSearchFields form={form} update={update} />
    </fieldset>
  );
}

function LDAPSearchFields({ form, update }: Omit<ProtocolSectionProps, "isEdit">) {
  return (
    <>
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
    </>
  );
}
