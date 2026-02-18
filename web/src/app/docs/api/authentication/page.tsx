"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function ApiAuthenticationPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.api.authentication.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.api.authentication.description")}
      </p>

      {/* Authentication Methods */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.api.authentication.methods.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.api.authentication.methods.description")}
        </p>
        <div className="space-y-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.api.authentication.methods.headerMethod")}
            </h3>
            <p className="text-sm text-muted-foreground mb-3">
              {t("docs.api.authentication.methods.headerMethodDesc")}
            </p>
            <div className="bg-muted rounded-lg p-4 font-mono text-sm">
              <pre className="text-green-500 dark:text-green-400">{`curl -H "X-API-Key: amk_your_api_key_here" \\
  https://your-domain.com/api/v1/ext/orgs/my-org/pods`}</pre>
            </div>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.api.authentication.methods.bearerMethod")}
            </h3>
            <p className="text-sm text-muted-foreground mb-3">
              {t("docs.api.authentication.methods.bearerMethodDesc")}
            </p>
            <div className="bg-muted rounded-lg p-4 font-mono text-sm">
              <pre className="text-green-500 dark:text-green-400">{`curl -H "Authorization: Bearer amk_your_api_key_here" \\
  https://your-domain.com/api/v1/ext/orgs/my-org/pods`}</pre>
            </div>
          </div>
        </div>
      </section>

      {/* Scopes */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.api.authentication.scopes.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.api.authentication.scopes.description")}
        </p>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">
                  {t("docs.api.authentication.scopes.scopeHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.api.authentication.scopes.descriptionHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.api.authentication.scopes.endpointsHeader")}
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              {(
                [
                  ["podsRead", "podsReadDesc", "podsReadEndpoints"],
                  ["podsWrite", "podsWriteDesc", "podsWriteEndpoints"],
                  ["ticketsRead", "ticketsReadDesc", "ticketsReadEndpoints"],
                  ["ticketsWrite", "ticketsWriteDesc", "ticketsWriteEndpoints"],
                  ["channelsRead", "channelsReadDesc", "channelsReadEndpoints"],
                  [
                    "channelsWrite",
                    "channelsWriteDesc",
                    "channelsWriteEndpoints",
                  ],
                  ["runnersRead", "runnersReadDesc", "runnersReadEndpoints"],
                  ["reposRead", "reposReadDesc", "reposReadEndpoints"],
                ] as const
              ).map(([scope, desc, endpoints], i, arr) => (
                <tr key={scope}>
                  <td
                    className={`p-3 font-medium font-mono ${i < arr.length - 1 ? "border-b border-border" : ""}`}
                  >
                    {t(`docs.api.authentication.scopes.${scope}`)}
                  </td>
                  <td
                    className={`p-3 ${i < arr.length - 1 ? "border-b border-border" : ""}`}
                  >
                    {t(`docs.api.authentication.scopes.${desc}`)}
                  </td>
                  <td
                    className={`p-3 ${i < arr.length - 1 ? "border-b border-border" : ""}`}
                  >
                    {t(`docs.api.authentication.scopes.${endpoints}`)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>

      {/* Error Handling */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.api.authentication.errors.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.api.authentication.errors.description")}
        </p>
        <div className="overflow-x-auto">
          <table className="w-full text-sm border border-border rounded-lg">
            <thead>
              <tr className="bg-muted">
                <th className="text-left p-3 border-b border-border">
                  {t("docs.api.authentication.errors.codeHeader")}
                </th>
                <th className="text-left p-3 border-b border-border">
                  {t("docs.api.authentication.errors.descriptionHeader")}
                </th>
              </tr>
            </thead>
            <tbody className="text-muted-foreground">
              <tr>
                <td className="p-3 border-b border-border font-medium font-mono">
                  400
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.authentication.errors.badRequest")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium font-mono">
                  401
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.authentication.errors.unauthorized")}
                </td>
              </tr>
              <tr>
                <td className="p-3 border-b border-border font-medium font-mono">
                  403
                </td>
                <td className="p-3 border-b border-border">
                  {t("docs.api.authentication.errors.forbidden")}
                </td>
              </tr>
              <tr>
                <td className="p-3 font-medium font-mono">404</td>
                <td className="p-3">
                  {t("docs.api.authentication.errors.notFound")}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <DocNavigation />
    </div>
  );
}
