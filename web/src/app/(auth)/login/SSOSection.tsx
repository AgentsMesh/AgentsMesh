"use client";

import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { getSSOAuthURL } from "@/lib/api/sso";
import type { SSOConfig } from "@/lib/api/sso";
import { useTranslations } from "next-intl";

interface SSOSectionProps {
  ssoConfigs: SSOConfig[];
  onLdapSubmit: (username: string, password: string) => void;
  ldapLoading: boolean;
}

export function SSOSection({ ssoConfigs, onLdapSubmit, ldapLoading }: SSOSectionProps) {
  const t = useTranslations();
  const [ldapExpanded, setLdapExpanded] = useState(false);
  const [ldapUsername, setLdapUsername] = useState("");
  const [ldapPassword, setLdapPassword] = useState("");

  const ldapConfig = ssoConfigs.find((c) => c.protocol === "ldap");
  const redirectConfigs = ssoConfigs.filter(
    (c) => c.protocol === "oidc" || c.protocol === "saml"
  );

  const handleSSORedirect = (config: SSOConfig) => {
    window.location.assign(getSSOAuthURL(config.domain, config.protocol));
  };

  return (
    <div className="rounded-lg border border-border bg-muted/30 p-4 space-y-3">
      <p className="text-xs font-medium text-muted-foreground text-center uppercase tracking-wide">
        {t("auth.sso.orSignInWithSSO")}
      </p>

      {/* OIDC / SAML redirect buttons */}
      {redirectConfigs.map((config) => (
        <Button
          key={`${config.domain}-${config.protocol}`}
          type="button"
          variant="outline"
          className="w-full"
          onClick={() => handleSSORedirect(config)}
        >
          {t("auth.sso.signInWith", { name: config.name })}
        </Button>
      ))}

      {/* LDAP form */}
      {ldapConfig && (
        <>
          {redirectConfigs.length > 0 && (
            <div className="border-t border-border" />
          )}
          <button
            type="button"
            className="flex w-full items-center justify-between rounded-md px-2 py-1.5 text-sm text-muted-foreground hover:text-foreground transition-colors"
            onClick={() => setLdapExpanded((v) => !v)}
          >
            <span>{t("auth.sso.signInWith", { name: ldapConfig.name })}</span>
            <svg
              className={`h-4 w-4 transition-transform ${ldapExpanded ? "rotate-180" : ""}`}
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M19 9l-7 7-7-7" />
            </svg>
          </button>
          {ldapExpanded && (
            <div className="space-y-3">
              <Input
                id="ldap-username"
                type="text"
                placeholder={t("auth.sso.ldapUsernamePlaceholder")}
                value={ldapUsername}
                onChange={(e) => setLdapUsername(e.target.value)}
              />
              <Input
                id="ldap-password"
                type="password"
                placeholder={t("auth.loginPage.passwordPlaceholder")}
                value={ldapPassword}
                onChange={(e) => setLdapPassword(e.target.value)}
              />
              <Button
                type="button"
                variant="outline"
                className="w-full"
                loading={ldapLoading}
                onClick={() => onLdapSubmit(ldapUsername, ldapPassword)}
              >
                {t("auth.sso.signInWith", { name: ldapConfig.name })}
              </Button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
