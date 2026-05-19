"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { lightCreateOrganization } from "@/lib/light-auth";
import { useRequireLightAuth } from "@/hooks/useRequireLightAuth";
import { useTranslations } from "next-intl";
import { Logo } from "@/components/common";

export default function CreateOrgPage() {
  const router = useRouter();
  const t = useTranslations();
  useRequireLightAuth();
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [slugEdited, setSlugEdited] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [siteHost, setSiteHost] = useState("agentsmesh.dev");

  useEffect(() => {
    setSiteHost(window.location.host);
  }, []);

  useEffect(() => {
    if (!slugEdited && name) {
      const generatedSlug = name
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-+|-+$/g, "")
        .substring(0, 50);
      setSlug(generatedSlug);
    }
  }, [name, slugEdited]);

  const handleSlugChange = (value: string) => {
    setSlugEdited(true);
    const sanitized = value
      .toLowerCase()
      .replace(/[^a-z0-9-]/g, "")
      .substring(0, 50);
    setSlug(sanitized);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!name.trim()) {
      setError(t("auth.onboarding.createOrg.enterWorkspaceName"));
      return;
    }

    if (!slug.trim()) {
      setError(t("auth.onboarding.createOrg.enterUrlIdentifier"));
      return;
    }

    if (slug.length < 3) {
      setError(t("auth.onboarding.createOrg.urlTooShort"));
      return;
    }

    setLoading(true);
    setError("");

    try {
      await lightCreateOrganization({ name: name.trim(), slug: slug.trim() });
      router.push("/onboarding/setup-runner");
    } catch (err) {
      if (err instanceof Error && err.message.includes("already")) {
        setError(t("auth.onboarding.createOrg.urlTaken"));
      } else {
        setError(t("auth.onboarding.createOrg.createFailed"));
      }
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-md space-y-8">
        {/* Header */}
        <div className="text-center">
          <Link href="/" className="inline-flex items-center gap-2">
            <div className="w-10 h-10 rounded-lg overflow-hidden">
              <Logo />
            </div>
            <span className="text-2xl font-bold text-foreground">AgentsMesh</span>
          </Link>
          <h1 className="mt-6 text-2xl font-semibold text-foreground">
            {t("auth.onboarding.createOrg.title")}
          </h1>
          <p className="mt-2 text-sm text-muted-foreground">
            {t("auth.onboarding.createOrg.subtitle")}
          </p>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="space-y-6">
          {error && (
            <div className="p-3 text-sm text-destructive bg-destructive/10 rounded-md">
              {error}
            </div>
          )}

          <div className="space-y-2">
            <label htmlFor="name" className="text-sm font-medium text-foreground">
              {t("auth.onboarding.createOrg.workspaceNameLabel")}
            </label>
            <Input
              id="name"
              placeholder={t("auth.onboarding.createOrg.workspaceNamePlaceholder")}
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
            <p className="text-xs text-muted-foreground">
              {t("auth.onboarding.createOrg.workspaceNameHint")}
            </p>
          </div>

          <div className="space-y-2">
            <label htmlFor="slug" className="text-sm font-medium text-foreground">
              {t("auth.onboarding.createOrg.urlIdentifierLabel")}
            </label>
            <div className="flex items-center gap-2">
              <span className="text-sm text-muted-foreground">{siteHost}/</span>
              <Input
                id="slug"
                placeholder={t("auth.onboarding.createOrg.urlIdentifierPlaceholder")}
                value={slug}
                onChange={(e) => handleSlugChange(e.target.value)}
                className="flex-1"
                required
              />
            </div>
            <p className="text-xs text-muted-foreground">
              {t("auth.onboarding.createOrg.urlIdentifierHint")}
            </p>
          </div>

          <Button type="submit" className="w-full" disabled={loading}>
            {loading ? t("auth.onboarding.creating") : t("auth.onboarding.createOrg.createWorkspace")}
          </Button>
        </form>

        {/* Back Link */}
        <div className="text-center">
          <Link
            href="/onboarding"
            className="text-sm text-muted-foreground hover:text-foreground"
          >
            {t("auth.onboarding.backToOptions")}
          </Link>
        </div>
      </div>
    </div>
  );
}
