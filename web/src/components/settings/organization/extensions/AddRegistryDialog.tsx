"use client";

import { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { extensionApi } from "@/lib/api";
import type { SkillRegistryAuthType } from "@/lib/api/extension";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { toast } from "sonner";
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogBody, DialogFooter } from "@/components/ui/dialog";
import type { TranslationFn } from "../GeneralSettings";

// Supported agents for the compatible_agents field
const SUPPORTED_AGENTS = [
  { slug: "claude-code", label: "Claude Code" },
  { slug: "gemini-cli", label: "Gemini CLI" },
  { slug: "codex-cli", label: "Codex CLI" },
  { slug: "aider", label: "Aider" },
] as const;

interface AddRegistryDialogProps {
  t: TranslationFn;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onAdded: () => void;
}

export function AddRegistryDialog({ t, open, onOpenChange, onAdded }: AddRegistryDialogProps) {
  const [addUrl, setAddUrl] = useState("");
  const [addBranch, setAddBranch] = useState("");
  const [addType, setAddType] = useState("");
  const [addCompatibleAgents, setAddCompatibleAgents] = useState<string[]>(["claude-code"]);
  const [addAuthType, setAddAuthType] = useState<SkillRegistryAuthType>("none");
  const [addAuthCredential, setAddAuthCredential] = useState("");
  const [adding, setAdding] = useState(false);

  const resetForm = useCallback(() => {
    setAddUrl("");
    setAddBranch("");
    setAddType("");
    setAddCompatibleAgents(["claude-code"]);
    setAddAuthType("none");
    setAddAuthCredential("");
  }, []);

  const handleAdd = useCallback(async () => {
    if (!addUrl.trim()) return;
    setAdding(true);
    try {
      await extensionApi.createSkillRegistry({
        repository_url: addUrl.trim(),
        branch: addBranch.trim() || undefined,
        source_type: addType.trim() || undefined,
        compatible_agents: addCompatibleAgents.length > 0 ? addCompatibleAgents : undefined,
        auth_type: addAuthType !== "none" ? addAuthType : undefined,
        auth_credential: addAuthCredential.trim() || undefined,
      });
      toast.success(t("extensions.sourceAdded"));
      onOpenChange(false);
      resetForm();
      onAdded();
    } catch (error) {
      toast.error(getLocalizedErrorMessage(error, t, t("extensions.failedToAddSource")));
    } finally {
      setAdding(false);
    }
  }, [addUrl, addBranch, addType, addCompatibleAgents, addAuthType, addAuthCredential, t, onAdded, onOpenChange, resetForm]);

  return (
    <Dialog open={open} onOpenChange={(o) => { onOpenChange(o); if (!o) resetForm(); }}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("extensions.skillRegistries.addSource")}</DialogTitle>
        </DialogHeader>
        <DialogBody>
          <div className="space-y-4">
            {/* Repository URL */}
            <div>
              <label className="text-sm font-medium mb-1 block">
                {t("extensions.repoUrl")} <span className="text-destructive">*</span>
              </label>
              <Input
                placeholder="https://github.com/owner/skills-repo"
                value={addUrl}
                onChange={(e) => setAddUrl(e.target.value)}
              />
            </div>

            {/* Branch */}
            <div>
              <label className="text-sm font-medium mb-1 block">
                {t("extensions.branch")}
              </label>
              <Input
                placeholder="main"
                value={addBranch}
                onChange={(e) => setAddBranch(e.target.value)}
              />
            </div>

            {/* Source Type */}
            <div>
              <label className="text-sm font-medium mb-1 block">
                {t("extensions.skillRegistries.sourceType")}
              </label>
              <Input
                placeholder={t("extensions.skillRegistries.sourceTypePlaceholder")}
                value={addType}
                onChange={(e) => setAddType(e.target.value)}
              />
            </div>

            {/* Compatible Agents */}
            <div>
              <label className="text-sm font-medium mb-1 block">
                {t("extensions.skillRegistries.compatibleAgents")}
              </label>
              <p className="text-xs text-muted-foreground mb-2">
                {t("extensions.skillRegistries.compatibleAgentsHint")}
              </p>
              <div className="flex flex-wrap gap-2">
                {SUPPORTED_AGENTS.map((agent) => {
                  const isSelected = addCompatibleAgents.includes(agent.slug);
                  return (
                    <Button
                      key={agent.slug}
                      type="button"
                      variant={isSelected ? "default" : "outline"}
                      size="sm"
                      onClick={() => {
                        setAddCompatibleAgents((prev) =>
                          isSelected
                            ? prev.filter((s) => s !== agent.slug)
                            : [...prev, agent.slug]
                        );
                      }}
                    >
                      {agent.label}
                    </Button>
                  );
                })}
              </div>
            </div>

            {/* Authentication */}
            <div className="border-t border-border pt-4">
              <label className="text-sm font-medium mb-1 block">
                {t("extensions.skillRegistries.authentication")}
              </label>
              <p className="text-xs text-muted-foreground mb-2">
                {t("extensions.skillRegistries.authenticationHint")}
              </p>
              <select
                className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
                value={addAuthType}
                onChange={(e) => {
                  setAddAuthType(e.target.value as SkillRegistryAuthType);
                  setAddAuthCredential("");
                }}
              >
                <option value="none">{t("extensions.skillRegistries.authNone")}</option>
                <option value="github_pat">{t("extensions.skillRegistries.authGitHubPAT")}</option>
                <option value="gitlab_pat">{t("extensions.skillRegistries.authGitLabPAT")}</option>
                <option value="ssh_key">{t("extensions.skillRegistries.authSSHKey")}</option>
              </select>

              {addAuthType !== "none" && (
                <div className="mt-3">
                  <label className="text-sm font-medium mb-1 block">
                    {addAuthType === "ssh_key"
                      ? t("extensions.skillRegistries.sshKeyLabel")
                      : t("extensions.skillRegistries.patLabel")}
                  </label>
                  {addAuthType === "ssh_key" ? (
                    <textarea
                      className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm font-mono min-h-[100px] resize-y"
                      placeholder={t("extensions.skillRegistries.sshKeyPlaceholder")}
                      value={addAuthCredential}
                      onChange={(e) => setAddAuthCredential(e.target.value)}
                      autoComplete="off"
                    />
                  ) : (
                    <Input
                      type="password"
                      placeholder={t("extensions.skillRegistries.patPlaceholder")}
                      value={addAuthCredential}
                      onChange={(e) => setAddAuthCredential(e.target.value)}
                      autoComplete="off"
                    />
                  )}
                </div>
              )}
            </div>
          </div>
        </DialogBody>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("common.cancel")}
          </Button>
          <Button disabled={adding || !addUrl.trim()} onClick={handleAdd}>
            {adding ? t("extensions.adding") : t("extensions.skillRegistries.addSource")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
