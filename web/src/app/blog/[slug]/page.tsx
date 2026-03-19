import Link from "next/link";
import { notFound } from "next/navigation";
import type { Metadata } from "next";
import { getLocale, getTranslations } from "next-intl/server";
import { PageHeader, PageFooter } from "@/components/common";
import { Markdown } from "@/components/ui/markdown";
import { getPost, getAllSlugs } from "@/lib/blog";

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

export async function generateStaticParams() {
  const slugs = await getAllSlugs();
  return slugs.map((slug) => ({ slug }));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug: string }>;
}): Promise<Metadata> {
  const { slug } = await params;
  const locale = await getLocale();
  const post = await getPost(locale, slug);

  if (!post) return { title: "Post not found" };

  return {
    title: post.title,
    description: post.excerpt,
    alternates: {
      canonical: `https://agentsmesh.ai/blog/${slug}`,
    },
    openGraph: {
      title: post.title,
      description: post.excerpt,
      type: "article",
      url: `https://agentsmesh.ai/blog/${slug}`,
      publishedTime: post.date,
      authors: [post.author],
      locale: dateLocaleMap[locale] ?? "en-US",
    },
  };
}

export default async function BlogPostPage({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const locale = await getLocale();
  const t = await getTranslations();
  const post = await getPost(locale, slug);

  if (!post) notFound();

  const dateLocale = dateLocaleMap[locale] ?? "en-US";

  return (
    <div className="min-h-screen bg-background">
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify([
            {
              "@context": "https://schema.org",
              "@type": "BlogPosting",
              headline: post.title,
              description: post.excerpt,
              datePublished: post.date,
              image: `https://agentsmesh.ai/blog/${slug}/opengraph-image`,
              author: {
                "@type": "Person",
                name: post.author,
              },
              publisher: {
                "@type": "Organization",
                name: "AgentsMesh",
                url: "https://agentsmesh.ai",
              },
              url: `https://agentsmesh.ai/blog/${slug}`,
              mainEntityOfPage: {
                "@type": "WebPage",
                "@id": `https://agentsmesh.ai/blog/${slug}`,
              },
            },
            {
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
                  item: "https://agentsmesh.ai/blog",
                },
                {
                  "@type": "ListItem",
                  position: 3,
                  name: post.title,
                },
              ],
            },
          ]),
        }}
      />
      <PageHeader />

      {/* Article */}
      <article className="py-12 px-4">
        <div className="container mx-auto max-w-3xl">
          {/* Back link */}
          <Link
            href="/blog"
            className="inline-flex items-center gap-2 text-muted-foreground hover:text-foreground mb-8"
          >
            <svg
              className="w-4 h-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M15 19l-7-7 7-7"
              />
            </svg>
            {t("blog.backToList")}
          </Link>

          {/* Meta */}
          <div className="flex items-center gap-4 text-sm text-muted-foreground mb-4">
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

          {/* Title */}
          <h1 className="text-4xl font-bold mb-6">{post.title}</h1>

          {/* Author */}
          <div className="flex items-center gap-3 mb-12 pb-8 border-b border-border">
            <div className="w-10 h-10 rounded-full bg-primary/20 flex items-center justify-center">
              <span className="text-sm font-medium text-primary">
                {post.author.charAt(0)}
              </span>
            </div>
            <div>
              <p className="font-medium">{post.author}</p>
            </div>
          </div>

          {/* Content */}
          <Markdown
            content={post.content}
            className="prose-lg [&_h2]:mt-10 [&_h2]:mb-4"
          />
        </div>
      </article>

      <PageFooter />
    </div>
  );
}
