import fs from "fs/promises";
import path from "path";
import matter from "gray-matter";
import { locales, defaultLocale } from "@/lib/i18n/config";

export interface PostMeta {
  slug: string;
  title: string;
  excerpt: string;
  date: string;
  author: string;
  category: string;
  readTime: number;
}

export interface Post extends PostMeta {
  content: string;
}

const CONTENT_DIR = path.join(process.cwd(), "src/content/blog");

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

export async function getPost(
  locale: string,
  slug: string
): Promise<Post | null> {
  const validLocale = (locales as readonly string[]).includes(locale)
    ? locale
    : defaultLocale;

  let filePath = path.join(CONTENT_DIR, validLocale, `${slug}.md`);
  let result = await readMarkdownFile(filePath);

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

export async function getAllPosts(locale: string): Promise<PostMeta[]> {
  const validLocale = (locales as readonly string[]).includes(locale)
    ? locale
    : defaultLocale;

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

  return posts.sort(
    (a, b) => new Date(b.date).getTime() - new Date(a.date).getTime()
  );
}

export async function getAllSlugs(): Promise<string[]> {
  const dir = path.join(CONTENT_DIR, defaultLocale);
  try {
    const files = await fs.readdir(dir);
    return files.filter((f) => f.endsWith(".md")).map((f) => f.replace(/\.md$/, ""));
  } catch {
    return [];
  }
}
