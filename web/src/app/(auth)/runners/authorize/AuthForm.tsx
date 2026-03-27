import Link from "next/link";
import { Button } from "@/components/ui/button";
import type { OrganizationData } from "@/lib/api/organization";

interface AuthFormProps {
  authToken: string | null;
  userEmail?: string;
  authKey: string;
  organizations: OrganizationData[] | null;
  selectedOrg: OrganizationData | null;
  onSelectOrg: (org: OrganizationData | null) => void;
  nodeIdInput: string;
  onNodeIdChange: (val: string) => void;
  authorizing: boolean;
  onAuthorize: () => void;
  error: string;
  t: (key: string, params?: Record<string, string | number>) => string;
  tCommon: (key: string) => string;
}

export function AuthForm({
  authToken, userEmail, authKey, organizations, selectedOrg,
  onSelectOrg, nodeIdInput, onNodeIdChange, authorizing, onAuthorize,
  error, t, tCommon,
}: AuthFormProps) {
  return (
    <div className="p-6 border border-border rounded-lg space-y-4">
      {/* Runner Icon */}
      <div className="flex justify-center">
        <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
          <svg className="w-8 h-8 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
              d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01" />
          </svg>
        </div>
      </div>

      <div className="text-center space-y-2">
        <h1 className="text-xl font-semibold text-foreground">{t("title")}</h1>
        <p className="text-sm text-muted-foreground">{t("description")}</p>
      </div>

      {error && (
        <div className="p-3 text-sm text-destructive bg-destructive/10 rounded-md">{error}</div>
      )}

      {!authToken || !userEmail ? (
        <UnauthenticatedPrompt authKey={authKey} t={t} />
      ) : (
        <AuthenticatedForm
          userEmail={userEmail} organizations={organizations} selectedOrg={selectedOrg}
          onSelectOrg={onSelectOrg} nodeIdInput={nodeIdInput} onNodeIdChange={onNodeIdChange}
          authorizing={authorizing} onAuthorize={onAuthorize} t={t} tCommon={tCommon} />
      )}
    </div>
  );
}

function UnauthenticatedPrompt({ authKey, t }: {
  authKey: string; t: (k: string, p?: Record<string, string | number>) => string;
}) {
  return (
    <div className="space-y-3">
      <p className="text-sm text-center text-muted-foreground">{t("signInRequired")}</p>
      <Link href={`/login?redirect=/runners/authorize?key=${authKey}`}>
        <Button className="w-full">{t("signInToAuthorize")}</Button>
      </Link>
      <p className="text-sm text-center text-muted-foreground">
        {t("noAccount")}{" "}
        <Link href={`/register?redirect=/runners/authorize?key=${authKey}`}
          className="text-primary hover:underline">{t("signUp")}</Link>
      </p>
    </div>
  );
}

function AuthenticatedForm({ userEmail, organizations, selectedOrg, onSelectOrg,
  nodeIdInput, onNodeIdChange, authorizing, onAuthorize, t, tCommon }: {
  userEmail: string; organizations: OrganizationData[] | null;
  selectedOrg: OrganizationData | null;
  onSelectOrg: (org: OrganizationData | null) => void;
  nodeIdInput: string; onNodeIdChange: (val: string) => void;
  authorizing: boolean; onAuthorize: () => void;
  t: (k: string, p?: Record<string, string | number>) => string;
  tCommon: (k: string) => string;
}) {
  return (
    <div className="space-y-4">
      <p className="text-sm text-center text-muted-foreground">
        {t("signedInAs")} <strong>{userEmail}</strong>
      </p>

      {organizations && organizations.length > 0 ? (
        <div className="space-y-2">
          <label className="text-sm font-medium text-foreground">{t("selectOrganization")}</label>
          <select className="w-full px-3 py-2 border border-input rounded-md bg-background text-foreground"
            value={selectedOrg?.id || ""} onChange={(e) => {
              const org = organizations.find((o) => o.id === parseInt(e.target.value));
              onSelectOrg(org || null);
            }}>
            <option value="" disabled>{t("selectOrgPlaceholder")}</option>
            {organizations.map((org) => (
              <option key={org.id} value={org.id}>{org.name}</option>
            ))}
          </select>
        </div>
      ) : (
        <div className="p-3 text-sm text-amber-600 dark:text-amber-400 bg-amber-100 dark:bg-amber-900/30 rounded-md">
          {t("noOrganizations")}
        </div>
      )}

      <div className="space-y-2">
        <label className="text-sm font-medium text-foreground">
          {t("nodeIdLabel")} <span className="text-muted-foreground">({tCommon("optional")})</span>
        </label>
        <input type="text"
          className="w-full px-3 py-2 border border-input rounded-md bg-background text-foreground placeholder:text-muted-foreground"
          placeholder={t("nodeIdPlaceholder")} value={nodeIdInput}
          onChange={(e) => onNodeIdChange(e.target.value)} />
        <p className="text-xs text-muted-foreground">{t("nodeIdHint")}</p>
      </div>

      <Button className="w-full" onClick={onAuthorize} disabled={authorizing || !selectedOrg}>
        {authorizing ? t("authorizing") : t("authorizeButton")}
      </Button>
    </div>
  );
}
