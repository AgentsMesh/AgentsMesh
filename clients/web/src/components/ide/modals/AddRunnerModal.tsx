"use client";

import { useState, useEffect } from "react";
import { getRunnerService } from "@/lib/wasm-core";
import { isApiErrorCode, getLocalizedErrorMessage } from "@/lib/api/errors";
import { Button } from "@/components/ui/button";
import { AlertCircle, Check, Copy, Terminal, ShieldAlert } from "lucide-react";
import { useServerUrl } from "@/hooks/useServerUrl";
import { useTranslations } from "next-intl";

interface AddRunnerModalProps {
  open: boolean;
  onClose: () => void;
  onCreated?: () => void;
}

export function AddRunnerModal({ open, onClose, onCreated }: AddRunnerModalProps) {
  const t = useTranslations();
  const serverUrl = useServerUrl();
  const [loading, setLoading] = useState(false);
  const [generatedToken, setGeneratedToken] = useState<string | null>(null);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) {
      setGeneratedToken(null);
      setLoading(false);
      setCopiedKey(null);
      setError(null);
    }
  }, [open]);

  if (!open) return null;

  const handleGenerate = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = JSON.parse(await getRunnerService().create_token(JSON.stringify({})));
      setGeneratedToken(res.token);
    } catch (err) {
      if (isApiErrorCode(err, "ADMIN_REQUIRED") || isApiErrorCode(err, "INSUFFICIENT_PERMISSIONS")) {
        setError(t("apiErrors.INSUFFICIENT_PERMISSIONS"));
      } else {
        setError(getLocalizedErrorMessage(err, t, t("apiErrors.INTERNAL_ERROR")));
      }
    } finally {
      setLoading(false);
    }
  };

  const copyText = (text: string, key: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  const CopyBtn = ({ text, id, className }: { text: string; id: string; className?: string }) => (
    <Button
      variant="ghost"
      size="sm"
      onClick={() => copyText(text, id)}
      className={className ?? "absolute top-2 right-2 h-7 text-xs text-muted-foreground hover:text-foreground"}
    >
      {copiedKey === id ? <Check className="w-3 h-3 text-green-500 dark:text-green-400" /> : t("runners.addRunnerModal.copyCommand")}
    </Button>
  );

  const handleDone = () => {
    setGeneratedToken(null);
    onCreated?.();
    onClose();
  };

  const handleClose = () => {
    setGeneratedToken(null);
    onClose();
  };

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
      <div className="bg-background border border-border rounded-lg w-full max-w-lg p-4 md:p-6 max-h-[90vh] overflow-y-auto">
        <h2 className="text-lg md:text-xl font-semibold mb-2">
          {t("runners.addRunnerModal.title")}
        </h2>
        <p className="text-sm text-muted-foreground mb-4">
          {t("runners.addRunnerModal.subtitle")}
        </p>

        {generatedToken ? (
          <div className="space-y-4">
            <div className="flex items-start gap-2 p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
              <AlertCircle className="w-5 h-5 text-yellow-600 dark:text-yellow-400 flex-shrink-0 mt-0.5" />
              <p className="text-sm text-yellow-800 dark:text-yellow-200">
                {t("runners.addRunnerModal.tokenWarning")}
              </p>
            </div>

            <div>
              <label className="block text-sm font-medium mb-2">
                {t("runners.addRunnerModal.tokenLabel")}
              </label>
              <div className="flex gap-2">
                <code className="flex-1 p-3 bg-muted rounded text-sm break-all font-mono">
                  {generatedToken}
                </code>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => copyText(generatedToken, "token")}
                  className="flex-shrink-0"
                >
                  {copiedKey === "token" ? (
                    <Check className="w-4 h-4 text-green-500 dark:text-green-400" />
                  ) : (
                    <Copy className="w-4 h-4" />
                  )}
                </Button>
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium mb-1">
                {t("runners.addRunnerModal.installTitle")}
              </label>
              <div className="space-y-2">
                <p className="text-xs text-muted-foreground">{t("runners.addRunnerModal.installHint")}</p>
                <div className="bg-muted rounded-lg p-3 relative">
                  <p className="text-xs text-muted-foreground mb-1"># macOS / Linux</p>
                  <code className="text-sm font-mono text-foreground block pr-24">
                    {`curl -fsSL ${serverUrl}/install.sh | sh`}
                  </code>
                  <CopyBtn text={`curl -fsSL ${serverUrl}/install.sh | sh`} id="install-mac" />
                </div>
                <div className="bg-muted rounded-lg p-3 relative">
                  <p className="text-xs text-muted-foreground mb-1"># Windows (PowerShell)</p>
                  <code className="text-sm font-mono text-foreground block pr-24">
                    {`irm ${serverUrl}/install.ps1 | iex`}
                  </code>
                  <CopyBtn text={`irm ${serverUrl}/install.ps1 | iex`} id="install-win" />
                </div>
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium mb-2">
                {t("runners.addRunnerModal.usageTitle")}
              </label>
              <div className="bg-muted rounded-lg p-4 relative">
                <div className="flex items-center gap-2 text-muted-foreground text-xs mb-2">
                  <Terminal className="w-4 h-4" />
                  <span>Terminal</span>
                </div>
                <code className="text-green-600 dark:text-green-400 text-sm font-mono block whitespace-pre-wrap pr-24">
                  {`agentsmesh-runner register --server ${serverUrl} --token ${generatedToken.substring(0, 16)}...
agentsmesh-runner run`}
                </code>
                <CopyBtn
                  text={`agentsmesh-runner register --server ${serverUrl} --token ${generatedToken}\nagentsmesh-runner run`}
                  id="command"
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium mb-1">
                {t("runners.addRunnerModal.serviceTitle")}
              </label>
              <div className="space-y-2">
                <p className="text-xs text-muted-foreground">{t("runners.addRunnerModal.serviceHint")}</p>
                <div className="bg-muted rounded-lg p-4 relative">
                  <div className="flex items-center gap-2 text-muted-foreground text-xs mb-2">
                    <Terminal className="w-4 h-4" />
                    <span>Terminal</span>
                  </div>
                  <code className="text-green-600 dark:text-green-400 text-sm font-mono block whitespace-pre-wrap pr-24">
                    {`agentsmesh-runner service install
agentsmesh-runner service start`}
                  </code>
                  <CopyBtn
                    text={`agentsmesh-runner service install\nagentsmesh-runner service start`}
                    id="service"
                  />
                </div>
              </div>
            </div>

            <div className="flex justify-end pt-2">
              <Button onClick={handleDone}>{t("runners.addRunnerModal.done")}</Button>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            {error ? (
              <div className="flex items-start gap-2 p-3 bg-destructive/10 border border-destructive/20 rounded-lg">
                <ShieldAlert className="w-5 h-5 text-destructive flex-shrink-0 mt-0.5" />
                <p className="text-sm text-destructive">{error}</p>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground">
                {t("runners.addRunnerModal.generateHint")}
              </p>
            )}

            <div className="flex flex-col-reverse sm:flex-row justify-end gap-3 mt-6">
              <Button variant="outline" onClick={handleClose}>
                {t("runners.addRunnerModal.cancel")}
              </Button>
              <Button onClick={handleGenerate} disabled={loading}>
                {loading
                  ? t("runners.addRunnerModal.generating")
                  : t("runners.addRunnerModal.generate")}
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

export default AddRunnerModal;
