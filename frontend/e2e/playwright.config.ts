import { defineConfig } from '@playwright/test';
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { resolve } from 'node:path';

const chromium = (() => {
  // `bundled` runs the Chromium `playwright install` fetched, whatever `chromium` is on PATH
  if (process.env.E2E_CHROMIUM === 'bundled') return undefined;
  if (process.env.E2E_CHROMIUM) return process.env.E2E_CHROMIUM;
  try {
    return execFileSync('which', ['chromium'], { encoding: 'utf8' }).trim() || undefined;
  } catch {
    return undefined;
  }
})();

// one port per checkout, so two worktrees' runs do not take each other's server; every Playwright process derives the same
const PORT = Number(
  process.env.E2E_PORT ||
    20000 +
      (parseInt(
        createHash('sha256')
          .update(import.meta.dirname)
          .digest('hex')
          .slice(0, 8),
        16
      ) %
        20000)
);
const WORKERS = Number(process.env.E2E_WORKERS || 5);
if (!Number.isInteger(WORKERS) || WORKERS < 1) throw new Error(`E2E_WORKERS must be a positive integer, got ${JSON.stringify(process.env.E2E_WORKERS)}`);
// V8 recompiles the hot wasm functions with TurboFan in the background, three quarters of a spec's CPU since most tests
// boot cold; Liftoff alone boots sooner and runs a long calculation slower. Perf runs measure the engine users run, and
// E2E_TURBOFAN=1 asks for it too; E2E_JS_FLAGS=<V8 flags> replaces the choice.
const TURBOFAN = process.env.E2E_PERF === '1' || process.env.E2E_BASELINE === '1' || process.env.E2E_TURBOFAN === '1';
const server = process.env.E2E_SERVER === 'dev' ? `vite dev --port ${PORT} --strictPort` : `vite preview --port ${PORT} --strictPort`;

export default defineConfig({
  testDir: '.',
  testMatch: '**/*.spec.ts',
  workers: process.env.E2E_PERF === '1' || process.env.E2E_BASELINE === '1' ? 1 : WORKERS,
  fullyParallel: process.env.E2E_FULLY_PARALLEL !== '0',
  // @slow tests run with E2E_SLOW=1 (mise run e2e --slow, CI); a SPECS or -g selection does not lift the filter
  grepInvert: process.env.E2E_SLOW === '1' ? undefined : /@slow/,
  retries: 0,
  timeout: 10 * 60_000,
  expect: { timeout: 30_000 },
  reporter: [['list']],
  outputDir: './.results/test-output',
  globalSetup: './global-setup.ts',
  globalTeardown: './global-teardown.ts',
  use: {
    baseURL: `http://localhost:${PORT}`,
    browserName: 'chromium',
    launchOptions: {
      executablePath: chromium,
      args: [
        '--enable-precise-memory-info',
        ...(process.env.E2E_JS_FLAGS ? [`--js-flags=${process.env.E2E_JS_FLAGS}`] : TURBOFAN ? [] : ['--js-flags=--liftoff-only']),
        ...(process.env.E2E_GPU === '1' ? ['--enable-gpu', '--use-angle=vulkan', '--enable-features=Vulkan'] : [])
      ]
    },
    viewport: { width: 1600, height: 1000 },
    trace: 'off',
    screenshot: 'off',
    video: 'off'
  },
  webServer: {
    command: `npx ${server}`,
    cwd: resolve(import.meta.dirname, '..'),
    port: PORT,
    reuseExistingServer: process.env.E2E_REUSE === '1',
    timeout: 120_000
  }
});
