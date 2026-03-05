import fs from "fs/promises";
import path from "path";
import matter from "gray-matter";
import { locales, defaultLocale } from "@/lib/i18n/config";

/** Blog post frontmatter fields */
export interface PostMeta {
  slug: string;
  title: string;
  excerpt: string;
  date: string;
  author: string;
  category: string;
  readTime: number;
}

/** Full blog post with content body */
export interface Post extends PostMeta {
  content: string;
}

const CONTENT_DIR = path.join(process.cwd(), "src/content/blog");

/**
 * Read and parse a single markdown file.
 * Returns null if the file does not exist.
 */
async function readMarkdownFile(
  filePath: string
): Promise<{ meta: Record<string, unknown>; content: string } | null> {
  try {
    const raw = await fs.readFile(filePath, "utf8");
    const { data, content } = matter(raw);
    return { meta: data, content };
  } catch {
    return null;
  }
}

/**
 * Get a single post by locale and slug.
 * Falls back to the default locale (en) if the requested locale is missing.
 */
export async function getPost(
  locale: string,
  slug: string
): Promise<Post | null> {
  const validLocale = (locales as readonly string[]).includes(locale)
    ? locale
    : defaultLocale;

  // Try requested locale first
  let filePath = path.join(CONTENT_DIR, validLocale, `${slug}.md`);
  let result = await readMarkdownFile(filePath);

  // Fallback to default locale
  if (!result && validLocale !== defaultLocale) {
    filePath = path.join(CONTENT_DIR, defaultLocale, `${slug}.md`);
    result = await readMarkdownFile(filePath);
  }

  if (!result) return null;

  return {
    slug,
    title: String(result.meta.title ?? ""),
    excerpt: String(result.meta.excerpt ?? ""),
    date: String(result.meta.date ?? ""),
    author: String(result.meta.author ?? ""),
    category: String(result.meta.category ?? ""),
    readTime: Number(result.meta.readTime ?? 0),
    content: result.content,
  };
}

/**
 * Get all posts for a locale, sorted by date (newest first).
 * Falls back to the default locale for missing translations.
 */
export async function getAllPosts(locale: string): Promise<PostMeta[]> {
  const validLocale = (locales as readonly string[]).includes(locale)
    ? locale
    : defaultLocale;

  // List files from the default locale to get all slugs
  const defaultDir = path.join(CONTENT_DIR, defaultLocale);
  let files: string[];
  try {
    files = await fs.readdir(defaultDir);
  } catch {
    return [];
  }

  const slugs = files
    .filter((f) => f.endsWith(".md"))
    .map((f) => f.replace(/\.md$/, ""));

  const posts: PostMeta[] = [];

  for (const slug of slugs) {
    // Try locale-specific file first, then fallback
    let filePath = path.join(CONTENT_DIR, validLocale, `${slug}.md`);
    let result = await readMarkdownFile(filePath);

    if (!result && validLocale !== defaultLocale) {
      filePath = path.join(CONTENT_DIR, defaultLocale, `${slug}.md`);
      result = await readMarkdownFile(filePath);
    }

    if (result) {
      posts.push({
        slug,
        title: String(result.meta.title ?? ""),
        excerpt: String(result.meta.excerpt ?? ""),
        date: String(result.meta.date ?? ""),
        author: String(result.meta.author ?? ""),
        category: String(result.meta.category ?? ""),
        readTime: Number(result.meta.readTime ?? 0),
      });
    }
  }

  // Sort by date descending
  return posts.sort(
    (a, b) => new Date(b.date).getTime() - new Date(a.date).getTime()
  );
}

/**
 * Get all slugs across all locales (for generateStaticParams).
 */
export async function getAllSlugs(): Promise<string[]> {
  const dir = path.join(CONTENT_DIR, defaultLocale);
  try {
    const files = await fs.readdir(dir);
    return files.filter((f) => f.endsWith(".md")).map((f) => f.replace(/\.md$/, ""));
  } catch {
    return [];
  }
}
