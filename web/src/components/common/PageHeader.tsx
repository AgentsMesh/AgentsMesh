"use client";

import Link from "next/link";
import { AuthButtons } from "./AuthButtons";
import { Logo } from "./Logo";

export function PageHeader() {
  return (
    <header className="border-b border-border">
      <div className="container mx-auto px-4 py-4 flex items-center justify-between">
        <Link href="/" className="flex items-center gap-2">
          <div className="w-8 h-8 rounded-lg overflow-hidden">
            <Logo />
          </div>
          <span className="text-xl font-bold">AgentsMesh</span>
        </Link>
        <AuthButtons consoleVariant="primary" />
      </div>
    </header>
  );
}
