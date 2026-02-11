"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function MeshPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.features.mesh.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.features.mesh.description")}
      </p>

      {/* Overview */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.mesh.overview.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed">
          {t("docs.features.mesh.overview.description")}
        </p>
      </section>

      {/* Topology Visualization */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.mesh.visualization.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.features.mesh.visualization.description")}
        </p>
        <ul className="list-disc list-inside text-muted-foreground space-y-2">
          <li>{t("docs.features.mesh.visualization.nodes")}</li>
          <li>{t("docs.features.mesh.visualization.edges")}</li>
          <li>{t("docs.features.mesh.visualization.colors")}</li>
          <li>{t("docs.features.mesh.visualization.animation")}</li>
        </ul>
      </section>

      {/* Node Colors & Status */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.mesh.nodeStatus.title")}
        </h2>
        <div className="space-y-4">
          <div className="flex items-start gap-4">
            <div className="w-28 text-sm font-medium text-green-500 dark:text-green-400">
              {t("docs.features.mesh.nodeStatus.running")}
            </div>
            <p className="text-muted-foreground">
              {t("docs.features.mesh.nodeStatus.runningDesc")}
            </p>
          </div>
          <div className="flex items-start gap-4">
            <div className="w-28 text-sm font-medium text-yellow-500 dark:text-yellow-400">
              {t("docs.features.mesh.nodeStatus.paused")}
            </div>
            <p className="text-muted-foreground">
              {t("docs.features.mesh.nodeStatus.pausedDesc")}
            </p>
          </div>
          <div className="flex items-start gap-4">
            <div className="w-28 text-sm font-medium text-orange-500 dark:text-orange-400">
              {t("docs.features.mesh.nodeStatus.disconnected")}
            </div>
            <p className="text-muted-foreground">
              {t("docs.features.mesh.nodeStatus.disconnectedDesc")}
            </p>
          </div>
          <div className="flex items-start gap-4">
            <div className="w-28 text-sm font-medium text-muted-foreground">
              {t("docs.features.mesh.nodeStatus.completed")}
            </div>
            <p className="text-muted-foreground">
              {t("docs.features.mesh.nodeStatus.completedDesc")}
            </p>
          </div>
          <div className="flex items-start gap-4">
            <div className="w-28 text-sm font-medium text-red-500 dark:text-red-400">
              {t("docs.features.mesh.nodeStatus.error")}
            </div>
            <p className="text-muted-foreground">
              {t("docs.features.mesh.nodeStatus.errorDesc")}
            </p>
          </div>
        </div>
      </section>

      {/* Pod Binding Mechanism */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.mesh.binding.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.features.mesh.binding.description")}
        </p>

        <h3 className="text-lg font-medium mb-3">
          {t("docs.features.mesh.binding.scopes")}
        </h3>
        <ul className="list-disc list-inside text-muted-foreground space-y-2 mb-6">
          <li>
            <code className="bg-muted px-1 rounded">
              {t("docs.features.mesh.binding.terminalRead")}
            </code>
          </li>
          <li>
            <code className="bg-muted px-1 rounded">
              {t("docs.features.mesh.binding.terminalWrite")}
            </code>
          </li>
        </ul>

        <h3 className="text-lg font-medium mb-3">
          {t("docs.features.mesh.binding.flow")}
        </h3>
        <ol className="list-decimal list-inside text-muted-foreground space-y-2">
          <li>{t("docs.features.mesh.binding.flowStep1")}</li>
          <li>{t("docs.features.mesh.binding.flowStep2")}</li>
          <li>{t("docs.features.mesh.binding.flowStep3")}</li>
          <li>{t("docs.features.mesh.binding.flowStep4")}</li>
        </ol>
      </section>

      {/* Channel Integration */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.mesh.channelIntegration.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed">
          {t("docs.features.mesh.channelIntegration.description")}
        </p>
      </section>

      {/* Real-Time Monitoring */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.mesh.monitoring.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.features.mesh.monitoring.description")}
        </p>
        <ul className="list-disc list-inside text-muted-foreground space-y-2">
          <li>{t("docs.features.mesh.monitoring.item1")}</li>
          <li>{t("docs.features.mesh.monitoring.item2")}</li>
          <li>{t("docs.features.mesh.monitoring.item3")}</li>
          <li>{t("docs.features.mesh.monitoring.item4")}</li>
        </ul>
      </section>

      <DocNavigation />
    </div>
  );
}
