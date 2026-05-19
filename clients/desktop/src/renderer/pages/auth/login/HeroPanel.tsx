import { Logo } from "@/components/common";
import { useTranslations } from "next-intl";

export function HeroPanel() {
  const t = useTranslations();
  return (
    <div className="relative flex h-full w-full flex-col items-center justify-center overflow-hidden">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0"
        style={{
          background:
            "radial-gradient(circle at 30% 35%, rgba(62, 125, 199, 0.18), transparent 55%), " +
            "radial-gradient(circle at 70% 65%, rgba(62, 125, 199, 0.12), transparent 60%)",
          animation: "auth-hero-pulse 8s ease-in-out infinite",
        }}
      />
      <div
        className="relative flex flex-col items-center gap-8"
        style={{ animation: "auth-hero-float 6s ease-in-out infinite" }}
      >
        <div className="h-32 w-32 overflow-hidden rounded-3xl shadow-xl">
          <Logo />
        </div>
        <div className="text-center">
          <h2 className="text-3xl font-semibold tracking-tight text-foreground">
            {t("auth.loginPage.tagline")}
          </h2>
          <p className="mt-3 text-sm text-muted-foreground">
            {t("auth.loginPage.taglineSub")}
          </p>
        </div>
      </div>
      <style>{`
        @keyframes auth-hero-float {
          0%, 100% { transform: translateY(0); }
          50% { transform: translateY(-8px); }
        }
        @keyframes auth-hero-pulse {
          0%, 100% { opacity: 0.85; }
          50% { opacity: 1; }
        }
      `}</style>
    </div>
  );
}
