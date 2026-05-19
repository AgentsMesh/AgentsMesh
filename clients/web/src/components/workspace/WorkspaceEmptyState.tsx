"use client";

import { useState, useEffect } from "react";
import { useTranslations } from "next-intl";
import { X } from "lucide-react";
import { RegisterLocalRunnerCard } from "@/components/onboarding/RegisterLocalRunnerCard";

interface RecipeCardProps {
  emoji: string;
  title: string;
  description: string;
  agents: string;
  duration: string;
  onClick?: () => void;
}

function RecipeCard({ emoji, title, description, agents, duration, onClick }: RecipeCardProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="group flex-1 rounded-[10px] border border-border bg-card p-3.5 text-left transition-colors hover:bg-muted"
    >
      <div className="flex flex-col gap-1.5">
        <div className="flex items-center gap-2">
          <span className="text-sm leading-none">{emoji}</span>
          <span className="text-[13px] font-semibold text-foreground">{title}</span>
        </div>
        <p className="text-[11px] leading-4 text-muted-foreground">{description}</p>
        <div className="flex items-center gap-1.5 pt-1">
          <span className="rounded-sm bg-muted px-1.5 py-0.5 font-mono text-[10px] font-medium text-muted-foreground">
            {agents}
          </span>
          <span className="text-[10px] text-muted-foreground/70">· {duration}</span>
        </div>
      </div>
    </button>
  );
}

interface WorkspaceEmptyStateProps {
  onCreatePod: () => void;
}

export function WorkspaceEmptyState({ onCreatePod }: WorkspaceEmptyStateProps) {
  const t = useTranslations();
  const [showBanner, setShowBanner] = useState(true);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "n") {
        const target = e.target as HTMLElement | null;
        const tag = target?.tagName;
        if (tag === "INPUT" || tag === "TEXTAREA" || target?.isContentEditable) return;
        e.preventDefault();
        onCreatePod();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onCreatePod]);

  return (
    <div className="flex h-full flex-col bg-background">
      {showBanner && (
        <div className="flex items-center gap-2.5 bg-accent px-6 py-2.5 text-[13px]">
          <span className="text-sm leading-none">👋</span>
          <span className="font-medium text-accent-foreground">{t("workspace.banner.newUser")}</span>
          <a href="#" className="text-primary hover:underline">
            {t("workspace.banner.watchIntro")}
          </a>
          <div className="flex-1" />
          <button
            type="button"
            onClick={() => setShowBanner(false)}
            className="text-accent-foreground/80 hover:text-accent-foreground"
            aria-label="Dismiss"
          >
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      )}

      <div className="flex flex-1 flex-col items-center justify-center gap-8 px-6 py-10">
        <div className="flex w-[520px] max-w-full">
          <RegisterLocalRunnerCard />
        </div>
        <div className="flex w-[520px] max-w-full flex-col items-center gap-4 text-center">
          <div className="flex h-20 w-20 items-center justify-center rounded-2xl border border-primary/40 bg-accent">
            <span className="font-mono text-[32px] font-semibold leading-none text-primary">{">_"}</span>
          </div>
          <h1 className="text-2xl font-semibold text-foreground">
            {t("workspace.emptyHeroTitle")}
          </h1>
          <p className="max-w-[460px] text-sm leading-[22px] text-muted-foreground">
            {t("workspace.emptyHeroDescription")}
          </p>
          <div className="flex items-center gap-2.5 pt-3">
            <button
              type="button"
              onClick={onCreatePod}
              className="flex h-10 items-center gap-2 rounded-lg bg-primary px-5 text-sm font-semibold text-primary-foreground shadow-sm hover:bg-primary-hover"
            >
              <span className="text-base leading-none">+</span>
              {t("workspace.createNewPod")}
            </button>
          </div>
        </div>

        <div className="flex w-[720px] max-w-full flex-col gap-2.5">
          <div className="text-center text-[11px] font-semibold uppercase tracking-[0.12em] text-muted-foreground/80">
            {t("workspace.recipesHeading")}
          </div>
          <div className="flex gap-3">
            <RecipeCard
              emoji="🔍"
              title={t("workspace.recipes.explain.title")}
              description={t("workspace.recipes.explain.description")}
              agents="claude-code"
              duration={t("workspace.recipes.explain.duration")}
              onClick={onCreatePod}
            />
            <RecipeCard
              emoji="🧪"
              title={t("workspace.recipes.tests.title")}
              description={t("workspace.recipes.tests.description")}
              agents="claude-code"
              duration={t("workspace.recipes.tests.duration")}
              onClick={onCreatePod}
            />
            <RecipeCard
              emoji="🐛"
              title={t("workspace.recipes.bug.title")}
              description={t("workspace.recipes.bug.description")}
              agents="codex · acp"
              duration={t("workspace.recipes.bug.duration")}
              onClick={onCreatePod}
            />
          </div>
        </div>
      </div>

      <div className="flex items-center justify-between border-t border-border px-6 py-4">
        <div className="flex items-center gap-5 font-mono text-xs text-muted-foreground">
          <span>⌘K  {t("workspace.hints.search")}</span>
          <span>⌘N  {t("workspace.hints.createPod")}</span>
        </div>
      </div>
    </div>
  );
}
