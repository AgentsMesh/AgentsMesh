"use client";

import Link from "next/link";
import { Logo } from "@/components/common";

interface VerifyStateScreenProps {
  t: (key: string) => string;
}

export function VerifyingScreen({ t }: VerifyStateScreenProps) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm space-y-6 text-center">
        <LogoHeader />
        <div className="flex justify-center">
          <div className="w-16 h-16 rounded-full bg-primary/10 flex items-center justify-center">
            <svg className="w-8 h-8 text-primary animate-spin" fill="none" viewBox="0 0 24 24">
              <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
              <path className="opacity-75" fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
            </svg>
          </div>
        </div>
        <div className="space-y-2">
          <h1 className="text-2xl font-semibold text-foreground">{t("auth.verifyEmailPage.verifying")}</h1>
          <p className="text-sm text-muted-foreground">{t("auth.verifyEmailPage.pleaseWait")}</p>
        </div>
      </div>
    </div>
  );
}

export function SuccessScreen({ t }: VerifyStateScreenProps) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-sm space-y-6 text-center">
        <LogoHeader />
        <div className="flex justify-center">
          <div className="w-16 h-16 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center">
            <svg className="w-8 h-8 text-green-600 dark:text-green-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
          </div>
        </div>
        <div className="space-y-2">
          <h1 className="text-2xl font-semibold text-foreground">{t("auth.verifyEmailPage.verificationSuccessTitle")}</h1>
          <p className="text-sm text-muted-foreground">{t("auth.verifyEmailPage.redirecting")}</p>
        </div>
      </div>
    </div>
  );
}

export function LogoHeader() {
  return (
    <div>
      <Link href="/" className="inline-flex items-center gap-2">
        <div className="w-10 h-10 rounded-lg overflow-hidden"><Logo /></div>
        <span className="text-2xl font-bold text-foreground">AgentsMesh</span>
      </Link>
    </div>
  );
}
