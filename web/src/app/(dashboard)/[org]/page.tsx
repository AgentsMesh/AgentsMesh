"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useAuthStore } from "@/stores/auth";
import { ticketApi, podApi, runnerApi } from "@/lib/api";
import { useTranslations } from "next-intl";
import {
  ClipboardList,
  Clock,
  Terminal,
  Server,
  Plus,
  Zap,
  FolderGit2,
  AlertTriangle,
  ArrowRight,
} from "lucide-react";

interface DashboardStats {
  totalTickets: number;
  openTickets: number;
  activePods: number;
  onlineRunners: number;
}

export default function OrganizationDashboard() {
  const { currentOrg } = useAuthStore();
  const t = useTranslations();
  const [stats, setStats] = useState<DashboardStats>({
    totalTickets: 0,
    openTickets: 0,
    activePods: 0,
    onlineRunners: 0,
  });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchStats = async () => {
      try {
        // Fetch tickets
        const ticketsRes = await ticketApi.list().catch(() => ({ tickets: [] }));
        const tickets = ticketsRes.tickets || [];
        const openTickets = tickets.filter(
          (t: { status: string }) => !["done", "cancelled"].includes(t.status)
        );

        // Fetch pods
        const podsRes = await podApi.list().catch(() => ({ pods: [] }));
        const pods = podsRes.pods || [];
        const activePods = pods.filter(
          (s: { status: string }) => s.status === "running"
        );

        // Fetch runners
        const runnersRes = await runnerApi.list().catch(() => ({ runners: [] }));
        const runners = runnersRes.runners || [];
        const onlineRunners = runners.filter(
          (r: { status: string }) => r.status === "online"
        );

        setStats({
          totalTickets: tickets.length,
          openTickets: openTickets.length,
          activePods: activePods.length,
          onlineRunners: onlineRunners.length,
        });
      } catch (error) {
        console.error("Failed to fetch dashboard stats:", error);
      } finally {
        setLoading(false);
      }
    };

    fetchStats();
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
      </div>
    );
  }

  const orgSlug = currentOrg?.slug || "";

  return (
    <div className="min-h-full">
      {/* Hero Section */}
      <div className="bg-gradient-to-br from-primary/5 via-background to-background px-4 sm:px-6 lg:px-8 py-8 sm:py-12 lg:py-16">
        <div className="max-w-6xl mx-auto">
          <h1 className="text-2xl sm:text-3xl lg:text-4xl font-bold text-foreground tracking-tight">
            {t("dashboard.welcomeWithOrg", { orgName: currentOrg?.name || "" })}
          </h1>
          <p className="text-muted-foreground mt-2 sm:mt-3 text-base sm:text-lg">
            {t("dashboard.overview")}
          </p>
        </div>
      </div>

      <div className="px-4 sm:px-6 lg:px-8 py-6 sm:py-8 lg:py-10">
        <div className="max-w-6xl mx-auto space-y-8 sm:space-y-10 lg:space-y-12">
          {/* Stats Grid - 2x2 on mobile, 4 columns on desktop */}
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4 lg:gap-6">
            <StatCard
              title={t("dashboard.stats.totalTickets")}
              value={stats.totalTickets}
              href={`/${orgSlug}/tickets`}
              icon={<ClipboardList className="w-5 h-5 sm:w-6 sm:h-6" />}
              color="blue"
            />
            <StatCard
              title={t("dashboard.stats.openTickets")}
              value={stats.openTickets}
              href={`/${orgSlug}/tickets?status=open`}
              icon={<Clock className="w-5 h-5 sm:w-6 sm:h-6" />}
              color="amber"
              highlight={stats.openTickets > 0}
            />
            <StatCard
              title={t("dashboard.stats.activePods")}
              value={stats.activePods}
              href={`/${orgSlug}/workspace`}
              icon={<Terminal className="w-5 h-5 sm:w-6 sm:h-6" />}
              color="green"
            />
            <StatCard
              title={t("dashboard.stats.onlineRunners")}
              value={stats.onlineRunners}
              href={`/${orgSlug}/runners`}
              icon={<Server className="w-5 h-5 sm:w-6 sm:h-6" />}
              color="purple"
              highlight={stats.onlineRunners === 0}
              highlightType="warning"
            />
          </div>

          {/* Quick Actions */}
          <section>
            <h2 className="text-lg sm:text-xl font-semibold text-foreground mb-4 sm:mb-6">
              {t("dashboard.quickActions.title")}
            </h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-5 lg:gap-6">
              <QuickActionCard
                title={t("dashboard.quickActions.createTicket")}
                description={t("dashboard.quickActions.createTicketDesc")}
                href={`/${orgSlug}/tickets`}
                icon={<Plus className="w-5 h-5 sm:w-6 sm:h-6" />}
                color="blue"
              />
              <QuickActionCard
                title={t("dashboard.quickActions.viewMesh")}
                description={t("dashboard.quickActions.viewMeshDesc")}
                href={`/${orgSlug}/mesh`}
                icon={<Zap className="w-5 h-5 sm:w-6 sm:h-6" />}
                color="purple"
              />
              <QuickActionCard
                title={t("dashboard.quickActions.manageRepos")}
                description={t("dashboard.quickActions.manageReposDesc")}
                href={`/${orgSlug}/repositories`}
                icon={<FolderGit2 className="w-5 h-5 sm:w-6 sm:h-6" />}
                color="green"
              />
            </div>
          </section>

          {/* No Runners Warning */}
          {stats.onlineRunners === 0 && (
            <section className="p-5 sm:p-6 lg:p-8 border border-amber-200 bg-gradient-to-br from-amber-50 to-amber-50/50 dark:from-amber-950/30 dark:to-amber-950/10 dark:border-amber-800/50 rounded-xl sm:rounded-2xl">
              <div className="flex flex-col sm:flex-row items-start gap-4 sm:gap-5">
                <div className="p-3 bg-amber-100 dark:bg-amber-900/50 rounded-xl shrink-0">
                  <AlertTriangle className="w-6 h-6 sm:w-7 sm:h-7 text-amber-600 dark:text-amber-400" />
                </div>
                <div className="flex-1 min-w-0">
                  <h3 className="font-semibold text-amber-800 dark:text-amber-200 text-base sm:text-lg">
                    {t("dashboard.noRunners.title")}
                  </h3>
                  <p className="text-sm sm:text-base text-amber-700 dark:text-amber-300/90 mt-1.5 sm:mt-2 leading-relaxed">
                    {t("dashboard.noRunners.description")}
                  </p>
                  <Link
                    href={`/${orgSlug}/runners`}
                    className="inline-flex items-center gap-2 mt-4 text-sm sm:text-base font-medium text-amber-700 dark:text-amber-300 hover:text-amber-900 dark:hover:text-amber-100 transition-colors group"
                  >
                    {t("dashboard.noRunners.setup")}
                    <ArrowRight className="w-4 h-4 transition-transform group-hover:translate-x-0.5" />
                  </Link>
                </div>
              </div>
            </section>
          )}
        </div>
      </div>
    </div>
  );
}

