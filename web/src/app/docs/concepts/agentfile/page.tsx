"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";
import { DeclarationKeywords } from "./_sections/DeclarationKeywords";
import { BuildLogicExpressions } from "./_sections/BuildLogicExpressions";
import { BuiltinVarsEscaping } from "./_sections/BuiltinVarsEscaping";
import { FullExample } from "./_sections/FullExample";

export default function AgentfilePage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.concepts.agentfile.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.concepts.agentfile.description")}
      </p>

      {/* What is AgentFile */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.concepts.agentfile.whatIs.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed">
          {t("docs.concepts.agentfile.whatIs.description")}
        </p>
      </section>

      {/* Structure */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.concepts.agentfile.structure.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.concepts.agentfile.structure.description")}
        </p>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.concepts.agentfile.structure.declarationsTitle")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.concepts.agentfile.structure.declarationsDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.concepts.agentfile.structure.buildLogicTitle")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.concepts.agentfile.structure.buildLogicDesc")}
            </p>
          </div>
        </div>
      </section>

      {/* Sections split for 200-line limit */}
      <DeclarationKeywords />
      <BuildLogicExpressions />
      <BuiltinVarsEscaping />
      <FullExample />

      <DocNavigation />
    </div>
  );
}
