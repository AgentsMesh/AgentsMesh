"use client";

import { useEffect, useState, use } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { useAuthStore } from "@/stores/auth";
import { invitationApi, InvitationInfo } from "@/lib/api/invitation";
import { organizationApi } from "@/lib/api/organization";
import { Logo } from "@/components/common";

interface PageParams {
  token: string;
}

export default function InvitePage({ params }: { params: Promise<PageParams> }) {
  const resolvedParams = use(params);
  const router = useRouter();
  const { token: authToken, user, setOrganizations, setCurrentOrg } = useAuthStore();
  const [invitation, setInvitation] = useState<InvitationInfo | null>(null);
  const [loading, setLoading] = useState(true);
  const [accepting, setAccepting] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    const fetchInvitation = async () => {
      try {
        const { invitation: inv } = await invitationApi.getByToken(resolvedParams.token);
        setInvitation(inv);
      } catch {
        setError("This invitation is invalid or has expired.");
      } finally {
        setLoading(false);
      }
    };

    fetchInvitation();
  }, [resolvedParams.token]);

  const handleAccept = async () => {
    if (!invitation) return;

    setAccepting(true);
    setError("");

    try {
      const { organization } = await invitationApi.accept(resolvedParams.token);

      // Refresh organizations list
      const { organizations } = await organizationApi.list();
      setOrganizations(organizations);

      // Set the new org as current
      const newOrg = organizations.find((o) => o.id === organization.id);
      if (newOrg) {
        setCurrentOrg(newOrg);
      }

      // Redirect to the organization workspace
      router.push(`/${organization.slug}/workspace`);
    } catch (err: unknown) {
      if (err && typeof err === "object" && "data" in err) {
        const apiErr = err as { data?: { error?: string } };
        setError(apiErr.data?.error || "Failed to accept invitation. Please try again.");
      } else {
        setError("Failed to accept invitation. Please try again.");
      }
      setAccepting(false);
    }
  };

  // Loading state
  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background px-4">
        <div className="w-full max-w-md space-y-6 text-center">
          <div className="flex justify-center">
            <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
          </div>
          <p className="text-sm text-muted-foreground">Loading invitation...</p>
        </div>
      </div>
    );
  }

  // Error state (invalid/expired)
  if (error && !invitation) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background px-4">
        <div className="w-full max-w-md space-y-6 text-center">
          {/* Logo */}
          <div>
            <Link href="/" className="inline-flex items-center gap-2">
              <div className="w-10 h-10 rounded-lg overflow-hidden">
                <Logo />
              </div>
              <span className="text-2xl font-bold text-foreground">AgentsMesh</span>
            </Link>
          </div>

          {/* Error Icon */}
          <div className="flex justify-center">
            <div className="w-16 h-16 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center">
              <svg
                className="w-8 h-8 text-red-600 dark:text-red-400"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </div>
          </div>

          {/* Content */}
          <div className="space-y-2">
            <h1 className="text-2xl font-semibold text-foreground">
              Invalid Invitation
            </h1>
            <p className="text-sm text-muted-foreground">{error}</p>
          </div>

          <Link href="/login">
            <Button className="w-full">Go to Sign In</Button>
          </Link>
        </div>
      </div>
    );
  }

  // Expired invitation
  if (invitation?.is_expired) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background px-4">
        <div className="w-full max-w-md space-y-6 text-center">
          {/* Logo */}
          <div>
            <Link href="/" className="inline-flex items-center gap-2">
              <div className="w-10 h-10 rounded-lg overflow-hidden">
                <Logo />
              </div>
              <span className="text-2xl font-bold text-foreground">AgentsMesh</span>
            </Link>
          </div>

          {/* Warning Icon */}
          <div className="flex justify-center">
            <div className="w-16 h-16 rounded-full bg-amber-100 dark:bg-amber-900/30 flex items-center justify-center">
              <svg
                className="w-8 h-8 text-amber-600 dark:text-amber-400"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
            </div>
          </div>

          {/* Content */}
          <div className="space-y-2">
            <h1 className="text-2xl font-semibold text-foreground">
              Invitation Expired
            </h1>
            <p className="text-sm text-muted-foreground">
              This invitation to join <strong>{invitation.organization_name}</strong> has expired.
              Please ask the organization admin to send a new invitation.
            </p>
          </div>

          <Link href="/login">
            <Button className="w-full">Go to Sign In</Button>
          </Link>
        </div>
      </div>
    );
  }

  // Valid invitation - show accept UI
  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-md space-y-6">
        {/* Logo */}
        <div className="text-center">
          <Link href="/" className="inline-flex items-center gap-2">
            <div className="w-10 h-10 rounded-lg overflow-hidden">
              <Logo />
            </div>
            <span className="text-2xl font-bold text-foreground">AgentsMesh</span>
          </Link>
        </div>

        {/* Invitation Card */}
        <div className="p-6 border border-border rounded-lg space-y-4">
          {/* Icon */}
          <div className="flex justify-center">
            <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
              <svg
                className="w-8 h-8 text-primary"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z"
                />
              </svg>
            </div>
          </div>

          {/* Content */}
          <div className="text-center space-y-2">
            <h1 className="text-xl font-semibold text-foreground">
              You&apos;re invited to join
            </h1>
            <p className="text-2xl font-bold text-primary">
              {invitation?.organization_name}
            </p>
            <p className="text-sm text-muted-foreground">
              {invitation?.inviter_name} has invited you to join as{" "}
              <span className="font-medium capitalize">{invitation?.role}</span>
            </p>
          </div>

          {/* Error message */}
          {error && (
            <div className="p-3 text-sm text-destructive bg-destructive/10 rounded-md">
              {error}
            </div>
          )}

          {/* Actions */}
          {authToken && user ? (
            <div className="space-y-3">
              <p className="text-sm text-center text-muted-foreground">
                Signed in as <strong>{user.email}</strong>
              </p>
              <Button className="w-full" onClick={handleAccept} disabled={accepting}>
                {accepting ? "Joining..." : "Accept Invitation"}
              </Button>
            </div>
          ) : (
            <div className="space-y-3">
              <p className="text-sm text-center text-muted-foreground">
                Sign in to accept this invitation
              </p>
              <Link href={`/login?redirect=/invite/${resolvedParams.token}`}>
                <Button className="w-full">Sign In to Accept</Button>
              </Link>
              <p className="text-sm text-center text-muted-foreground">
                Don&apos;t have an account?{" "}
                <Link
                  href={`/register?redirect=/invite/${resolvedParams.token}`}
                  className="text-primary hover:underline"
                >
                  Sign up
                </Link>
              </p>
            </div>
          )}
        </div>

        {/* Expiration notice */}
        {invitation && (
          <p className="text-center text-xs text-muted-foreground">
            This invitation expires on{" "}
            {new Date(invitation.expires_at).toLocaleDateString()}
          </p>
        )}
      </div>
    </div>
  );
}
