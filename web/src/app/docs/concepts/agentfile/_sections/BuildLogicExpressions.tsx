"use client";

import { useTranslations } from "next-intl";

export function BuildLogicExpressions() {
  const t = useTranslations();

  return (
    <>
      {/* Build Logic */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.concepts.agentfile.buildLogic.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-6">
          {t("docs.concepts.agentfile.buildLogic.description")}
        </p>
        <pre className="bg-muted rounded-lg p-4 text-sm overflow-x-auto mb-4">
          <code>{`# Variable assignment
model_flag = "--model " + config.model

# arg — append CLI argument
arg model_flag

# file — write a file
file ".env" "NODE_ENV=production"

# mkdir — create a directory
mkdir sandbox.work_dir + "/output"

# if / else
if config.verbose {
  arg "--verbose"
} else {
  arg "--quiet"
}

# for / in
for server in mcp.servers {
  arg "--mcp-server " + server
}

# when — shorthand conditional
when config.verbose arg "--verbose"`}</code>
        </pre>
      </section>

      {/* Expressions */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.concepts.agentfile.expressions.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-6">
          {t("docs.concepts.agentfile.expressions.description")}
        </p>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">
                  {t("docs.concepts.agentfile.expressions.typeHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.concepts.agentfile.expressions.exampleHeader")}
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border">Strings</td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  {`"hello" + " world"`}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">Numbers</td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  42, 3.14
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">Booleans</td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  true, false
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">Dot access</td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  config.model, sandbox.root
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">Operators</td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  +, ==, !=, and, or, not
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border">Functions</td>
                <td className="p-3 border-b border-border font-mono text-xs">
                  json(...), str_replace(...), env(...)
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </>
  );
}
