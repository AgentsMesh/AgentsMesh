"use client";

import Link from "next/link";
import { Button } from "@/components/ui/button";
import { useAuthStore } from "@/stores/auth";
import { useTranslations } from "next-intl";

export function PageHeader() {
  const { token, _hasHydrated } = useAuthStore();
  const isLoggedIn = _hasHydrated && !!token;
  const t = useTranslations();

  return (
    <header className="border-b border-border">
      <div className="container mx-auto px-4 py-4 flex items-center justify-between">
        <Link href="/" className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-lg bg-primary flex items-center justify-center">
            <svg
              className="w-5 h-5 text-primary-foreground"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
              />
            </svg>
          </div>
          <span className="text-xl font-bold">AgentsMesh</span>
        </Link>
        {isLoggedIn ? (
          <Link href="/mesh">
            <Button className="bg-primary text-primary-foreground hover:bg-primary/90">
              {t("landing.nav.console")}
            </Button>
          </Link>
        ) : (
          <Link href="/login">
            <Button variant="outline">{t("landing.nav.signIn")}</Button>
          </Link>
        )}
      </div>
    </header>
  );
}
