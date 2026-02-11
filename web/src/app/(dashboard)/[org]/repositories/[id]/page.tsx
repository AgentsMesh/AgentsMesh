"use client";

import { useParams } from "next/navigation";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { CenteredSpinner } from "@/components/ui/spinner";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { useTranslations } from "next-intl";
import {
  EditRepositoryModal,
  useRepositoryDetail,
  RepositoryHeader,
  RepositoryInfoCard,
  GitProviderCard,
  WebhookSettingsCard,
  BranchesTab,
  RepositoryTabs,
} from "./components";

export default function RepositoryDetailPage() {
  const t = useTranslations();
  const params = useParams();
  const repositoryId = Number(params.id);

  const {
    repository,
    branches,
    loading,
    loadingBranches,
    activeTab,
    showEditModal,
    deleteDialog,
    setActiveTab,
    setShowEditModal,
    loadRepository,
    handleDelete,
  } = useRepositoryDetail(repositoryId);

  if (loading) {
    return <CenteredSpinner />;
  }

  if (!repository) {
    return (
      <div className="p-6">
        <div className="text-center py-12">
          <p className="text-muted-foreground mb-4">{t("repositories.detail.notFound")}</p>
          <Link href="../repositories">
            <Button variant="outline">{t("repositories.detail.backToList")}</Button>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="p-6">
      <RepositoryHeader
        repository={repository}
        onEdit={() => setShowEditModal(true)}
        onDelete={handleDelete}
      />

      <RepositoryTabs activeTab={activeTab} onTabChange={setActiveTab} />

      {activeTab === "info" && (
        <div className="grid gap-6 md:grid-cols-2">
          <RepositoryInfoCard repository={repository} />
          <GitProviderCard repository={repository} />
          <WebhookSettingsCard
            repository={repository}
            onStatusChange={loadRepository}
          />
        </div>
      )}

      {activeTab === "branches" && (
        <BranchesTab
          branches={branches}
          defaultBranch={repository.default_branch}
          loading={loadingBranches}
        />
      )}

      {showEditModal && (
        <EditRepositoryModal
          repository={repository}
          onClose={() => setShowEditModal(false)}
          onUpdated={() => {
            setShowEditModal(false);
            loadRepository();
          }}
        />
      )}

      <ConfirmDialog {...deleteDialog.dialogProps} />
    </div>
  );
}
