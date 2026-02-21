/**
 * 环境变量工具函数
 *
 * =============================================================================
 * URL 解析优先级（所有 getXxxUrl 函数遵循此顺序）
 * =============================================================================
 * 1. 显式配置的环境变量（NEXT_PUBLIC_API_URL, NEXT_PUBLIC_WS_URL 等）
 * 2. 从 NEXT_PUBLIC_PRIMARY_DOMAIN + NEXT_PUBLIC_USE_HTTPS 派生
 * 3. 客户端 fallback：window.location.origin（支持 IP 访问和 on-premise）
 * 4. 默认值：localhost:10000
 *
 * =============================================================================
 * 部署场景
 * =============================================================================
 *
 * 【SaaS 生产环境】
 * - 设置 NEXT_PUBLIC_PRIMARY_DOMAIN=agentsmesh.cn
 * - 设置 NEXT_PUBLIC_USE_HTTPS=true
 *
 * 【本地开发】(dev.sh)
 * - 设置 NEXT_PUBLIC_API_URL="" → 使用相对路径，由 Next.js rewrites 代理
 *
 * 【On-premise / 纯 IP 访问】
 * - 不配置任何环境变量
 * - 客户端自动使用 window.location.origin
 * - 支持 http://192.168.1.100:3000 这类访问方式
 *
 * =============================================================================
 * 环境变量说明
 * =============================================================================
 * - NEXT_PUBLIC_PRIMARY_DOMAIN → 主域名 (e.g., "agentsmesh.cn")
 * - NEXT_PUBLIC_USE_HTTPS → 是否使用 HTTPS (true/false)
 * - NEXT_PUBLIC_API_URL → 显式指定 API URL（覆盖自动派生）
 * - NEXT_PUBLIC_WS_URL → 显式指定 WebSocket URL
 * - NEXT_PUBLIC_OAUTH_URL → 显式指定 OAuth 回调 URL
 */

// =============================================================================
// Unified Domain Configuration Helpers
// =============================================================================

/**
 * 获取主域名配置
 * 过滤未被 docker-entrypoint.sh 替换的占位符（如 "__PRIMARY_DOMAIN__"）
 */
function getPrimaryDomain(): string | undefined {
  const domain = process.env.NEXT_PUBLIC_PRIMARY_DOMAIN;
  if (domain && domain.startsWith("__")) return undefined;
  return domain;
}

/**
 * 是否使用 HTTPS
 * 过滤未被 docker-entrypoint.sh 替换的占位符（如 "__USE_HTTPS__"）
 */
function isHttpsEnabled(): boolean {
  const val = process.env.NEXT_PUBLIC_USE_HTTPS;
  if (!val || val.startsWith("__")) return false;
  return val === "true";
}

/**
 * 从 PRIMARY_DOMAIN 派生 HTTP(S) URL
 */
function deriveHttpUrl(): string | undefined {
  const domain = getPrimaryDomain();
  if (!domain) return undefined;
  const protocol = isHttpsEnabled() ? "https" : "http";
  return `${protocol}://${domain}`;
}

/**
 * 从 PRIMARY_DOMAIN 派生 WS(S) URL
 */
function deriveWsUrl(): string | undefined {
  const domain = getPrimaryDomain();
  if (!domain) return undefined;
  const protocol = isHttpsEnabled() ? "wss" : "ws";
  return `${protocol}://${domain}`;
}

// =============================================================================
// Public API
// =============================================================================

/**
 * 获取 API 基础 URL
 * - 本地开发：返回空字符串（使用相对路径，由 Next.js rewrites 代理）
 * - Docker/生产：返回完整 URL
 * - On-premise：自动使用当前页面 origin（支持 IP 访问）
 */
export function getApiBaseUrl(): string {
  // NEXT_PUBLIC_API_URL="" 表示使用相对路径（本地开发模式）
  if (process.env.NEXT_PUBLIC_API_URL === "") {
    return "";
  }

  // 显式配置的 API URL 优先
  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL;
  }

  // 浏览器端：优先使用当前页面 origin，自动继承正确协议（http/https）
  // 这样可避免 Next.js 构建时常量折叠导致 USE_HTTPS 被错误求值的问题
  if (typeof window !== "undefined") {
    return window.location.origin;
  }

  // 服务端：从 PRIMARY_DOMAIN 派生（用于 SSR fetch 调用）
  const derived = deriveHttpUrl();
  if (derived) return derived;

  return "http://localhost:10000";
}

/**
 * 获取 OAuth 基础 URL（用于浏览器跳转）
 * OAuth 必须使用完整 URL，因为是浏览器直接跳转到后端
 */
export function getOAuthBaseUrl(): string {
  // Explicit configuration takes priority
  if (process.env.NEXT_PUBLIC_OAUTH_URL) {
    return process.env.NEXT_PUBLIC_OAUTH_URL;
  }
  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL;
  }

  // 浏览器端：优先使用当前页面 origin
  if (typeof window !== "undefined") {
    return window.location.origin;
  }

  // 服务端：从 PRIMARY_DOMAIN 派生
  const derived = deriveHttpUrl();
  if (derived) return derived;

  return "http://localhost:10000";
}

/**
 * 获取 WebSocket 基础 URL
 * WebSocket 必须使用完整 URL，因为不能通过 Next.js rewrites 代理
 */
export function getWsBaseUrl(): string {
  // Explicit configuration takes priority
  if (process.env.NEXT_PUBLIC_WS_URL) {
    return process.env.NEXT_PUBLIC_WS_URL;
  }

  // Derive from API URL
  const apiUrl = process.env.NEXT_PUBLIC_API_URL;
  if (apiUrl) {
    return apiUrl.replace(/^http/, "ws");
  }

  // 浏览器端：优先从当前页面派生，自动继承正确协议
  if (typeof window !== "undefined") {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const host = window.location.host;
    return `${protocol}//${host}`;
  }

  // 服务端：从 PRIMARY_DOMAIN 派生
  const derived = deriveWsUrl();
  if (derived) return derived;

  return "ws://localhost:10000";
}

// Default server URL for SSR and production
const DEFAULT_SERVER_URL = "https://api.agentsmesh.ai";

/**
 * 获取服务器部署 URL（SSR-safe 版本）
 * 返回在服务端和客户端初始渲染时相同的值，避免 hydration mismatch
 *
 * @returns 服务器 URL（基于环境变量配置）
 */
export function getServerUrlSSR(): string {
  // 使用环境变量或默认值（服务端和客户端一致）
  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL;
  }
  return DEFAULT_SERVER_URL;
}

/**
 * 获取服务器部署 URL（用于 Runner 注册等外部访问）
 * - 客户端：使用当前页面的 origin
 * - 服务端：使用 NEXT_PUBLIC_API_URL 或默认值
 *
 * ⚠️ 注意：此函数在 SSR 组件中使用会导致 hydration mismatch
 * 对于 SSR 组件，请使用 getServerUrlSSR() 获取初始值，
 * 然后在 useEffect 中调用 getServerUrl() 更新
 *
 * @returns 完整的服务器 URL（如 https://api.agentsmesh.ai）
 */
export function getServerUrl(): string {
  // 客户端：使用当前页面的 origin
  if (typeof window !== "undefined") {
    return window.location.origin;
  }

  // 服务端：使用环境变量或默认值
  return getServerUrlSSR();
}

