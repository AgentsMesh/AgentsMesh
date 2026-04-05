/**
 * Docs navigation configuration.
 * Shared by docs/layout.tsx (sidebar + breadcrumbs) and DocNavigation (prev/next).
 */

export interface DocNavItem {
  /** i18n key for the title (resolved at render time) */
  titleKey: string;
  /** Route path */
  href: string;
}

export interface DocNavSection {
  /** i18n key for the section title */
  titleKey: string;
  items: DocNavItem[];
}

/**
 * Sectioned navigation for the sidebar.
 */
export const docsNavSections: DocNavSection[] = [
  {
    titleKey: "docs.nav.gettingStarted",
    items: [
      { titleKey: "docs.nav.introduction", href: "/docs" },
      { titleKey: "docs.nav.quickStart", href: "/docs/getting-started" },
    ],
  },
  {
    titleKey: "docs.nav.tutorials",
    items: [
      {
        titleKey: "docs.nav.tutorialRunnerSetup",
        href: "/docs/tutorials/runner-setup",
      },
      {
        titleKey: "docs.nav.tutorialGitSetup",
        href: "/docs/tutorials/git-setup",
      },
      {
        titleKey: "docs.nav.tutorialMcpSkills",
        href: "/docs/tutorials/mcp-and-skills",
      },
      {
        titleKey: "docs.nav.tutorialFirstPod",
        href: "/docs/tutorials/first-pod",
      },
      {
        titleKey: "docs.nav.tutorialTicketWorkflow",
        href: "/docs/tutorials/ticket-workflow",
      },
      {
        titleKey: "docs.nav.tutorialMultiAgent",
        href: "/docs/tutorials/multi-agent-collaboration",
      },
      {
        titleKey: "docs.nav.tutorialLoops",
        href: "/docs/tutorials/automated-loops",
      },
    ],
  },
  {
    titleKey: "docs.nav.concepts",
    items: [
      { titleKey: "docs.nav.coreConcepts", href: "/docs/concepts" },
      { titleKey: "docs.nav.agentpod", href: "/docs/features/agentpod" },
      { titleKey: "docs.nav.workspace", href: "/docs/features/workspace" },
      { titleKey: "docs.nav.channels", href: "/docs/features/channels" },
      { titleKey: "docs.nav.meshTopology", href: "/docs/features/mesh" },
      { titleKey: "docs.nav.tickets", href: "/docs/features/tickets" },
      { titleKey: "docs.nav.loops", href: "/docs/features/loops" },
      {
        titleKey: "docs.nav.repositoriesGit",
        href: "/docs/concepts/repositories-git",
      },
      {
        titleKey: "docs.nav.agentfile",
        href: "/docs/concepts/agentfile",
      },
      {
        titleKey: "docs.nav.agentfileLayer",
        href: "/docs/concepts/agentfile-layer",
      },
      { titleKey: "docs.nav.mcpTools", href: "/docs/runners/mcp-tools" },
      {
        titleKey: "docs.nav.teamManagement",
        href: "/docs/guides/team-management",
      },
    ],
  },
  {
    titleKey: "docs.nav.api",
    items: [
      { titleKey: "docs.nav.apiOverview", href: "/docs/api" },
      {
        titleKey: "docs.nav.apiAuthentication",
        href: "/docs/api/authentication",
      },
      { titleKey: "docs.nav.apiPods", href: "/docs/api/pods" },
      { titleKey: "docs.nav.apiTickets", href: "/docs/api/tickets" },
      { titleKey: "docs.nav.apiChannels", href: "/docs/api/channels" },
      { titleKey: "docs.nav.apiLoops", href: "/docs/api/loops" },
      { titleKey: "docs.nav.apiRunners", href: "/docs/api/runners" },
      {
        titleKey: "docs.nav.apiRepositories",
        href: "/docs/api/repositories",
      },
    ],
  },
  {
    titleKey: "docs.nav.help",
    items: [{ titleKey: "docs.nav.faq", href: "/docs/faq" }],
  },
];

/**
 * Flat ordered list of all doc pages for prev/next navigation.
 */
export const docsNavFlat: DocNavItem[] = docsNavSections.flatMap(
  (section) => section.items
);

/**
 * Find the current page index in the flat navigation list.
 */
export function findCurrentPageIndex(pathname: string): number {
  return docsNavFlat.findIndex((item) => item.href === pathname);
}

/**
 * Get previous and next navigation items for a given pathname.
 */
export function getPrevNext(pathname: string): {
  prev: DocNavItem | null;
  next: DocNavItem | null;
} {
  const index = findCurrentPageIndex(pathname);
  if (index === -1) return { prev: null, next: null };
  return {
    prev: index > 0 ? docsNavFlat[index - 1] : null,
    next: index < docsNavFlat.length - 1 ? docsNavFlat[index + 1] : null,
  };
}

/**
 * Get breadcrumb data for a given pathname.
 * Returns: [{ titleKey, href }, ...]
 */
export function getBreadcrumbs(
  pathname: string
): Array<{ titleKey: string; href?: string }> {
  const crumbs: Array<{ titleKey: string; href?: string }> = [
    { titleKey: "docs.title", href: "/docs" },
  ];

  for (const section of docsNavSections) {
    const item = section.items.find((i) => i.href === pathname);
    if (item) {
      // Don't add section crumb for root "/docs" page
      if (pathname !== "/docs") {
        crumbs.push({ titleKey: section.titleKey });
        crumbs.push({ titleKey: item.titleKey });
      }
      break;
    }
  }

  return crumbs;
}
