import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'

export default defineConfig({
  plugins: [react()],
  test: {
    globals: true,
    environment: 'jsdom',
    // `threads` (worker_threads) shares the Node module cache across
    // parallel test files. Default `forks` spawns one process per file
    // and each fresh process re-walks pnpm's virtual store, picking up
    // a different `node_modules/.aspect_rules_js/.../node_modules/react`
    // copy for every peer-dep that pulls react in — which trips the
    // "Invalid hook call" rule across ~250 component tests under Bazel.
    // Threads share the resolved React instance, so the dedupe is
    // effectively automatic.
    pool: 'threads',
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.{test,spec}.{js,mjs,cjs,ts,mts,cts,jsx,tsx}'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html', 'cobertura'],
      reportsDirectory: './coverage',
      exclude: [
        'node_modules/',
        'src/test/',
        '**/*.d.ts',
        '**/*.config.*',
        '**/types/**',
      ],
    },
    reporters: ['default', 'junit'],
    outputFile: {
      junit: './report.xml',
    },
  },
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
      // Generated proto TS message classes (committed mirror).
      // Resolves to the same path tsconfig.json's `@proto/*` alias points to,
      // keeping vitest, ESLint, and Next.js in lockstep.
      '@proto': resolve(__dirname, '../../proto/gen/ts'),
    },
    // Force a single copy of React across the workspace. Without this,
    // vitest's forked pool workers walk pnpm's virtual store from each
    // peer-dep's own `node_modules/`, picking up multiple React instances
    // and tripping the "Invalid hook call" rule.
    dedupe: ['react', 'react-dom'],
  },
})
