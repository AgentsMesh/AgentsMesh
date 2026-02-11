"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function TicketsPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.features.tickets.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.features.tickets.description")}
      </p>

      {/* Overview */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.tickets.overview.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.features.tickets.overview.description")}
        </p>
        <ul className="list-disc list-inside text-muted-foreground space-y-2">
          <li>{t("docs.features.tickets.overview.item1")}</li>
          <li>{t("docs.features.tickets.overview.item2")}</li>
          <li>{t("docs.features.tickets.overview.item3")}</li>
          <li>{t("docs.features.tickets.overview.item4")}</li>
          <li>{t("docs.features.tickets.overview.item5")}</li>
        </ul>
      </section>

      {/* Ticket Types */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.tickets.types.title")}
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.tickets.types.task")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.tickets.types.taskDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.tickets.types.bug")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.tickets.types.bugDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.tickets.types.feature")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.tickets.types.featureDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.tickets.types.epic")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.tickets.types.epicDesc")}
            </p>
          </div>
        </div>
      </section>

      {/* Ticket Status */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.tickets.status.title")}
        </h2>
        <p className="text-muted-foreground mb-4">
          {t("docs.features.tickets.status.description")}
        </p>
        <div className="flex flex-wrap gap-2">
          <span className="px-3 py-1 bg-muted rounded text-sm">
            {t("docs.features.tickets.status.backlog")}
          </span>
          <span className="text-muted-foreground">&rarr;</span>
          <span className="px-3 py-1 bg-muted rounded text-sm">
            {t("docs.features.tickets.status.todo")}
          </span>
          <span className="text-muted-foreground">&rarr;</span>
          <span className="px-3 py-1 bg-blue-500/20 text-blue-600 dark:text-blue-400 rounded text-sm">
            {t("docs.features.tickets.status.inProgress")}
          </span>
          <span className="text-muted-foreground">&rarr;</span>
          <span className="px-3 py-1 bg-yellow-500/20 text-yellow-600 dark:text-yellow-400 rounded text-sm">
            {t("docs.features.tickets.status.inReview")}
          </span>
          <span className="text-muted-foreground">&rarr;</span>
          <span className="px-3 py-1 bg-green-500/20 text-green-600 dark:text-green-400 rounded text-sm">
            {t("docs.features.tickets.status.done")}
          </span>
        </div>
        <p className="text-sm text-muted-foreground mt-4">
          {t("docs.features.tickets.status.cancelled")}
        </p>
      </section>

      {/* Priority */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.tickets.priority.title")}
        </h2>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">
                  {t("docs.features.tickets.priority.priorityHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.features.tickets.priority.descriptionHeader")}
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border text-red-500 dark:text-red-400 font-medium">
                  {t("docs.features.tickets.priority.urgent")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.features.tickets.priority.urgentDesc")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border text-orange-500 dark:text-orange-400 font-medium">
                  {t("docs.features.tickets.priority.high")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.features.tickets.priority.highDesc")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border text-yellow-500 dark:text-yellow-400 font-medium">
                  {t("docs.features.tickets.priority.medium")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.features.tickets.priority.mediumDesc")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border text-muted-foreground font-medium">
                  {t("docs.features.tickets.priority.low")}
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.features.tickets.priority.lowDesc")}
                </td>
              </tr>
              <tr>
                <td className="p-3 font-medium">
                  {t("docs.features.tickets.priority.none")}
                </td>
                <td className="p-3">
                  {t("docs.features.tickets.priority.noneDesc")}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* Pod Integration */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.tickets.podIntegration.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.features.tickets.podIntegration.description")}
        </p>
        <ul className="list-disc list-inside text-muted-foreground space-y-2">
          <li>
            <strong>{t("docs.features.tickets.podIntegration.context").split(" — ")[0]}</strong>
            {" — "}
            {t("docs.features.tickets.podIntegration.context").split(" — ")[1]}
          </li>
          <li>
            <strong>{t("docs.features.tickets.podIntegration.progress").split(" — ")[0]}</strong>
            {" — "}
            {t("docs.features.tickets.podIntegration.progress").split(" — ")[1]}
          </li>
          <li>
            <strong>{t("docs.features.tickets.podIntegration.autoUpdate").split(" — ")[0]}</strong>
            {" — "}
            {t("docs.features.tickets.podIntegration.autoUpdate").split(" — ")[1]}
          </li>
          <li>
            <strong>{t("docs.features.tickets.podIntegration.history").split(" — ")[0]}</strong>
            {" — "}
            {t("docs.features.tickets.podIntegration.history").split(" — ")[1]}
          </li>
        </ul>
      </section>

      {/* Git Integration */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.tickets.gitIntegration.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.features.tickets.gitIntegration.description")}
        </p>
        <div className="space-y-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.tickets.gitIntegration.commits")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.tickets.gitIntegration.commitsDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.features.tickets.gitIntegration.mrs")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.features.tickets.gitIntegration.mrsDesc")}
            </p>
          </div>
        </div>
      </section>

      {/* Estimation */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.features.tickets.storyPoints.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.features.tickets.storyPoints.description")}
        </p>
        <div className="flex flex-wrap gap-2">
          {[1, 2, 3, 5, 8, 13, 21].map((point) => (
            <span
              key={point}
              className="w-10 h-10 flex items-center justify-center bg-muted rounded text-sm font-medium"
            >
              {point}
            </span>
          ))}
        </div>
      </section>

      <DocNavigation />
    </div>
  );
}
