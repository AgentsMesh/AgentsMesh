"use client";

import { useState, useEffect } from "react";
import { runnerApi } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { AlertCircle, Check, Copy, Terminal } from "lucide-react";
import { useServerUrl } from "@/hooks/useServerUrl";
import { useTranslations } from "next-intl";

interface AddRunnerModalProps {
  open: boolean;
  onClose: () => void;
  onCreated?: () => void;
}

/**
 * AddRunnerModal - Modal for generating runner registration token
 *
 * Extracted from runners/page.tsx to be reusable across the application
 */
export function AddRunnerModal({ open, onClose, onCreated }: AddRunnerModalProps) {
  const t = useTranslations();
  const serverUrl = useServerUrl();
  const [loading, setLoading] = useState(false);
  const [generatedToken, setGeneratedToken] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  // Reset state when modal closes
  useEffect(() => {
    if (!open) {
      setGeneratedToken(null);
      setLoading(false);
      setCopied(false);
    }
  }, [open]);

  if (!open) return null;

  const handleGenerate = async () => {
    setLoading(true);
    try {
      const res = await runnerApi.createToken();
      setGeneratedToken(res.token);
    } catch (error) {
      console.error("Failed to generate token:", error);
    } finally {
      setLoading(false);
    }
  };

  const copyToken = () => {
    if (generatedToken) {
      navigator.clipboard.writeText(generatedToken);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  const copyCommand = () => {
    if (generatedToken) {
      const command = `agentsmesh-runner register --server ${serverUrl} --token ${generatedToken}\nagentsmesh-runner run`;
      navigator.clipboard.writeText(command);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

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
      <div className="bg-background border border-border rounded-lg w-full max-w-lg p-4 md:p-6">
        <h2 className="text-lg md:text-xl font-semibold mb-2">
          {t("runners.addRunnerModal.title")}
        </h2>
        <p className="text-sm text-muted-foreground mb-4">
          {t("runners.addRunnerModal.subtitle")}
        </p>

        {generatedToken ? (
          <div className="space-y-4">
            {/* Warning */}
            <div className="flex items-start gap-2 p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
              <AlertCircle className="w-5 h-5 text-yellow-600 dark:text-yellow-400 flex-shrink-0 mt-0.5" />
              <p className="text-sm text-yellow-800 dark:text-yellow-200">
                {t("runners.addRunnerModal.tokenWarning")}
              </p>
            </div>

            {/* Token display */}
            <div>
              <label className="block text-sm font-medium mb-2">
                {t("runners.addRunnerModal.tokenLabel")}
              </label>
              <div className="flex gap-2">
                <code className="flex-1 p-3 bg-muted rounded text-sm break-all font-mono">
                  {generatedToken}
                </code>
                <Button variant="outline" size="sm" onClick={copyToken} className="flex-shrink-0">
                  {copied ? (
                    <Check className="w-4 h-4 text-green-500 dark:text-green-400" />
                  ) : (
                    <Copy className="w-4 h-4" />
                  )}
                </Button>
              </div>
            </div>

            {/* Usage instructions */}
            <div>
              <label className="block text-sm font-medium mb-2">
                {t("runners.addRunnerModal.usageTitle")}
              </label>
              <div className="bg-muted rounded-lg p-4 relative">
                <div className="flex items-center gap-2 text-muted-foreground text-xs mb-2">
                  <Terminal className="w-4 h-4" />
                  <span>Terminal</span>
                </div>
                <code className="text-green-600 dark:text-green-400 text-sm font-mono block whitespace-pre-wrap">
                  {`agentsmesh-runner register --server ${serverUrl} --token ${generatedToken.substring(0, 16)}...
agentsmesh-runner run`}
                </code>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={copyCommand}
                  className="absolute top-2 right-2 h-7 text-xs text-muted-foreground hover:text-foreground"
                >
                  {t("runners.addRunnerModal.copyCommand")}
                </Button>
              </div>
            </div>

            <div className="flex justify-end pt-2">
              <Button onClick={handleDone}>{t("runners.addRunnerModal.done")}</Button>
            </div>
          </div>
        ) : (
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              {t("runners.addRunnerModal.generateHint")}
            </p>

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
