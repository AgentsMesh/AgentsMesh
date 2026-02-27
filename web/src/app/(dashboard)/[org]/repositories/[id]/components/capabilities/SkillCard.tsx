"use client";

import { useTranslations } from "next-intl";
import { Badge } from "@/components/ui/badge";
import { Switch } from "@/components/ui/switch";
import { Button } from "@/components/ui/button";
import { ExternalLink, Trash2 } from "lucide-react";
import type { InstalledSkill } from "@/lib/api";

interface SkillCardProps {
  skill: InstalledSkill;
  canManage: boolean;
  onToggle: (skill: InstalledSkill) => void;
  onDelete: (skill: InstalledSkill) => void;
}

export function SkillCard({ skill, canManage, onToggle, onDelete }: SkillCardProps) {
  const t = useTranslations();

  const sourceLabel = {
    market: t("extensions.sourceMarket"),
    github: t("extensions.sourceGithub"),
    upload: t("extensions.sourceUpload"),
  }[skill.install_source] || skill.install_source;

  // Determine the best source URL for linking
  const sourceUrl = skill.source_url || skill.market_item?.registry?.repository_url || null;

  return (
    <div className="border border-border rounded-lg p-4 flex items-center justify-between gap-4">
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-1">
          <span className="font-medium truncate">{skill.slug}</span>
          <Badge variant="secondary" className="text-xs shrink-0">
            {sourceLabel}
          </Badge>
          {skill.pinned_version != null && (
            <Badge variant="outline" className="text-xs shrink-0">
              v{skill.pinned_version}
            </Badge>
          )}
          {sourceUrl && (
            <a
              href={sourceUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="text-muted-foreground hover:text-foreground shrink-0"
              title={t("extensions.viewSource")}
              onClick={(e) => e.stopPropagation()}
            >
              <ExternalLink className="w-3.5 h-3.5" />
            </a>
          )}
        </div>
        {skill.source_url && (
          <a
            href={skill.source_url}
            target="_blank"
            rel="noopener noreferrer"
            className="text-xs text-muted-foreground hover:text-foreground hover:underline truncate block"
            onClick={(e) => e.stopPropagation()}
          >
            {skill.source_url}
          </a>
        )}
      </div>

      {canManage && (
        <div className="flex items-center gap-3 shrink-0">
          <Switch
            checked={skill.is_enabled}
            onCheckedChange={() => onToggle(skill)}
            aria-label={t("extensions.toggleEnabled")}
          />
          <Button
            variant="ghost"
            size="sm"
            onClick={() => onDelete(skill)}
            className="text-destructive hover:text-destructive"
          >
            <Trash2 className="w-4 h-4" />
          </Button>
        </div>
      )}
    </div>
  );
}
