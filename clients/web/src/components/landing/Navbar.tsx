"use client";

import Link from "next/link";
import { useState, useEffect } from "react";
import { LanguageSwitcher } from "@/components/i18n";
import { LightAuthButtons as AuthButtons, Logo } from "@/components/common";
import { useTranslations } from "next-intl";

const GITHUB_URL = "https://github.com/AgentsMesh/AgentsMesh";

function GithubIcon() {
  return (
    <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24">
      <path fillRule="evenodd" clipRule="evenodd" d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" />
    </svg>
  );
}

export function Navbar() {
  const [isScrolled, setIsScrolled] = useState(false);
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const t = useTranslations();

  const navLinks = [
    { href: "#features", label: t("landing.nav.features") },
    { href: "#pricing", label: t("landing.nav.pricing") },
    { href: "/download", label: t("landing.nav.download") },
    { href: "/docs", label: t("landing.nav.docs") },
  ];

  useEffect(() => {
    const handleScroll = () => setIsScrolled(window.scrollY > 10);
    window.addEventListener("scroll", handleScroll);
    return () => window.removeEventListener("scroll", handleScroll);
  }, []);

  const collapsedRadius = isMobileMenuOpen ? "rounded-3xl" : "rounded-full";
  const containerStyle = isMobileMenuOpen
    ? "bg-[var(--azure-bg-high)]/95 backdrop-blur-xl border border-white/15 shadow-[0_24px_48px_-12px_rgba(0,0,0,0.6)]"
    : isScrolled
      ? "azure-glass border border-white/10 azure-glow-cyan-lg"
      : "bg-transparent border border-transparent";

  return (
    <nav className="fixed top-0 left-0 right-0 z-50 px-4 pt-4 sm:pt-6">
      <div
        className={`mx-auto max-w-6xl ${collapsedRadius} px-5 sm:px-7 py-3 transition-all duration-300 ${containerStyle}`}
      >
        <div className="flex items-center justify-between">
          <Link href="/" className="flex items-center gap-2.5">
            <div className="w-7 h-7 rounded-lg overflow-hidden">
              <Logo />
            </div>
            <span className="font-headline text-lg font-bold tracking-tighter text-[var(--azure-cyan)]">
              AgentsMesh
            </span>
          </Link>

          <div className="hidden md:flex items-center gap-8">
            {navLinks.map((link) => (
              <Link
                key={link.href}
                href={link.href}
                className="font-headline text-xs uppercase tracking-[0.18em] text-[var(--azure-text-muted)] hover:text-[var(--azure-cyan)] transition-colors"
              >
                {link.label}
              </Link>
            ))}
          </div>

          <div className="hidden md:flex items-center gap-4">
            <a
              href={GITHUB_URL}
              target="_blank"
              rel="noopener noreferrer"
              className="text-[var(--azure-text-muted)] hover:text-[var(--azure-cyan)] transition-colors"
              aria-label="GitHub"
            >
              <GithubIcon />
            </a>
            <LanguageSwitcher variant="icon" />
            <AuthButtons size="sm" showRegister className="flex items-center gap-3" />
          </div>

          <button
            className="md:hidden p-2 text-[var(--azure-text-muted)]"
            onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
            aria-label="Toggle menu"
          >
            <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              {isMobileMenuOpen ? (
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              ) : (
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
              )}
            </svg>
          </button>
        </div>

        {isMobileMenuOpen && (
          <div className="md:hidden mt-4 pt-4 border-t border-white/5 flex flex-col gap-4">
            {navLinks.map((link) => (
              <Link
                key={link.href}
                href={link.href}
                className="font-headline text-xs uppercase tracking-[0.18em] text-[var(--azure-text-muted)] hover:text-[var(--azure-cyan)] transition-colors"
                onClick={() => setIsMobileMenuOpen(false)}
              >
                {link.label}
              </Link>
            ))}
            <div className="flex flex-col gap-3 pt-3 border-t border-white/5">
              <a
                href={GITHUB_URL}
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center gap-2 text-sm text-[var(--azure-text-muted)] hover:text-[var(--azure-cyan)] transition-colors"
              >
                <GithubIcon />
                GitHub
              </a>
              <div className="flex items-center justify-between py-1">
                <span className="text-sm text-[var(--azure-text-muted)]">{t("landing.nav.language")}</span>
                <LanguageSwitcher variant="full" />
              </div>
              <AuthButtons
                size="sm"
                onClick={() => setIsMobileMenuOpen(false)}
                className="flex flex-col gap-2 [&_button]:w-full"
              />
            </div>
          </div>
        )}
      </div>
    </nav>
  );
}
