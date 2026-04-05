"use client";

import { useTranslations } from "next-intl";

export function BuiltinVarsEscaping() {
  const t = useTranslations();

  return (
    <>
      {/* Built-in Variables */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.concepts.agentfile.builtinVars.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-6">
          {t("docs.concepts.agentfile.builtinVars.description")}
        </p>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">
                  {t("docs.concepts.agentfile.builtinVars.variableHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.concepts.agentfile.builtinVars.descHeader")}
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border font-mono text-xs">
                  config.*
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.concepts.agentfile.builtinVars.configDesc")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-mono text-xs">
                  sandbox.root
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.concepts.agentfile.builtinVars.sandboxRoot")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-mono text-xs">
                  sandbox.work_dir
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.concepts.agentfile.builtinVars.sandboxWorkDir")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-mono text-xs">
                  mcp.enabled
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.concepts.agentfile.builtinVars.mcpEnabled")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-mono text-xs">
                  mcp.servers
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.concepts.agentfile.builtinVars.mcpServers")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-mono text-xs">
                  mcp.format
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.concepts.agentfile.builtinVars.mcpFormat")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-mono text-xs">
                  mode
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.concepts.agentfile.builtinVars.modeDesc")}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* String Escaping */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.concepts.agentfile.escaping.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.concepts.agentfile.escaping.description")}
        </p>
        <pre className="bg-muted rounded-lg p-3 text-sm overflow-x-auto">
          <code>{`\\\\  →  backslash
\\"  →  double quote
\\n  →  newline
\\t  →  tab`}</code>
        </pre>
      </section>
    </>
  );
}
