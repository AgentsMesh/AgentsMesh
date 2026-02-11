"use client";

import Link from "next/link";
import { PageHeader } from "@/components/common";
import { useTranslations, useLocale } from "next-intl";

interface BlogPost {
  slug: string;
  titleKey: string;
  excerptKey: string;
  date: string;
  author: string;
  category: string;
  readTime: number;
}

const blogPosts: BlogPost[] = [
  {
    slug: "introducing-agentsmesh",
    titleKey: "blog.posts.introducing.title",
    excerptKey: "blog.posts.introducing.excerpt",
    date: "2025-01-08",
    author: "AgentsMesh Team",
    category: "Announcement",
    readTime: 5,
  },
  {
    slug: "multi-agent-collaboration",
    titleKey: "blog.posts.multiAgent.title",
    excerptKey: "blog.posts.multiAgent.excerpt",
    date: "2024-12-20",
    author: "AgentsMesh Team",
    category: "Technical",
    readTime: 8,
  },
  {
    slug: "self-hosted-runners",
    titleKey: "blog.posts.selfHosted.title",
    excerptKey: "blog.posts.selfHosted.excerpt",
    date: "2024-12-05",
    author: "AgentsMesh Team",
    category: "Guide",
    readTime: 6,
  },
];

export default function BlogPage() {
  const t = useTranslations();
  const locale = useLocale();

  return (
    <div className="min-h-screen bg-background">
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
            {blogPosts.map((post) => (
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
                      {new Date(post.date).toLocaleDateString(
                        locale === "zh" ? "zh-CN" : "en-US",
                        {
                          year: "numeric",
                          month: "long",
                          day: "numeric",
                        }
                      )}
                    </time>
                    <span>•</span>
                    <span>
                      {post.readTime} {t("blog.minRead")}
                    </span>
                  </div>
                  <h2 className="text-2xl font-bold mb-2 group-hover:text-primary transition-colors">
                    {t(post.titleKey)}
                  </h2>
                  <p className="text-muted-foreground mb-4">
                    {t(post.excerptKey)}
                  </p>
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

      {/* Footer */}
      <footer className="border-t border-border mt-16">
        <div className="container mx-auto px-4 py-8">
          <div className="flex flex-col md:flex-row justify-between items-center gap-4">
            <p className="text-sm text-muted-foreground">
              &copy; {new Date().getFullYear()} AgentsMesh. {t("common.allRightsReserved")}
            </p>
            <div className="flex gap-6">
              <Link
                href="/privacy"
                className="text-sm text-muted-foreground hover:text-foreground"
              >
                {t("landing.footer.legal.privacy")}
              </Link>
              <Link
                href="/terms"
                className="text-sm text-muted-foreground hover:text-foreground"
              >
                {t("landing.footer.legal.terms")}
              </Link>
              <Link
                href="/docs"
                className="text-sm text-muted-foreground hover:text-foreground"
              >
                {t("landing.footer.resources.documentation")}
              </Link>
            </div>
          </div>
        </div>
      </footer>
    </div>
  );
}
