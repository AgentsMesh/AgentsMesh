"use client";

import { useState, useEffect } from "react";
import { useRouter, useParams } from "next/navigation";
import { runnerApi, type RunnerData } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { CenteredSpinner } from "@/components/ui/spinner";
import { useConfirmDialog, ConfirmDialog } from "@/components/ui/confirm-dialog";
import { Server, Plus, RefreshCw, Power, Cpu, HardDrive } from "lucide-react";
import { getLocalizedErrorMessage } from "@/lib/api/errors";
import { useServerUrl } from "@/hooks/useServerUrl";
import { toast } from "sonner";
import { useTranslations } from "next-intl";
import { StatCard, AddRunnerModal, RunnerConfigModal, RunnerCardList, RunnerTable } from "./components";

export default function RunnersPage() {
  const t = useTranslations();
  const router = useRouter();
  const params = useParams();
  const [runners, setRunners] = useState<RunnerData[]>([]);
  const [latestVersion, setLatestVersion] = useState<string | undefined>();
  const [loading, setLoading] = useState(true);
  const [showAddRunnerModal, setShowAddRunnerModal] = useState(false);
  const [selectedRunner, setSelectedRunner] = useState<RunnerData | null>(null);
  const serverUrl = useServerUrl();

  const deleteDialog = useConfirmDialog({
    title: t("runners.page.deleteDialog.title"),
    description: t("runners.page.deleteDialog.description"),
    confirmText: t("common.delete"),
    variant: "destructive",
  });

  useEffect(() => { loadData(); }, []);

  const loadData = async () => {
    try {
      const runnersRes = await runnerApi.list();
      setRunners(runnersRes.runners || []);
      setLatestVersion(runnersRes.latest_runner_version);
    } catch (error) { console.error("Failed to load data:", error); }
    finally { setLoading(false); }
  };

  const handleToggleEnabled = async (runner: RunnerData) => {
    try { await runnerApi.update(runner.id, { is_enabled: !runner.is_enabled }); loadData(); }
    catch (error) { console.error("Failed to update runner:", error); toast.error(getLocalizedErrorMessage(error, t, t("common.error"))); }
  };

  const handleDeleteRunner = async (runner: RunnerData) => {
    const confirmed = await deleteDialog.confirm();
    if (!confirmed) return;
    try { await runnerApi.delete(runner.id); loadData(); }
    catch (error) { console.error("Failed to delete runner:", error); toast.error(getLocalizedErrorMessage(error, t, t("common.error"))); }
  };

  if (loading) return <CenteredSpinner />;

  const onlineCount = runners.filter((r) => r.status === "online").length;
  const totalPods = runners.reduce((sum, r) => sum + r.current_pods, 0);
  const totalCapacity = runners.reduce((sum, r) => sum + r.max_concurrent_pods, 0);
  const navigateToRunner = (runnerId: number) => router.push(`/${params.org}/runners/${runnerId}`);

  return (
    <div className="p-4 md:p-6 space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-xl md:text-2xl font-bold text-foreground">{t("runners.page.title")}</h1>
          <p className="text-sm text-muted-foreground">{t("runners.page.subtitle")}</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={loadData}>
            <RefreshCw className="w-4 h-4 mr-2" />{t("runners.page.refresh")}
          </Button>
          <Button onClick={() => setShowAddRunnerModal(true)}>
            <Plus className="w-4 h-4 mr-2" />{t("runners.page.addRunner")}
          </Button>
        </div>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3 md:gap-4">
        <StatCard title={t("runners.page.totalRunners")} value={runners.length} icon={<Server className="w-5 h-5" />} />
        <StatCard title={t("runners.page.online")} value={onlineCount} icon={<Power className="w-5 h-5" />} variant="success" />
        <StatCard title={t("runners.page.activePods")} value={totalPods} icon={<Cpu className="w-5 h-5" />} />
        <StatCard title={t("runners.page.totalCapacity")} value={totalCapacity} icon={<HardDrive className="w-5 h-5" />} />
      </div>

      {/* Runners List */}
      <div className="space-y-4">
        <h2 className="text-lg font-semibold">{t("runners.page.activeRunners")}</h2>
        <div className="block md:hidden">
          <RunnerCardList runners={runners} latestVersion={latestVersion} t={t}
            onNavigate={navigateToRunner} onConfigure={setSelectedRunner}
            onToggleEnabled={handleToggleEnabled} onDelete={handleDeleteRunner} />
        </div>
        <div className="hidden md:block">
          <RunnerTable runners={runners} latestVersion={latestVersion} t={t}
            onNavigate={navigateToRunner} onConfigure={setSelectedRunner}
            onToggleEnabled={handleToggleEnabled} onDelete={handleDeleteRunner} />
        </div>
      </div>

      {/* Modals */}
      {showAddRunnerModal && (
        <AddRunnerModal t={t} onClose={() => setShowAddRunnerModal(false)}
          onCreated={() => { setShowAddRunnerModal(false); loadData(); }} serverUrl={serverUrl} />
      )}
      {selectedRunner && (
        <RunnerConfigModal t={t} runner={selectedRunner} onClose={() => setSelectedRunner(null)}
          onUpdated={() => { setSelectedRunner(null); loadData(); }} />
      )}
      <ConfirmDialog {...deleteDialog.dialogProps} />
    </div>
  );
}
