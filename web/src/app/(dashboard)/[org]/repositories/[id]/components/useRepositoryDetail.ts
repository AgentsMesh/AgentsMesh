"use client";

import { useState, useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";
import { repositoryApi, RepositoryData } from "@/lib/api";
import { useTranslations } from "next-intl";
import { useConfirmDialog } from "@/components/ui/confirm-dialog";

export type RepositoryTab = "info" | "branches";

export interface UseRepositoryDetailResult {
  repository: RepositoryData | null;
  branches: string[];
  loading: boolean;
  loadingBranches: boolean;
  activeTab: RepositoryTab;
  showEditModal: boolean;
  deleteDialog: ReturnType<typeof useConfirmDialog>;
  setActiveTab: (tab: RepositoryTab) => void;
  setShowEditModal: (show: boolean) => void;
  loadRepository: () => Promise<void>;
  handleDelete: () => Promise<void>;
}

export function useRepositoryDetail(repositoryId: number): UseRepositoryDetailResult {
  const t = useTranslations();
  const router = useRouter();

  const [repository, setRepository] = useState<RepositoryData | null>(null);
  const [branches, setBranches] = useState<string[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingBranches, setLoadingBranches] = useState(false);
  const [activeTab, setActiveTab] = useState<RepositoryTab>("info");
  const [showEditModal, setShowEditModal] = useState(false);

  const deleteDialog = useConfirmDialog({
    title: t("repositories.detail.deleteDialog.title"),
    description: t("repositories.detail.deleteDialog.description"),
    confirmText: t("common.delete"),
    variant: "destructive",
  });

  const loadRepository = useCallback(async () => {
    try {
      const res = await repositoryApi.get(repositoryId);
      setRepository(res.repository);
    } catch (error) {
      console.error("Failed to load repository:", error);
    } finally {
      setLoading(false);
    }
  }, [repositoryId]);

  useEffect(() => {
    loadRepository();
  }, [loadRepository]);

  const loadBranches = useCallback(async () => {
    if (!repository) return;
    setLoadingBranches(true);
    try {
      // Note: Branch listing requires access token which should come from user's Git connection
      setBranches([]);
    } catch (error) {
      console.error("Failed to load branches:", error);
    } finally {
      setLoadingBranches(false);
    }
  }, [repository]);

  const handleDelete = useCallback(async () => {
    if (!repository) return;
    const confirmed = await deleteDialog.confirm();
    if (!confirmed) return;
    try {
      await repositoryApi.delete(repositoryId);
      router.push("../repositories");
    } catch (error) {
      console.error("Failed to delete repository:", error);
    }
  }, [repository, repositoryId, router, deleteDialog]);

  useEffect(() => {
    if (activeTab === "branches" && branches.length === 0 && repository) {
      loadBranches();
    }
  }, [activeTab, branches.length, repository, loadBranches]);

  return {
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
  };
}
