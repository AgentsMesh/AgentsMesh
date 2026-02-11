"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/common";
import { useTranslations, useLocale } from "next-intl";

interface BlogPost {
  slug: string;
  titleKey: string;
  contentKey: string;
  date: string;
  author: string;
  category: string;
  readTime: number;
}

const blogPosts: Record<string, BlogPost> = {
  "introducing-agentsmesh": {
    slug: "introducing-agentsmesh",
    titleKey: "blog.posts.introducing.title",
    contentKey: "blog.posts.introducing.content",
    date: "2025-01-08",
    author: "AgentsMesh Team",
    category: "Announcement",
    readTime: 5,
  },
  "multi-agent-collaboration": {
    slug: "multi-agent-collaboration",
    titleKey: "blog.posts.multiAgent.title",
    contentKey: "blog.posts.multiAgent.content",
    date: "2024-12-20",
    author: "AgentsMesh Team",
    category: "Technical",
    readTime: 8,
  },
  "self-hosted-runners": {
    slug: "self-hosted-runners",
    titleKey: "blog.posts.selfHosted.title",
    contentKey: "blog.posts.selfHosted.content",
    date: "2024-12-05",
    author: "AgentsMesh Team",
    category: "Guide",
    readTime: 6,
  },
};

export default function BlogPostPage() {
  const params = useParams();
  const slug = params.slug as string;
  const t = useTranslations();
  const locale = useLocale();

  const post = blogPosts[slug];

  if (!post) {
    return (
      <div className="min-h-screen bg-background flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-bold mb-4">{t("blog.notFound")}</h1>
          <Link href="/blog">
            <Button>{t("blog.backToList")}</Button>
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-background">
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

          {/* Title */}
          <h1 className="text-4xl font-bold mb-6">{t(post.titleKey)}</h1>

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
          <div className="prose prose-neutral dark:prose-invert max-w-none">
            <p className="text-lg text-muted-foreground whitespace-pre-line">
              {t(post.contentKey)}
            </p>
          </div>
        </div>
      </article>

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
