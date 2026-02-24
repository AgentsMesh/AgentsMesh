"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Users,
  Building2,
  Server,
  ScrollText,
  LogOut,
  Tag,
  Radio,
} from "lucide-react";
import { cn } from "@/lib/utils";
import { useAuthStore } from "@/stores/auth";
import { Button } from "@/components/ui/button";

const navItems = [
  {
    title: "Dashboard",
    href: "/",
    icon: LayoutDashboard,
  },
  {
    title: "Users",
    href: "/users",
    icon: Users,
  },
  {
    title: "Organizations",
    href: "/organizations",
    icon: Building2,
  },
  {
    title: "Runners",
    href: "/runners",
    icon: Server,
  },
  {
    title: "Relays",
    href: "/relays",
    icon: Radio,
  },
  {
    title: "Promo Codes",
    href: "/promo-codes",
    icon: Tag,
  },
  {
    title: "Audit Logs",
    href: "/audit-logs",
    icon: ScrollText,
  },
];

export function Sidebar() {
  const pathname = usePathname();
  const { user, logout } = useAuthStore();

  return (
    <aside className="flex h-screen w-64 flex-col border-r border-border bg-card">
      {/* Logo */}
      <div className="flex h-16 items-center gap-2 border-b border-border px-6">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 400 400"
          className="h-7 w-7 rounded-md"
        >
          <rect x="0" y="0" width="400" height="400" rx="32" ry="32" fill="#3E7DC7" />
          <rect x="110" y="60" width="180" height="80" rx="14" ry="14" fill="#FFFFFF" />
          <rect x="120" y="90" width="160" height="10" rx="2" ry="2" fill="#3E7DC7" />
          <rect x="140" y="126" width="120" height="34" fill="#FFFFFF" />
          <rect x="65" y="142" width="270" height="20" rx="10" ry="10" fill="#FFFFFF" />
          <polygon points="153,156 247,156 262,308 138,308" fill="#FFFFFF" />
          <rect x="116" y="302" width="168" height="36" rx="8" ry="8" fill="#FFFFFF" />
        </svg>
        <span className="text-lg font-semibold">Admin Console</span>
      </div>

      {/* Navigation */}
      <nav className="flex-1 space-y-1 p-4">
        {navItems.map((item) => {
          const isActive = pathname === item.href ||
            (item.href !== "/" && pathname.startsWith(item.href));
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                isActive
                  ? "bg-primary/10 text-primary"
                  : "text-muted-foreground hover:bg-accent hover:text-foreground"
              )}
            >
              <item.icon className="h-5 w-5" />
              {item.title}
            </Link>
          );
        })}
      </nav>

      {/* User Info & Logout */}
      <div className="border-t border-border p-4">
        {user && (
          <div className="mb-3 flex items-center gap-3">
            {user.avatar_url ? (
              <img
                src={user.avatar_url}
                alt={user.username}
                className="h-8 w-8 rounded-full"
              />
            ) : (
              <div className="flex h-8 w-8 items-center justify-center rounded-full bg-primary/20 text-sm font-medium text-primary">
                {user.username[0].toUpperCase()}
              </div>
            )}
            <div className="flex-1 truncate">
              <p className="text-sm font-medium truncate">{user.name || user.username}</p>
              <p className="text-xs text-muted-foreground truncate">{user.email}</p>
            </div>
          </div>
        )}
        <Button
          variant="ghost"
          className="w-full justify-start text-muted-foreground"
          onClick={logout}
        >
          <LogOut className="mr-2 h-4 w-4" />
          Sign Out
        </Button>
      </div>
    </aside>
  );
}
