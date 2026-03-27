"use client";

import { Button } from "@/components/ui/button";
import {
  Bot,
  Plus,
  Check,
  Trash2,
  Edit2,
  Server,
  Key,
  ChevronDown,
  ChevronRight,
  Star,
} from "lucide-react";
import type { AgentItemProps } from "./types";

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
 * Shows the agent header with expand/collapse toggle,
 * RunnerHost as first option, and custom credential profiles below.
 */
export function AgentItem({
  agent,
  profiles,
  isExpanded,
  isRunnerHostDefault,
  onToggle,
  onSetRunnerHostDefault,
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
          {/* RunnerHost - always shown as first option, not deletable */}
          <div className="px-4 py-3 flex items-center justify-between hover:bg-muted/50 border-b border-border">
            <div className="flex items-center gap-3">
              <Server className="w-4 h-4 text-muted-foreground" />
              <div>
                <div className="flex items-center gap-2">
                  <span className="font-medium">RunnerHost</span>
                  {isRunnerHostDefault && (
                    <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs bg-primary/10 text-primary">
                      <Star className="w-3 h-3 mr-0.5" />
                      {t("settings.agentCredentials.default")}
                    </span>
                  )}
                </div>
                <div className="text-xs text-muted-foreground">
                  {t("settings.agentCredentials.runnerHostHint")}
                </div>
              </div>
            </div>
            <div className="flex items-center gap-1">
              {!isRunnerHostDefault && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={onSetRunnerHostDefault}
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
                  <div className="flex items-center gap-3">
                    <Key className="w-4 h-4 text-muted-foreground" />
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="font-medium">{profile.name}</span>
                        {profile.is_default && (
                          <span className="inline-flex items-center px-1.5 py-0.5 rounded text-xs bg-primary/10 text-primary">
                            <Star className="w-3 h-3 mr-0.5" />
                            {t("settings.agentCredentials.default")}
                          </span>
                        )}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {profile.configured_fields?.length
                          ? `${t("settings.agentCredentials.configured")}: ${profile.configured_fields.join(", ")}`
                          : t("settings.agentCredentials.notConfigured")}
                      </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-1">
                    {!profile.is_default && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => onSetDefault(profile.id)}
                        title={t("settings.agentCredentials.setAsDefault")}
                      >
                        <Check className="w-4 h-4" />
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => onEdit(profile)}
                    >
                      <Edit2 className="w-4 h-4" />
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => onDelete(profile.id)}
                      className="text-destructive hover:text-destructive"
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
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
