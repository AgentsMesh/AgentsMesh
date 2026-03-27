import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Logo as LogoIcon } from "@/components/common";

export function BrandLogo() {
  return (
    <Link href="/" className="inline-flex items-center gap-2">
      <div className="w-10 h-10 rounded-lg overflow-hidden">
        <LogoIcon />
      </div>
      <span className="text-2xl font-bold text-foreground">AgentsMesh</span>
    </Link>
  );
}

function StatusIcon({ bg, color, children }: {
  bg: string; color: string; children: React.ReactNode;
}) {
  return (
    <div className="flex justify-center">
      <div className={`w-16 h-16 rounded-full ${bg} flex items-center justify-center`}>
        <svg className={`w-8 h-8 ${color}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
          {children}
        </svg>
      </div>
    </div>
  );
}

export function LoadingScreen({ message }: { message: string }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-md space-y-6 text-center">
        <div className="flex justify-center">
          <div className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
        </div>
        <p className="text-sm text-muted-foreground">{message}</p>
      </div>
    </div>
  );
}

export function ErrorScreen({ error, loginLabel }: { error: string; loginLabel: string }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-md space-y-6 text-center">
        <BrandLogo />
        <StatusIcon bg="bg-red-100 dark:bg-red-900/30" color="text-red-600 dark:text-red-400">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
        </StatusIcon>
        <div className="space-y-2">
          <p className="text-sm text-muted-foreground">{error}</p>
        </div>
        <Link href="/login"><Button className="w-full">{loginLabel}</Button></Link>
      </div>
    </div>
  );
}

export function ExpiredScreen({ title, description, hint }: {
  title: string; description: string; hint: string;
}) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-md space-y-6 text-center">
        <BrandLogo />
        <StatusIcon bg="bg-amber-100 dark:bg-amber-900/30" color="text-amber-600 dark:text-amber-400">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2}
            d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </StatusIcon>
        <div className="space-y-2">
          <h1 className="text-2xl font-semibold text-foreground">{title}</h1>
          <p className="text-sm text-muted-foreground">{description}</p>
        </div>
        <p className="text-sm text-muted-foreground">{hint}</p>
      </div>
    </div>
  );
}

export function SuccessScreen({ title, description, hint }: {
  title: string; description: string; hint: string;
}) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-background px-4">
      <div className="w-full max-w-md space-y-6 text-center">
        <BrandLogo />
        <StatusIcon bg="bg-green-100 dark:bg-green-900/30" color="text-green-600 dark:text-green-400">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
        </StatusIcon>
        <div className="space-y-2">
          <h1 className="text-2xl font-semibold text-foreground">{title}</h1>
          <p className="text-sm text-muted-foreground">{description}</p>
        </div>
        <p className="text-sm text-muted-foreground">{hint}</p>
      </div>
    </div>
  );
}
