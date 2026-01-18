/**
 * 环境变量工具函数
 *
 * 本地开发时（使用 dev.sh）：
 * - NEXT_PUBLIC_API_URL="" → 使用相对路径，由 Next.js rewrites 代理
 * - NEXT_PUBLIC_OAUTH_URL → OAuth 浏览器跳转使用
 * - NEXT_PUBLIC_WS_URL → WebSocket 连接使用
 *
 * Docker/生产环境：
 * - NEXT_PUBLIC_API_URL → 完整的后端 URL
 */

/**
 * 获取 API 基础 URL
 * - 本地开发：返回空字符串（使用相对路径）
 * - Docker/生产：返回完整 URL
 */
export function getApiBaseUrl(): string {
  // NEXT_PUBLIC_API_URL="" 表示使用相对路径
  // NEXT_PUBLIC_API_URL=undefined 表示未配置，使用默认值
  if (typeof process.env.NEXT_PUBLIC_API_URL === "string") {
    return process.env.NEXT_PUBLIC_API_URL;
  }
  return "http://localhost:8080";
}

/**
 * 获取 OAuth 基础 URL（用于浏览器跳转）
 * OAuth 必须使用完整 URL，因为是浏览器直接跳转到后端
 */
export function getOAuthBaseUrl(): string {
  return (
    process.env.NEXT_PUBLIC_OAUTH_URL ||
    process.env.NEXT_PUBLIC_API_URL ||
    "http://localhost:8080"
  );
}

/**
 * 获取 WebSocket 基础 URL
 * WebSocket 必须使用完整 URL，因为不能通过 Next.js rewrites 代理
 */
export function getWsBaseUrl(): string {
  // 优先使用显式配置的 WS URL
  if (process.env.NEXT_PUBLIC_WS_URL) {
    return process.env.NEXT_PUBLIC_WS_URL;
  }

  // 从 API URL 推导
  const apiUrl = process.env.NEXT_PUBLIC_API_URL;
  if (apiUrl) {
    return apiUrl.replace(/^http/, "ws");
  }

  // 客户端：从当前页面推导
  if (typeof window !== "undefined") {
    const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
    const host = window.location.host;
    return `${protocol}//${host}`;
  }

  return "ws://localhost:8080";
}

/**
 * 获取服务器部署 URL（用于 Runner 注册等外部访问）
 * - 客户端：使用当前页面的 origin
 * - 服务端：使用 NEXT_PUBLIC_API_URL 或默认值
 *
 * @returns 完整的服务器 URL（如 https://api.agentsmesh.ai）
 */
export function getServerUrl(): string {
  // 客户端：使用当前页面的 origin
  if (typeof window !== "undefined") {
    return window.location.origin;
  }

  // 服务端：使用环境变量或默认值
  if (process.env.NEXT_PUBLIC_API_URL) {
    return process.env.NEXT_PUBLIC_API_URL;
  }

  return "https://api.agentsmesh.ai";
}
