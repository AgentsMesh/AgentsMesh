"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useAuthStore } from "@/stores/auth";
import { ticketApi, podApi, runnerApi } from "@/lib/api";

interface DashboardStats {
  totalTickets: number;
  openTickets: number;
  activePods: number;
  onlineRunners: number;
}

export default function OrganizationDashboard() {
  const { currentOrg } = useAuthStore();
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

  return (
    <div className="space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-foreground">
          Welcome to {currentOrg?.name || "Dashboard"}
        </h1>
        <p className="text-muted-foreground mt-1">
          Here&apos;s an overview of your workspace
        </p>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard
          title="Total Tickets"
          value={stats.totalTickets}
          href={currentOrg ? `/${currentOrg.slug}/tickets` : "#"}
          icon={
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2" />
            </svg>
          }
        />
        <StatCard
          title="Open Tickets"
          value={stats.openTickets}
          href={currentOrg ? `/${currentOrg.slug}/tickets?status=open` : "#"}
          icon={
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          }
          highlight={stats.openTickets > 0}
        />
        <StatCard
          title="Active Pods"
          value={stats.activePods}
          href={currentOrg ? `/${currentOrg.slug}/workspace` : "#"}
          icon={
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
          }
        />
        <StatCard
          title="Online Runners"
          value={stats.onlineRunners}
          href={currentOrg ? `/${currentOrg.slug}/settings/runners` : "#"}
          icon={
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2" />
            </svg>
          }
          highlight={stats.onlineRunners === 0}
          highlightColor="warning"
        />
      </div>

      {/* Quick Actions */}
      <div>
        <h2 className="text-lg font-semibold text-foreground mb-4">Quick Actions</h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <QuickActionCard
            title="Create Ticket"
            description="Create a new task or issue for your team"
            href={currentOrg ? `/${currentOrg.slug}/tickets/new` : "#"}
            icon={
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
            }
          />
          <QuickActionCard
            title="View Mesh"
            description="Monitor your AI agent network"
            href={currentOrg ? `/${currentOrg.slug}/mesh` : "#"}
            icon={
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            }
          />
          <QuickActionCard
            title="Manage Repositories"
            description="Connect and manage your code repositories"
            href={currentOrg ? `/${currentOrg.slug}/repositories` : "#"}
            icon={
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
              </svg>
            }
          />
        </div>
      </div>

      {/* Getting Started (shown when no runners) */}
      {stats.onlineRunners === 0 && (
        <div className="p-6 border border-amber-200 bg-amber-50 dark:bg-amber-950/20 dark:border-amber-800 rounded-lg">
          <div className="flex items-start gap-4">
            <div className="p-2 bg-amber-100 dark:bg-amber-900/50 rounded-lg">
              <svg className="w-6 h-6 text-amber-600 dark:text-amber-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
              </svg>
            </div>
            <div className="flex-1">
              <h3 className="font-semibold text-amber-800 dark:text-amber-200">No Runners Connected</h3>
              <p className="text-sm text-amber-700 dark:text-amber-300 mt-1">
                You need at least one online Runner to execute AI agent tasks. Set up a Runner to get started.
              </p>
              <Link
                href={currentOrg ? `/${currentOrg.slug}/settings/runners` : "#"}
                className="inline-flex items-center gap-2 mt-3 text-sm font-medium text-amber-700 dark:text-amber-300 hover:underline"
              >
                Set up a Runner
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
                </svg>
              </Link>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// Stat Card Component
function StatCard({
  title,
  value,
  href,
  icon,
  highlight = false,
  highlightColor = "primary",
}: {
  title: string;
  value: number;
  href: string;
  icon: React.ReactNode;
  highlight?: boolean;
  highlightColor?: "primary" | "warning";
}) {
  const highlightStyles = {
    primary: "border-primary/50 bg-primary/5",
    warning: "border-amber-500/50 bg-amber-50 dark:bg-amber-950/20",
  };

  return (
    <Link
      href={href}
      className={`block p-4 border rounded-lg transition-colors hover:border-primary/50 ${
        highlight ? highlightStyles[highlightColor] : "border-border"
      }`}
    >
      <div className="flex items-center justify-between">
        <div className="text-muted-foreground">{icon}</div>
        <span className="text-2xl font-bold text-foreground">{value}</span>
      </div>
      <p className="mt-2 text-sm text-muted-foreground">{title}</p>
    </Link>
  );
}

// Quick Action Card Component
function QuickActionCard({
  title,
  description,
  href,
  icon,
}: {
  title: string;
  description: string;
  href: string;
  icon: React.ReactNode;
}) {
  return (
    <Link
      href={href}
      className="block p-4 border border-border rounded-lg transition-all hover:border-primary/50 hover:shadow-sm"
    >
      <div className="flex items-start gap-3">
        <div className="p-2 bg-primary/10 text-primary rounded-lg">{icon}</div>
        <div>
          <h3 className="font-medium text-foreground">{title}</h3>
          <p className="text-sm text-muted-foreground mt-1">{description}</p>
        </div>
      </div>
    </Link>
  );
}
