"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { extensionApi, SkillMarketItem } from "@/lib/api";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogBody, DialogFooter } from "@/components/ui/dialog";
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { ExternalLink, Search, Upload } from "lucide-react";

interface AddSkillDialogProps {
  repositoryId: number;
  scope: "org" | "user";
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onInstalled: () => void;
  installedSlugs?: Set<string>;
}

export function AddSkillDialog({ repositoryId, scope, open, onOpenChange, onInstalled, installedSlugs }: AddSkillDialogProps) {
  const t = useTranslations();
  const [installing, setInstalling] = useState(false);

  // Marketplace state
  const [marketSkills, setMarketSkills] = useState<SkillMarketItem[]>([]);
  const [marketQuery, setMarketQuery] = useState("");
  const [loadingMarket, setLoadingMarket] = useState(false);

  // GitHub form state
  const [githubUrl, setGithubUrl] = useState("");
  const [githubBranch, setGithubBranch] = useState("");
  const [githubPath, setGithubPath] = useState("");

  // Upload state
  const fileInputRef = useRef<HTMLInputElement>(null);

  const loadMarketSkills = useCallback(async (query?: string) => {
    setLoadingMarket(true);
    try {
      const res = await extensionApi.listMarketSkills(query);
      setMarketSkills(res.skills || []);
    } catch (error) {
      console.error("Failed to load market skills:", error);
    } finally {
      setLoadingMarket(false);
    }
  }, []);

  useEffect(() => {
    if (open) {
      loadMarketSkills();
    }
  }, [open, loadMarketSkills]);

  const handleSearchMarket = useCallback(() => {
    loadMarketSkills(marketQuery || undefined);
  }, [marketQuery, loadMarketSkills]);

  const handleInstallFromMarket = useCallback(
    async (item: SkillMarketItem) => {
      setInstalling(true);
      try {
        await extensionApi.installSkillFromMarket(repositoryId, {
          market_item_id: item.id,
          scope,
        });
        toast.success(t("extensions.installed"));
        onInstalled();
      } catch (error) {
        toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToInstall")));
      } finally {
        setInstalling(false);
      }
    },
    [repositoryId, scope, t, onInstalled]
  );

  const handleInstallFromGitHub = useCallback(async () => {
    if (!githubUrl.trim()) return;
    setInstalling(true);
    try {
      await extensionApi.installSkillFromGitHub(repositoryId, {
        url: githubUrl.trim(),
        branch: githubBranch.trim() || undefined,
        path: githubPath.trim() || undefined,
        scope,
      });
      toast.success(t("extensions.installed"));
      onInstalled();
    } catch (error) {
      toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToInstall")));
    } finally {
      setInstalling(false);
    }
  }, [repositoryId, githubUrl, githubBranch, githubPath, scope, t, onInstalled]);

  const handleUpload = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0];
      if (!file) return;
      setInstalling(true);
      try {
        await extensionApi.installSkillFromUpload(repositoryId, file, scope);
        toast.success(t("extensions.installed"));
        onInstalled();
      } catch (error) {
        toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToInstall")));
      } finally {
        setInstalling(false);
        // Reset file input
        if (fileInputRef.current) {
          fileInputRef.current.value = "";
        }
      }
    },
    [repositoryId, scope, t, onInstalled]
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{t("extensions.addSkill")}</DialogTitle>
        </DialogHeader>
        <DialogBody>
          <Tabs defaultValue="market">
            <TabsList className="mb-4">
              <TabsTrigger value="market">{t("extensions.marketplace")}</TabsTrigger>
              <TabsTrigger value="github">{t("extensions.githubRepo")}</TabsTrigger>
              <TabsTrigger value="upload">{t("extensions.upload")}</TabsTrigger>
            </TabsList>

            {/* Marketplace tab */}
            <TabsContent value="market">
              <div className="flex gap-2 mb-4">
                <div className="relative flex-1">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
                  <Input
                    className="pl-9"
                    placeholder={t("extensions.searchSkills")}
                    value={marketQuery}
                    onChange={(e) => setMarketQuery(e.target.value)}
                    onKeyDown={(e) => e.key === "Enter" && handleSearchMarket()}
                  />
                </div>
                <Button variant="outline" onClick={handleSearchMarket}>
                  {t("extensions.search")}
                </Button>
              </div>
              {loadingMarket ? (
                <div className="py-8 text-center">
                  <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-primary mx-auto"></div>
                </div>
              ) : marketSkills.length === 0 ? (
                <p className="text-sm text-muted-foreground text-center py-8">
                  {t("extensions.noMarketSkills")}
                </p>
              ) : (
                <div className="space-y-2 max-h-80 overflow-y-auto">
                  {marketSkills.map((item) => {
                    const isInstalled = installedSlugs?.has(item.slug) ?? false;
                    return (
                      <div
                        key={item.id}
                        className="border border-border rounded-lg p-3 flex items-center justify-between gap-3"
                      >
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <span className="font-medium">{item.display_name || item.slug}</span>
                            {item.category && (
                              <Badge variant="outline" className="text-xs">{item.category}</Badge>
                            )}
                            {isInstalled && (
                              <Badge variant="secondary" className="text-xs">{t("extensions.alreadyInstalled")}</Badge>
                            )}
                            {item.registry?.repository_url && (
                              <a
                                href={item.registry.repository_url}
                                target="_blank"
                                rel="noopener noreferrer"
                                className="text-muted-foreground hover:text-foreground shrink-0"
                                title={t("extensions.viewSource")}
                                onClick={(e) => e.stopPropagation()}
                              >
                                <ExternalLink className="w-3.5 h-3.5" />
                              </a>
                            )}
                          </div>
                          {item.description && (
                            <p className="text-xs text-muted-foreground mt-1 line-clamp-2">
                              {item.description}
                            </p>
                          )}
                        </div>
                        <Button
                          size="sm"
                          disabled={installing || isInstalled}
                          onClick={() => handleInstallFromMarket(item)}
                        >
                          {isInstalled ? t("extensions.alreadyInstalled") : t("extensions.install")}
                        </Button>
                      </div>
                    );
                  })}
                </div>
              )}
            </TabsContent>

            {/* GitHub tab */}
            <TabsContent value="github">
              <div className="space-y-4">
                <div>
                  <label className="text-sm font-medium mb-1 block">
                    {t("extensions.repoUrl")}
                  </label>
                  <Input
                    placeholder="https://github.com/owner/repo"
                    value={githubUrl}
                    onChange={(e) => setGithubUrl(e.target.value)}
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="text-sm font-medium mb-1 block">
                      {t("extensions.branch")}
                    </label>
                    <Input
                      placeholder="main"
                      value={githubBranch}
                      onChange={(e) => setGithubBranch(e.target.value)}
                    />
                  </div>
                  <div>
                    <label className="text-sm font-medium mb-1 block">
                      {t("extensions.path")}
                    </label>
                    <Input
                      placeholder="/"
                      value={githubPath}
                      onChange={(e) => setGithubPath(e.target.value)}
                    />
                  </div>
                </div>
                <Button
                  className="w-full"
                  disabled={installing || !githubUrl.trim()}
                  onClick={handleInstallFromGitHub}
                >
                  {installing ? t("extensions.installing") : t("extensions.install")}
                </Button>
              </div>
            </TabsContent>

            {/* Upload tab */}
            <TabsContent value="upload">
              <div className="border-2 border-dashed border-border rounded-lg p-8 text-center">
                <Upload className="w-8 h-8 mx-auto mb-3 text-muted-foreground" />
                <p className="text-sm text-muted-foreground mb-3">
                  {t("extensions.uploadHint")}
                </p>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".tar.gz,.tgz,.zip"
                  className="hidden"
                  onChange={handleUpload}
                />
                <Button
                  variant="outline"
                  disabled={installing}
                  onClick={() => fileInputRef.current?.click()}
                >
                  {installing ? t("extensions.uploading") : t("extensions.selectFile")}
                </Button>
              </div>
            </TabsContent>
          </Tabs>
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
