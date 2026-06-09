"use client";

import { Button } from "@/components/ui/button";
import {
  Bot,
  Plus,
  Check,
  Server,
  ChevronDown,
  ChevronRight,
  Star,
} from "lucide-react";
import type { AgentItemProps } from "./types";
import { CredentialProfileRow } from "../CredentialProfileRow";

/**
 * AgentIcon - Returns an icon based on agent slug
 */
function AgentIcon({ slug: _slug }: { slug: string }) {
  void _slug; // Reserved for future per-agent icons
  return <Bot className="w-5 h-5" />;
}

/**
 * AgentItem - Expandable panel for a single agent's credentials
 *
 * Shows the agent header with expand/collapse toggle, the "no bundle"
 * (Runner-native env) row as first option, and custom credential bundles
 * below.
 */
export function AgentItem({
  agent,
  profiles,
  isExpanded,
  noPrimaryBundle,
  onToggle,
  onClearPrimary,
  onSetDefault,
  onEdit,
  onDelete,
  onAdd,
  t,
}: AgentItemProps) {
  return (
    <div className="border border-border rounded-lg overflow-hidden">
      {/* Agent Header */}
      <button
        className="w-full flex items-center justify-between p-4 hover:bg-muted/50 transition-colors"
        onClick={onToggle}
      >
        <div className="flex items-center gap-3">
          {isExpanded ? (
            <ChevronDown className="w-4 h-4 text-muted-foreground" />
          ) : (
            <ChevronRight className="w-4 h-4 text-muted-foreground" />
          )}
          <AgentIcon slug={agent.slug} />
          <div className="text-left">
            <div className="font-medium">{agent.name}</div>
            {agent.description && (
              <div className="text-xs text-muted-foreground">
                {agent.description}
              </div>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground">
            {profiles.length} {t("settings.agentCredentials.profiles")}
          </span>
        </div>
      </button>

      {/* Profiles List */}
      {isExpanded && (
        <div className="border-t border-border bg-muted/20">
          {/* "No bundle" — always shown as first option, not deletable.
              Represents "use the Runner's native env"; selecting this clears
              any primary bundle in this (user, agent, kind=credential) group. */}
          <div className="px-4 py-3 flex items-center justify-between hover:bg-muted/50 border-b border-border">
            <div className="flex items-center gap-3">
              <Server className="w-4 h-4 text-muted-foreground" />
              <div>
                <div className="flex items-center gap-2">
                  <span className="font-medium">{t("settings.agentCredentials.noBundleLabel")}</span>
                  {noPrimaryBundle && (
                    <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs bg-primary/10 text-primary">
                      <Star className="w-3 h-3 mr-0.5" />
                      {t("settings.agentCredentials.default")}
                    </span>
                  )}
                </div>
                <div className="text-xs text-muted-foreground">
                  {t("settings.agentCredentials.noBundleHint")}
                </div>
              </div>
            </div>
            <div className="flex items-center gap-1">
              {!noPrimaryBundle && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={onClearPrimary}
                  title={t("settings.agentCredentials.setAsDefault")}
                >
                  <Check className="w-4 h-4" />
                </Button>
              )}
            </div>
          </div>

          {/* Custom credential profiles */}
          {profiles.length > 0 && (
            <div className="divide-y divide-border">
              {profiles.map((profile) => (
                <div
                  key={profile.id}
                  className="px-4 py-3 flex items-center justify-between hover:bg-muted/50"
                >
                  <CredentialProfileRow
                    profile={profile}
                    agentSlug={agent.slug}
                    onSetDefault={onSetDefault}
                    onEdit={onEdit}
                    onDelete={onDelete}
                    t={t}
                  />
                </div>
              ))}
            </div>
          )}

          {/* Add button */}
          <div className="px-4 py-3 border-t border-border">
            <Button
              variant="outline"
              size="sm"
              onClick={onAdd}
            >
              <Plus className="w-4 h-4 mr-1" />
              {t("settings.agentCredentials.addProfile")}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

export default AgentItem;