// Stat Card Component with improved design
function StatCard({
  title,
  value,
  href,
  icon,
  color,
  highlight = false,
  highlightType = "primary",
}: {
  title: string;
  value: number;
  href: string;
  icon: React.ReactNode;
  color: "blue" | "amber" | "green" | "purple";
  highlight?: boolean;
  highlightType?: "primary" | "warning";
}) {
  const colorStyles = {
    blue: {
      icon: "text-blue-600 dark:text-blue-400",
      bg: "bg-blue-50 dark:bg-blue-950/30",
      border: "border-blue-100 dark:border-blue-900/50",
    },
    amber: {
      icon: "text-amber-600 dark:text-amber-400",
      bg: "bg-amber-50 dark:bg-amber-950/30",
      border: "border-amber-100 dark:border-amber-900/50",
    },
    green: {
      icon: "text-green-600 dark:text-green-400",
      bg: "bg-green-50 dark:bg-green-950/30",
      border: "border-green-100 dark:border-green-900/50",
    },
    purple: {
      icon: "text-purple-600 dark:text-purple-400",
      bg: "bg-purple-50 dark:bg-purple-950/30",
      border: "border-purple-100 dark:border-purple-900/50",
    },
  };

  const highlightStyles = {
    primary: "ring-2 ring-primary/20",
    warning: "ring-2 ring-amber-400/30",
  };

  const styles = colorStyles[color];

  return (
    <Link
      href={href}
      className={`
        relative block p-4 sm:p-5 lg:p-6
        bg-card border rounded-xl sm:rounded-2xl
        transition-all duration-200
        hover:shadow-md hover:border-primary/30 hover:-translate-y-0.5
        ${highlight ? highlightStyles[highlightType] : "border-border"}
      `}
    >
      <div className="flex items-start justify-between gap-3">
        <div className={`p-2 sm:p-2.5 rounded-lg sm:rounded-xl ${styles.bg}`}>
          <div className={styles.icon}>{icon}</div>
        </div>
        <span className="text-2xl sm:text-3xl lg:text-4xl font-bold text-foreground tabular-nums">
          {value}
        </span>
      </div>
      <p className="mt-3 sm:mt-4 text-xs sm:text-sm font-medium text-muted-foreground">
        {title}
      </p>
    </Link>
  );
}

// Quick Action Card Component with improved design
function QuickActionCard({
  title,
  description,
  href,
  icon,
  color,
}: {
  title: string;
  description: string;
  href: string;
  icon: React.ReactNode;
  color: "blue" | "purple" | "green";
}) {
  const colorStyles = {
    blue: {
      icon: "text-blue-600 dark:text-blue-400",
      bg: "bg-blue-50 dark:bg-blue-950/30",
      hover: "group-hover:bg-blue-100 dark:group-hover:bg-blue-900/40",
    },
    purple: {
      icon: "text-purple-600 dark:text-purple-400",
      bg: "bg-purple-50 dark:bg-purple-950/30",
      hover: "group-hover:bg-purple-100 dark:group-hover:bg-purple-900/40",
    },
    green: {
      icon: "text-green-600 dark:text-green-400",
      bg: "bg-green-50 dark:bg-green-950/30",
      hover: "group-hover:bg-green-100 dark:group-hover:bg-green-900/40",
    },
  };

  const styles = colorStyles[color];

  return (
    <Link
      href={href}
      className="group block p-5 sm:p-6 bg-card border border-border rounded-xl sm:rounded-2xl transition-all duration-200 hover:shadow-md hover:border-primary/30 hover:-translate-y-0.5"
    >
      <div className="flex items-start gap-4">
        <div className={`p-3 rounded-xl transition-colors ${styles.bg} ${styles.hover}`}>
          <div className={styles.icon}>{icon}</div>
        </div>
        <div className="flex-1 min-w-0">
          <h3 className="font-semibold text-foreground text-base sm:text-lg group-hover:text-primary transition-colors">
            {title}
          </h3>
          <p className="text-sm text-muted-foreground mt-1.5 leading-relaxed line-clamp-2">
            {description}
          </p>
        </div>
      </div>
    </Link>
  );
}
