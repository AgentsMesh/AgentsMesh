"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { cn } from "@/lib/utils";
import { useCurrentOrg, useAuthOrganizations, useAuthStore } from "@/stores/auth";
import { useIDEStore } from "@/stores/ide";
import { Check, Plus, Building2 } from "lucide-react";
import { useTranslations } from "next-intl";

export function OrgSwitcher() {
  const router = useRouter();
  const t = useTranslations();
  const currentOrg = useCurrentOrg();
  const organizations = useAuthOrganizations();
  const setCurrentOrg = useAuthStore((s) => s.setCurrentOrg);
  const activeActivity = useIDEStore((s) => s.activeActivity);

  const [open, setOpen] = useState(false);
  const wrapRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const onDocClick = (ev: MouseEvent) => {
      if (wrapRef.current && !wrapRef.current.contains(ev.target as Node)) {
        setOpen(false);
      }
    };
    document.addEventListener("mousedown", onDocClick);
    return () => document.removeEventListener("mousedown", onDocClick);
  }, [open]);

  const handleSwitch = useCallback(
    (slug: string) => {
      const next = organizations.find((o) => o.slug === slug);
      if (!next) return;
      setCurrentOrg(next);
      setOpen(false);
      const dest = activeActivity ?? "workspace";
      router.push(`/${next.slug}/${dest}`);
    },
    [organizations, setCurrentOrg, activeActivity, router],
  );

  const initial = currentOrg?.name?.charAt(0)?.toUpperCase() ?? "?";

  return (
    <div ref={wrapRef} className="relative">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-label={currentOrg?.name ?? "Organization"}
        className={cn(
          "flex h-9 w-9 items-center justify-center rounded-lg text-[15px] font-semibold text-primary-foreground transition-colors",
          "bg-primary hover:bg-primary-hover",
          open && "ring-2 ring-primary/30 ring-offset-1 ring-offset-background",
        )}
      >
        {initial}
      </button>

      {open && (
        <div className="absolute left-11 top-0 z-50 w-64 overflow-hidden rounded-md border border-border bg-popover shadow-lg">
          <div className="border-b border-border px-3 py-2">
            <div className="text-[10px] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
              {t("ide.orgSwitcher.title")}
            </div>
            {currentOrg && (
              <div className="truncate text-sm font-medium text-foreground">{currentOrg.name}</div>
            )}
          </div>

          <ul className="max-h-80 overflow-y-auto py-1">
            {organizations.length === 0 ? (
              <li className="px-3 py-3 text-xs text-muted-foreground">{t("ide.orgSwitcher.empty")}</li>
            ) : (
              organizations.map((org) => {
                const active = org.id === currentOrg?.id;
                return (
                  <li key={org.id}>
                    <button
                      type="button"
                      onClick={() => handleSwitch(org.slug)}
                      className={cn(
                        "flex w-full items-center gap-2.5 px-3 py-2 text-left text-sm transition-colors",
                        active ? "bg-accent/50 text-foreground" : "text-foreground hover:bg-muted",
                      )}
                    >
                      <span className="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-md bg-primary/10 text-xs font-semibold text-primary">
                        {org.name.charAt(0).toUpperCase()}
                      </span>
                      <span className="flex-1 truncate">{org.name}</span>
                      {active && <Check className="h-4 w-4 flex-shrink-0 text-primary" />}
                    </button>
                  </li>
                );
              })
            )}
          </ul>

          <div className="border-t border-border p-1">
            <button
              type="button"
              onClick={() => {
                setOpen(false);
                router.push("/onboarding/create-org");
              }}
              className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm text-foreground transition-colors hover:bg-muted"
            >
              <Plus className="h-4 w-4 text-muted-foreground" />
              {t("ide.orgSwitcher.create")}
            </button>
            <button
              type="button"
              onClick={() => {
                setOpen(false);
                if (currentOrg) router.push(`/${currentOrg.slug}/settings`);
              }}
              className="flex w-full items-center gap-2 rounded-md px-3 py-2 text-sm text-foreground transition-colors hover:bg-muted"
            >
              <Building2 className="h-4 w-4 text-muted-foreground" />
              {t("ide.orgSwitcher.manage")}
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

export default OrgSwitcher;
