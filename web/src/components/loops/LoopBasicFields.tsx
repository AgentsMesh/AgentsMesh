"use client";

import React from "react";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { AgentSelect } from "@/components/pod/CreatePodForm/AgentSelect";
import { PromptInput } from "@/components/pod/CreatePodForm/PromptInput";
import type { AgentData } from "@/lib/api";

interface LoopBasicFieldsProps {
  name: string;
  setName: (v: string) => void;
  description: string;
  setDescription: (v: string) => void;
  promptTemplate: string;
  setPromptTemplate: (v: string) => void;
  selectedAgentSlug: string | null;
  setSelectedAgentSlug: (slug: string | null) => void;
  availableAgents: AgentData[];
  t: (key: string) => string;
}

export function LoopBasicFields({
  name,
  setName,
  description,
  setDescription,
  promptTemplate,
  setPromptTemplate,
  selectedAgentSlug,
  setSelectedAgentSlug,
  availableAgents,
  t,
}: LoopBasicFieldsProps) {
  return (
    <>
      {/* Name */}
      <div className="space-y-1.5">
        <Label>{t("loops.name")}</Label>
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="daily-code-review"
        />
      </div>

      {/* Description */}
      <div className="space-y-1.5">
        <Label>{t("loops.description")}</Label>
        <Input
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder={t("loops.descriptionPlaceholder")}
        />
      </div>

      {/* Agent Select */}
      <AgentSelect
        agents={availableAgents}
        selectedAgentSlug={selectedAgentSlug}
        onSelect={setSelectedAgentSlug}
        t={t}
      />

      {/* Prompt Template (shown when agent selected) */}
      {selectedAgentSlug && (
        <PromptInput
          value={promptTemplate}
          onChange={setPromptTemplate}
          placeholder={t("loops.promptPlaceholder")}
          t={t}
        />
      )}
    </>
  );
}
