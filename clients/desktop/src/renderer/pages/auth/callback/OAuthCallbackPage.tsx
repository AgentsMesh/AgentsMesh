import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { useAuthStore } from "@/stores/auth";
import { userApi, organizationApi } from "@/lib/api";
import { Logo } from "@/components/common";

function OAuthCallbackContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const token = searchParams.get("token");
  const refreshToken = searchParams.get("refresh_token");
  const error = searchParams.get("error");
  const { setAuth, setOrganizations } = useAuthStore();

  const [status, setStatus] = useState<"loading" | "success" | "error">("loading");
  const [errorMessage, setErrorMessage] = useState("");

  useEffect(() => {
    const handleCallback = async () => {
      if (error) {
        setStatus("error");
        setErrorMessage(error === "access_denied"
          ? "You cancelled the authorization request."
          : `Authentication failed: ${error}`);
        return;
      }

      if (!token) {
        setStatus("error");
        setErrorMessage("Authentication token is missing.");
        return;
      }

      try {
        // MUST await: setAuth fans out authApplySession IPC to main-process Rust AuthManager
        // (ApiClient's SSOT). v0.31.x onboarding-loop bug: synchronous path left token in
        // renderer-only state, next userGetMe IPC saw no token → 401. Placeholder is
        // overwritten by getMe() below; catch MUST logout() to wipe if anything throws.
        await setAuth(token, { id: 0, email: "", username: "" }, refreshToken || undefined);

        const userResponse = await userApi.getMe();
        const user = userResponse.user;
        await setAuth(token, user, refreshToken || undefined);

        const orgsResponse = await organizationApi.list();
        if (orgsResponse.organizations && orgsResponse.organizations.length > 0) {
          await setOrganizations(orgsResponse.organizations);
          setStatus("success");
          setTimeout(() => {
            router.push(`/${orgsResponse.organizations[0].slug}/workspace`);
          }, 1500);
        } else {
          setStatus("success");
          setTimeout(() => {
            router.push("/onboarding");
          }, 1500);
        }
      } catch (err: unknown) {
        await useAuthStore.getState().logout();
        setStatus("error");
        if (err instanceof Error) {
          setErrorMessage(err.message);
        } else {
          setErrorMessage("Failed to complete authentication.");
        }
      }
    };

    handleCallback();
  }, [token, refreshToken, error, setAuth, setOrganizations, router]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm space-y-6 text-center">
        <div>
          <Link href="/" className="inline-flex items-center gap-2">
            <div className="w-10 h-10 rounded-lg overflow-hidden">
              <Logo />
            </div>
            <span className="text-2xl font-bold text-foreground">AgentsMesh</span>
          </Link>
        </div>

        {status === "loading" && (
          <>
            <div className="flex justify-center">
              <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center animate-pulse">
                <svg
                  className="w-8 h-8 text-primary animate-spin"
                  fill="none"
                  viewBox="0 0 24 24"
                >
                  <circle
                    className="opacity-25"
                    cx="12"
                    cy="12"
                    r="10"
                    stroke="currentColor"
                    strokeWidth="4"
                  />
                  <path
                    className="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                  />
                </svg>
              </div>
            </div>
            <div className="space-y-2">
              <h1 className="text-2xl font-semibold text-foreground">
                Completing sign in...
              </h1>
              <p className="text-sm text-muted-foreground">
                Please wait while we set up your account.
              </p>
            </div>
          </>
        )}

        {status === "success" && (
          <>
            <div className="flex justify-center">
              <div className="w-16 h-16 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center">
                <svg
                  className="w-8 h-8 text-green-600 dark:text-green-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M5 13l4 4L19 7"
                  />
                </svg>
              </div>
            </div>
            <div className="space-y-2">
              <h1 className="text-2xl font-semibold text-foreground">
                Welcome!
              </h1>
              <p className="text-sm text-muted-foreground">
                You have signed in successfully.
                <br />
                Redirecting...
              </p>
            </div>
          </>
        )}

        {status === "error" && (
          <>
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
            <div className="space-y-2">
              <h1 className="text-2xl font-semibold text-foreground">
                Sign in failed
              </h1>
              <p className="text-sm text-muted-foreground">{errorMessage}</p>
            </div>
            <div className="space-y-3">
              <Link href="/login">
                <Button className="w-full">Try again</Button>
              </Link>
              <Link href="/register">
                <Button variant="outline" className="w-full">Create account</Button>
              </Link>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

export function OAuthCallbackPage() {
  return (
    <Suspense fallback={
      <div className="flex min-h-screen items-center justify-center bg-background px-4">
        <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center animate-pulse">
          <svg
            className="w-8 h-8 text-primary animate-spin"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle
              className="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="4"
            />
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            />
          </svg>
        </div>
      </div>
    }>
      <OAuthCallbackContent />
    </Suspense>
  );
}
