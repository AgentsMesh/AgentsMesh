import Link from "next/link";
import type { Metadata } from "next";
import { getLocale, getTranslations } from "next-intl/server";
import { PageHeader, PageFooter } from "@/components/common";
import { getAllPosts } from "@/lib/blog";

/** Locale → Intl date locale mapping */
const dateLocaleMap: Record<string, string> = {
  en: "en-US",
  zh: "zh-CN",
  ja: "ja-JP",
  ko: "ko-KR",
  es: "es-ES",
  fr: "fr-FR",
  de: "de-DE",
  pt: "pt-BR",
};

export async function generateMetadata(): Promise<Metadata> {
  const t = await getTranslations();

  return {
    title: { absolute: `${t("blog.hero.title")} | AgentsMesh` },
    description: t("blog.hero.subtitle"),
    alternates: {
      canonical: "https://agentsmesh.ai/blog",
    },
    openGraph: {
      title: `${t("blog.hero.title")} | AgentsMesh`,
      description: t("blog.hero.subtitle"),
      url: "https://agentsmesh.ai/blog",
    },
  };
}

export default async function BlogPage() {
  const locale = await getLocale();
  const t = await getTranslations();
  const posts = await getAllPosts(locale);
  const dateLocale = dateLocaleMap[locale] ?? "en-US";

  return (
    <div className="min-h-screen bg-background">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "BreadcrumbList",
            itemListElement: [
              {
                "@type": "ListItem",
                position: 1,
                name: "Home",
                item: "https://agentsmesh.ai",
              },
              {
                "@type": "ListItem",
                position: 2,
                name: "Blog",
              },
            ],
          }),
        }}
      />
      <PageHeader />

      {/* Hero */}
      <section className="py-16 px-4 text-center">
        <div className="container mx-auto max-w-4xl">
          <h1 className="text-4xl md:text-5xl font-bold mb-4">
            {t("blog.hero.title")}
          </h1>
          <p className="text-xl text-muted-foreground">
            {t("blog.hero.subtitle")}
          </p>
        </div>
      </section>

      {/* Blog Posts */}
      <section className="py-12 px-4">
        <div className="container mx-auto max-w-4xl">
          <div className="space-y-8">
            {posts.map((post) => (
              <article
                key={post.slug}
                className="group p-6 rounded-xl border border-border hover:border-primary/50 transition-colors"
              >
                <Link href={`/blog/${post.slug}`}>
                  <div className="flex items-center gap-4 text-sm text-muted-foreground mb-3">
                    <span className="px-2 py-1 rounded bg-primary/10 text-primary text-xs font-medium">
                      {post.category}
                    </span>
                    <time>
                      {new Date(post.date).toLocaleDateString(dateLocale, {
                        year: "numeric",
                        month: "long",
                        day: "numeric",
                      })}
                    </time>
                    <span>•</span>
                    <span>
                      {post.readTime} {t("blog.minRead")}
                    </span>
                  </div>
                  <h2 className="text-2xl font-bold mb-2 group-hover:text-primary transition-colors">
                    {post.title}
                  </h2>
                  <p className="text-muted-foreground mb-4">{post.excerpt}</p>
                  <div className="flex items-center gap-2 text-sm">
                    <div className="w-6 h-6 rounded-full bg-primary/20 flex items-center justify-center">
                      <span className="text-xs font-medium text-primary">
                        {post.author.charAt(0)}
                      </span>
                    </div>
                    <span className="text-muted-foreground">{post.author}</span>
                  </div>
                </Link>
              </article>
            ))}
          </div>
        </div>
      </section>

      <PageFooter />
    </div>
  );
}
