import { createHashRouter as createBrowserRouter, Navigate, Outlet, useParams } from "react-router-dom";
import { useEffect } from "react";
import { DashboardShell } from "@/pages/layouts/DashboardShell";
import { useAuthStore, useCurrentOrg, useAuthOrganizations, useIsAuthenticated } from "@/stores/auth";
import { RequireAuth } from "@/components/auth/RequireAuth";

import { LoginPage } from "@/pages/auth/login/LoginPage";
import { RegisterPage } from "@/pages/auth/register/RegisterPage";
import { ForgotPasswordPage } from "@/pages/auth/forgot-password/ForgotPasswordPage";
import { ResetPasswordPage } from "@/pages/auth/reset-password/ResetPasswordPage";
import { VerifyEmailPage } from "@/pages/auth/verify-email/VerifyEmailPage";
import { VerifyEmailCallbackPage } from "@/pages/auth/verify-email/callback/VerifyEmailCallbackPage";
import { OAuthCallbackPage } from "@/pages/auth/callback/OAuthCallbackPage";
import { SSOCallbackPage } from "@/pages/auth/sso-callback/SSOCallbackPage";
import { InvitePage } from "@/pages/auth/invite/InvitePage";
import { OnboardingPage } from "@/pages/auth/onboarding/OnboardingPage";
import { CreateOrgPage } from "@/pages/auth/onboarding/create-org/CreateOrgPage";
import { SetupRunnerPage } from "@/pages/auth/onboarding/setup-runner/SetupRunnerPage";
import { LocalRunnerSetupPage } from "@/pages/auth/onboarding/setup-runner/local/LocalRunnerSetupPage";
import { RunnerAuthorizePage } from "@/pages/auth/runners-authorize/RunnerAuthorizePage";

import { WorkspacePage } from "@/pages/dashboard/workspace/WorkspacePage";
import { TicketsPage } from "@/pages/dashboard/tickets/TicketsPage";
import { TicketDetailPage } from "@/pages/dashboard/tickets/TicketDetailPage";
import { ChannelsPage } from "@/pages/dashboard/channels/ChannelsPage";
import { RunnerDetailPage } from "@/pages/dashboard/runner-detail/RunnerDetailPage";
import { LoopsPage } from "@/pages/dashboard/loops/LoopsPage";
import { LoopDetailPage } from "@/pages/dashboard/loop-detail/LoopDetailPage";
import { MeshPage } from "@/pages/dashboard/mesh/MeshPage";
import { BlocksPage } from "@/pages/dashboard/blocks/BlocksPage";
import InfraPage from "@/app/(dashboard)/[org]/infra/page";
import { RepositoryDetailPage } from "@/pages/dashboard/repository-detail/RepositoryDetailPage";
import { SettingsPage } from "@/pages/dashboard/settings/OrgSettingsPage";

import { PersonalSettingsPage } from "@/pages/settings/SettingsRootPage";
import { GeneralSettingsPage } from "@/pages/settings/general/GeneralSettingsPage";
import { GitSettingsPage } from "@/pages/settings/git/GitSettingsPage";
import { PersonalNotificationsPage } from "@/pages/settings/notifications/NotificationsPage";

import { SupportPage } from "@/pages/support/SupportPage";
import { SupportTicketDetailPage } from "@/pages/support/detail/SupportDetailPage";

import { PopoutTerminalPage } from "@/pages/popout/terminal/PopoutTerminalPage";

import { RouteErrorBoundary } from "@/pages/RouteErrorBoundary";

// Workaround: react-router v7 `<Navigate>` drops the search string intermittently under
// HashRouter (runners-nav.spec/repositories-nav.spec flake). Same hash-write fallback as
// shims/next-navigation.ts replaceWithHashFallback — write full hash atomically.
function LegacyInfraTabRedirect({ tab }: { tab: "runners" | "repositories" }) {
  const { org } = useParams<{ org: string }>();
  useEffect(() => {
    if (!org) return;
    const target = `#/${org}/infra?tab=${tab}`;
    if (window.location.hash !== target) {
      const base = window.location.href.split("#")[0];
      window.history.replaceState(null, "", base + target);
      window.dispatchEvent(new HashChangeEvent("hashchange"));
    }
  }, [org, tab]);
  return null;
}

