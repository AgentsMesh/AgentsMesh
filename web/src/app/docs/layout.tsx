"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetTrigger,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { cn } from "@/lib/utils";
import { useTranslations } from "next-intl";
import { docsNavSections, getBreadcrumbs } from "@/lib/docs-navigation";

function SidebarNav({
  onNavigate,
}: {
  onNavigate?: () => void;
}) {
  const pathname = usePathname();
  const t = useTranslations();

  return (
    <nav className="space-y-6">
      {docsNavSections.map((section) => (
        <div key={section.titleKey}>
          <h3 className="font-semibold text-sm mb-2">
            {t(section.titleKey)}
          </h3>
          <ul className="space-y-1">
            {section.items.map((item) => (
              <li key={item.href}>
                <Link
                  href={item.href}
                  onClick={onNavigate}
                  className={cn(
                    "text-sm block py-1 transition-colors",
                    pathname === item.href
                      ? "text-primary font-medium"
                      : "text-muted-foreground hover:text-foreground"
                  )}
                >
                  {t(item.titleKey)}
                </Link>
              </li>
            ))}
          </ul>
        </div>
      ))}
    </nav>
  );
}

export default function DocsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const t = useTranslations();
  const [mobileOpen, setMobileOpen] = useState(false);
  const breadcrumbs = getBreadcrumbs(pathname);

  return (
    <div className="min-h-screen bg-background">
      {/* Header */}
      <header className="border-b border-border sticky top-0 bg-background z-10">
        <div className="container mx-auto px-4 py-4 flex items-center justify-between">
          <div className="flex items-center gap-2">
            {/* Mobile hamburger */}
            <Sheet open={mobileOpen} onOpenChange={setMobileOpen}>
              <SheetTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="md:hidden"
                  aria-label={t("docs.nav.menu")}
                >
                  <svg
                    className="w-5 h-5"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M4 6h16M4 12h16M4 18h16"
                    />
                  </svg>
                </Button>
              </SheetTrigger>
              <SheetContent side="left" className="w-72 p-4 pt-6">
                <SheetHeader className="mb-4">
                  <SheetTitle>{t("docs.title")}</SheetTitle>
                </SheetHeader>
                <SidebarNav onNavigate={() => setMobileOpen(false)} />
              </SheetContent>
            </Sheet>

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
          </div>
          <div className="flex items-center gap-4">
            <Link
              href="/docs"
              className="text-sm text-muted-foreground hover:text-foreground"
            >
              {t("landing.nav.docs")}
            </Link>
            <Link href="/login">
              <Button variant="outline">{t("landing.nav.signIn")}</Button>
            </Link>
          </div>
        </div>
      </header>

      <div className="flex">
        {/* Desktop Sidebar */}
        <aside className="w-64 border-r border-border min-h-[calc(100vh-65px)] p-4 hidden md:block sticky top-[65px] h-[calc(100vh-65px)] overflow-y-auto">
          <SidebarNav />
        </aside>

        {/* Content */}
        <main className="flex-1 p-4 md:p-8 max-w-4xl min-w-0">
          {/* Breadcrumbs */}
          {breadcrumbs.length > 1 && (
            <nav className="flex items-center gap-1.5 text-sm text-muted-foreground mb-6">
              {breadcrumbs.map((crumb, index) => (
                <span key={index} className="flex items-center gap-1.5">
                  {index > 0 && <span className="text-border">/</span>}
                  {crumb.href ? (
                    <Link
                      href={crumb.href}
                      className="hover:text-foreground transition-colors"
                    >
                      {t(crumb.titleKey)}
                    </Link>
                  ) : (
                    <span>{t(crumb.titleKey)}</span>
                  )}
                </span>
              ))}
            </nav>
          )}

          {children}
        </main>
      </div>

      {/* Footer */}
      <footer className="border-t border-border mt-16">
        <div className="container mx-auto px-4 py-8">
          <div className="flex flex-col md:flex-row justify-between items-center gap-4">
            <p className="text-sm text-muted-foreground">
              &copy; {new Date().getFullYear()} AgentsMesh.{" "}
              {t("common.allRightsReserved")}
            </p>
            <div className="flex gap-6">
              <Link
                href="/privacy"
                className="text-sm text-muted-foreground hover:text-foreground"
              >
                {t("landing.footer.legal.privacy")}
              </Link>
              <Link
                href="/terms"
                className="text-sm text-muted-foreground hover:text-foreground"
              >
                {t("landing.footer.legal.terms")}
              </Link>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
