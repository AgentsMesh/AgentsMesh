"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function TeamManagementPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.guides.teamManagement.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.guides.teamManagement.description")}
      </p>

      {/* Overview */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.guides.teamManagement.overview.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed">
          {t("docs.guides.teamManagement.overview.description")}
        </p>
      </section>

      {/* Creating an Organization */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.guides.teamManagement.createOrg.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.guides.teamManagement.createOrg.description")}
        </p>
        <div className="border border-border rounded-lg p-6">
          <ol className="list-decimal list-inside text-muted-foreground space-y-3">
            <li>{t("docs.guides.teamManagement.createOrg.step1")}</li>
            <li>{t("docs.guides.teamManagement.createOrg.step2")}</li>
            <li>{t("docs.guides.teamManagement.createOrg.step3")}</li>
            <li>{t("docs.guides.teamManagement.createOrg.step4")}</li>
          </ol>
        </div>
      </section>

      {/* Inviting Members */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.guides.teamManagement.inviteMembers.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.guides.teamManagement.inviteMembers.description")}
        </p>
        <div className="border border-border rounded-lg p-6">
          <ol className="list-decimal list-inside text-muted-foreground space-y-3">
            <li>{t("docs.guides.teamManagement.inviteMembers.step1")}</li>
            <li>{t("docs.guides.teamManagement.inviteMembers.step2")}</li>
            <li>{t("docs.guides.teamManagement.inviteMembers.step3")}</li>
            <li>{t("docs.guides.teamManagement.inviteMembers.step4")}</li>
            <li>{t("docs.guides.teamManagement.inviteMembers.step5")}</li>
          </ol>
          <div className="bg-muted rounded-lg p-4 mt-4">
            <p className="text-sm text-muted-foreground">
              {t("docs.guides.teamManagement.inviteMembers.note")}
            </p>
          </div>
        </div>
      </section>

      {/* Roles & Permissions */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.guides.teamManagement.roles.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.guides.teamManagement.roles.description")}
        </p>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.guides.teamManagement.roles.ownerTitle")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.guides.teamManagement.roles.ownerDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.guides.teamManagement.roles.adminTitle")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.guides.teamManagement.roles.adminDesc")}
            </p>
          </div>
          <div className="border border-border rounded-lg p-4">
            <h3 className="font-medium mb-2">
              {t("docs.guides.teamManagement.roles.memberTitle")}
            </h3>
            <p className="text-sm text-muted-foreground">
              {t("docs.guides.teamManagement.roles.memberDesc")}
            </p>
          </div>
        </div>
      </section>

      {/* Organization Settings */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.guides.teamManagement.orgSettings.title")}
        </h2>
        <p className="text-muted-foreground leading-relaxed mb-4">
          {t("docs.guides.teamManagement.orgSettings.description")}
        </p>
        <div className="border border-border rounded-lg p-6">
          <ul className="list-disc list-inside text-muted-foreground space-y-3">
            <li>{t("docs.guides.teamManagement.orgSettings.general")}</li>
            <li>{t("docs.guides.teamManagement.orgSettings.members")}</li>
            <li>{t("docs.guides.teamManagement.orgSettings.runners")}</li>
            <li>{t("docs.guides.teamManagement.orgSettings.repositories")}</li>
            <li>{t("docs.guides.teamManagement.orgSettings.providers")}</li>
            <li>{t("docs.guides.teamManagement.orgSettings.billing")}</li>
          </ul>
        </div>
      </section>

      <DocNavigation />
    </div>
  );
}