export const router = createBrowserRouter([
  { path: "/login", element: <LoginPage /> },
  { path: "/register", element: <RegisterPage /> },
  { path: "/forgot-password", element: <ForgotPasswordPage /> },
  { path: "/reset-password", element: <ResetPasswordPage /> },
  { path: "/verify-email", element: <VerifyEmailPage /> },
  { path: "/verify-email/callback", element: <VerifyEmailCallbackPage /> },
  { path: "/auth/callback", element: <OAuthCallbackPage /> },
  { path: "/auth/sso/callback", element: <SSOCallbackPage /> },
  { path: "/invite/:token", element: <InvitePage /> },
  { path: "/runners/authorize", element: <RunnerAuthorizePage /> },
  {
    path: "/onboarding",
    children: [
      { index: true, element: <OnboardingPage /> },
      { path: "create-org", element: <CreateOrgPage /> },
      { path: "setup-runner", element: <SetupRunnerPage /> },
      { path: "setup-runner/local", element: <LocalRunnerSetupPage /> },
    ],
  },

  // Popout (no shell): window.open()-spawned renderer runs its own AppProviders.PlatformGate
  // bootstrap, so session has been rehydrated from localStorage by the time this route mounts.
  { path: "/popout/terminal/:podKey", element: <RequireAuth><PopoutTerminalPage /></RequireAuth> },

  {
    path: "/:org",
    element: <DashboardShell><Outlet /></DashboardShell>,
    errorElement: <RouteErrorBoundary />,
    children: [
      { index: true, element: <Navigate to="workspace" replace /> },
      { path: "workspace", element: <WorkspacePage /> },
      { path: "tickets", element: <TicketsPage /> },
      { path: "tickets/:slug", element: <TicketDetailPage /> },
      { path: "channels", element: <ChannelsPage /> },
      { path: "channels/:id", element: <ChannelsPage /> },
      { path: "runners", element: <LegacyInfraTabRedirect tab="runners" /> },
      { path: "runners/:id", element: <RunnerDetailPage /> },
      { path: "loops", element: <LoopsPage /> },
      { path: "loops/:slug", element: <LoopDetailPage /> },
      { path: "mesh", element: <MeshPage /> },
      { path: "blocks", element: <BlocksPage /> },
      { path: "infra", element: <InfraPage /> },
      { path: "repositories", element: <LegacyInfraTabRedirect tab="repositories" /> },
      { path: "repositories/:id", element: <RepositoryDetailPage /> },
      { path: "settings", element: <SettingsPage /> },
    ],
  },

  {
    path: "/settings",
    element: <DashboardShell><Outlet /></DashboardShell>,
    errorElement: <RouteErrorBoundary />,
    children: [
      { index: true, element: <PersonalSettingsPage /> },
      { path: "general", element: <GeneralSettingsPage /> },
      { path: "git", element: <GitSettingsPage /> },
      { path: "notifications", element: <PersonalNotificationsPage /> },
    ],
  },

  {
    path: "/support",
    element: <DashboardShell><Outlet /></DashboardShell>,
    errorElement: <RouteErrorBoundary />,
    children: [
      { index: true, element: <SupportPage /> },
      { path: ":id", element: <SupportTicketDetailPage /> },
    ],
  },

  { path: "/", element: <RootRedirect />, errorElement: <RouteErrorBoundary /> },
  { path: "*", element: <RouteErrorBoundary /> },
]);

function RootRedirect() {
  const _hasHydrated = useAuthStore((s) => s._hasHydrated);
  const isAuthenticated = useIsAuthenticated();
  const currentOrg = useCurrentOrg();
  const organizations = useAuthOrganizations();
  if (!_hasHydrated) return null;
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  const org = currentOrg ?? organizations[0];
  return org?.slug
    ? <Navigate to={`/${org.slug}/workspace`} replace />
    : <Navigate to="/onboarding" replace />;
}
