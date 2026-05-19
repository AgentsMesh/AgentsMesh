import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,
  // Override default ignores of eslint-config-next.
  globalIgnores([
    // Default ignores of eslint-config-next:
    ".next/**",
    "out/**",
    "build/**",
    "next-env.d.ts",
  ]),
  // Playwright fixture files use a `use()` callback that the React Hooks
  // plugin mistakes for React 19's `use()` hook. The naming collision is
  // in Playwright's public API, not something we can rename in our code.
  {
    files: ["e2e-playwright/**/*.ts", "e2e/**/*.ts"],
    rules: {
      "react-hooks/rules-of-hooks": "off",
    },
  },
  // (auth) route group is wasm-zero by design — the whole point of the
  // light-auth rollout is keeping 40MB of wasm off the critical path to
  // /login. Wasm helpers may sneak back in via the dashboard's import
  // graph; this rule fails CI before the regression lands.
  {
    files: ["src/app/(auth)/**/*.ts", "src/app/(auth)/**/*.tsx"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          paths: [
            { name: "@/lib/wasm-core", message: "(auth) routes must stay wasm-zero. Use @/lib/light-auth instead." },
            { name: "@/lib/wasm-getters", message: "(auth) routes must stay wasm-zero. Use @/lib/light-auth instead." },
            { name: "@/stores/auth", message: "(auth) routes must stay wasm-zero. Use useLightSession + @/lib/light-auth instead." },
            { name: "@/providers/WasmProvider", message: "(auth) routes must stay wasm-zero." },
            { name: "@/components/auth/AuthBootstrap", message: "(auth) routes must stay wasm-zero." },
            { name: "agentsmesh-wasm", message: "(auth) routes must stay wasm-zero." },
            { name: "@agentsmesh/service-runtime", message: "(auth) routes must stay wasm-zero." },
          ],
        },
      ],
    },
  },
]);

export default eslintConfig;
