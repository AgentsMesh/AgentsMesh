"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { useCurrentUser, useCurrentOrg, useAuthStore } from "@/stores/auth";
import { useTranslations } from "next-intl";

interface AuthButtonsProps {
  /** Button size variant */
  size?: "sm" | "default";
  /** Style variant for the console button when logged in */
  consoleVariant?: "primary" | "outline";
  /** Whether to show the "Get Started" (register) button when logged out */
  showRegister?: boolean;
  /** Additional callback when a link is clicked (e.g. close mobile menu) */
  onClick?: () => void;
  /** Additional className for the container */
  className?: string;
}

/**
 * Shared auth-aware action buttons.
 * - Logged in: shows "Console" button linking to /{org}/workspace
 * - Logged out: shows "Sign In" (and optionally "Get Started")
 *
 * Single source of truth for auth CTA — used by landing Navbar, docs header,
 * PageHeader, and any other public-facing page header.
 */
export function AuthButtons({
  size = "default",
  consoleVariant = "primary",
  showRegister = false,
  onClick,
  className,
}: AuthButtonsProps) {
  const user = useCurrentUser();
  const currentOrg = useCurrentOrg();
  const _hasHydrated = useAuthStore((s) => s._hasHydrated);
  const isLoggedIn = _hasHydrated && !!user;
  const consoleHref = currentOrg?.slug
    ? `/${currentOrg.slug}/workspace`
    : "/login";
  const t = useTranslations();

  // Avoid flash during hydration — render nothing until store is ready
  if (!_hasHydrated) return null;

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
