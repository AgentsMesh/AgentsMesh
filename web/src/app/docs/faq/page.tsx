"use client";

import { useTranslations } from "next-intl";
import { DocNavigation } from "@/components/docs/DocNavigation";

export default function FAQPage() {
  const t = useTranslations();

  return (
    <div>
      <h1 className="text-4xl font-bold mb-8">
        {t("docs.faq.title")}
      </h1>

      <p className="text-muted-foreground leading-relaxed mb-8">
        {t("docs.faq.description")}
      </p>

      {/* Runner Issues */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.faq.categories.runner")}
        </h2>
        <div className="space-y-3">
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.runnerConnection.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.runnerConnection.answer")}
            </p>
          </details>
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.runnerMultiple.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.runnerMultiple.answer")}
            </p>
          </details>
        </div>
      </section>

      {/* Pod Issues */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.faq.categories.pod")}
        </h2>
        <div className="space-y-3">
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.podCreationFail.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.podCreationFail.answer")}
            </p>
          </details>
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.podStuck.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.podStuck.answer")}
            </p>
          </details>
        </div>
      </section>

      {/* API Key Configuration */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.faq.categories.apiKey")}
        </h2>
        <div className="space-y-3">
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.apiKeyFormat.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.apiKeyFormat.answer")}
            </p>
          </details>
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.apiKeyMultiple.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.apiKeyMultiple.answer")}
            </p>
          </details>
        </div>
      </section>

      {/* Git Integration */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.faq.categories.git")}
        </h2>
        <div className="space-y-3">
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.gitCloneFail.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.gitCloneFail.answer")}
            </p>
          </details>
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.gitWorktreeConflict.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.gitWorktreeConflict.answer")}
            </p>
          </details>
        </div>
      </section>

      {/* Billing & Plans */}
      <section className="mb-12">
        <h2 className="text-2xl font-semibold mb-4">
          {t("docs.faq.categories.billing")}
        </h2>
        <div className="space-y-3">
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.billingBYOK.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.billingBYOK.answer")}
            </p>
          </details>
          <details className="border border-border rounded-lg p-4">
            <summary className="font-medium cursor-pointer">
              {t("docs.faq.items.billingFree.question")}
            </summary>
            <p className="text-sm text-muted-foreground mt-3">
              {t("docs.faq.items.billingFree.answer")}
            </p>
          </details>
        </div>
      </section>

      <DocNavigation />

      {/* FAQPage JSON-LD for SEO */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "FAQPage",
            mainEntity: [
              {
                "@type": "Question",
                name: "My Runner shows as 'Offline'. How do I fix it?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Check the following: 1) Ensure the Runner process is actually running (agentsmesh-runner run). 2) Verify network connectivity to the AgentsMesh server. 3) Ensure firewalls allow outbound gRPC connections on port 9443. 4) Check Runner logs for certificate errors — you may need to re-register if the certificate has expired.",
                },
              },
              {
                "@type": "Question",
                name: "Can I run multiple Runners on the same machine?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Yes, but each Runner needs its own registration token and configuration directory. Use the --config flag to specify different config paths. Each Runner can handle multiple Pods concurrently.",
                },
              },
              {
                "@type": "Question",
                name: "Pod creation fails with 'No available runners'. What should I do?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "This means no Runner is currently online in your organization. Ensure at least one Runner is running and shows as 'Online' in Settings → Runners. If a Runner is running but shows offline, check its network connectivity.",
                },
              },
              {
                "@type": "Question",
                name: "My Pod is stuck in 'Initializing' status. What's happening?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "This usually means the repository clone is taking a long time, or the Runner is struggling to create the Git worktree. Check the Runner logs for details. For large repositories, the initial clone may take several minutes.",
                },
              },
              {
                "@type": "Question",
                name: "What format should my API keys be in?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Anthropic keys start with 'sk-ant-', OpenAI keys start with 'sk-', and Google keys are typically longer alphanumeric strings. Enter the API key exactly as provided by your AI provider. Keys are encrypted at rest.",
                },
              },
              {
                "@type": "Question",
                name: "Can I use different API keys for different Pods?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Currently, API keys are configured at the organization level. All Pods in the organization share the same API keys. This may change in future releases.",
                },
              },
              {
                "@type": "Question",
                name: "Git clone fails when creating a Pod. What should I check?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Verify: 1) The Git provider is connected in Settings → Personal → Git Settings. 2) The repository URL is correct. 3) For SSH access, ensure the Runner machine has the correct SSH keys. 4) For HTTPS, ensure the token has repository read access.",
                },
              },
              {
                "@type": "Question",
                name: "Can multiple Pods work on the same repository simultaneously?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Yes! Each Pod gets its own Git worktree, so multiple Pods can work on the same repository without conflicts. Each worktree operates on its own branch, and changes are isolated.",
                },
              },
              {
                "@type": "Question",
                name: "How does the BYOK (Bring Your Own Key) billing model work?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "You provide your own API keys for AI providers (Anthropic, OpenAI, Google, etc.). You pay the AI providers directly based on your usage. AgentsMesh charges only for platform usage (runners, storage, etc.).",
                },
              },
              {
                "@type": "Question",
                name: "Is there a free tier?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Yes, AgentsMesh offers a free tier with limited concurrent Pods and storage. Check the Pricing page for current plan details and limits.",
                },
              },
            ],
          }),
        }}
      />
    </div>
  );
}
