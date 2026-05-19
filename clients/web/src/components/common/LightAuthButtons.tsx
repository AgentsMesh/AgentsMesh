"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { useLightSession } from "@/hooks/useLightSession";
import { useTranslations } from "next-intl";

interface LightAuthButtonsProps {
  size?: "sm" | "default";
  consoleVariant?: "primary" | "outline";
  showRegister?: boolean;
  onClick?: () => void;
  className?: string;
}

// Auth-aware CTA for marketing pages — reads PersistedSession from localStorage
// directly instead of going through wasm. Interface matches AuthButtons so
// PageHeader / Navbar can swap imports without changing call sites.
// Use this on routes that must stay wasm-free (/, /docs, /about, /blog, ...).
// Use AuthButtons inside (auth) / (dashboard) where wasm is already loaded.
export function LightAuthButtons({
  size = "default",
  consoleVariant = "primary",
  showRegister = false,
  onClick,
  className,
}: LightAuthButtonsProps) {
  const { session, hydrated } = useLightSession();
  const t = useTranslations();

  if (!hydrated) return null;

  const isLoggedIn = !!session?.isAuthenticated;
  const consoleHref = isLoggedIn && session?.currentOrgSlug
    ? `/${session.currentOrgSlug}/workspace`
    : "/login";

  if (isLoggedIn) {
    return (
      <div className={className}>
        <Link href={consoleHref} onClick={onClick}>
          <Button
            size={size}
            variant={consoleVariant === "outline" ? "outline" : "default"}
            className={
              consoleVariant === "primary"
                ? "bg-primary text-primary-foreground hover:bg-primary/90"
                : undefined
            }
          >
            {t("landing.nav.console")}
          </Button>
        </Link>
      </div>
    );
  }

  return (
    <div className={className}>
      <Link href="/login" onClick={onClick}>
        <Button variant={showRegister ? "ghost" : "outline"} size={size}>
          {t("landing.nav.signIn")}
        </Button>
      </Link>
      {showRegister && (
        <Link href="/register" onClick={onClick}>
          <Button
            size={size}
            className="bg-primary text-primary-foreground hover:bg-primary/90"
          >
            {t("landing.nav.getStarted")}
          </Button>
        </Link>
      )}
    </div>
  );
}
