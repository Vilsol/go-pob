import { defineConfig } from '@playwright/test';
import { execFileSync } from 'node:child_process';
import { resolve } from 'node:path';

const chromium = (() => {
  if (process.env.E2E_CHROMIUM) return process.env.E2E_CHROMIUM;
  try {
    return execFileSync('which', ['chromium'], { encoding: 'utf8' }).trim() || undefined;
  } catch {
    return undefined;
  }
})();

const PORT = Number(process.env.E2E_PORT || 4391);
const WORKERS = Number(process.env.E2E_WORKERS || 5);
if (!Number.isInteger(WORKERS) || WORKERS < 1) throw new Error(`E2E_WORKERS must be a positive integer, got ${JSON.stringify(process.env.E2E_WORKERS)}`);
const server = process.env.E2E_SERVER === 'dev' ? `vite dev --port ${PORT} --strictPort` : `vite preview --port ${PORT} --strictPort`;

export default defineConfig({
  testDir: '.',
  testMatch: '**/*.spec.ts',
  workers: process.env.E2E_PERF === '1' || process.env.E2E_BASELINE === '1' ? 1 : WORKERS,
  fullyParallel: process.env.E2E_FULLY_PARALLEL !== '0',
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
      args: ['--enable-precise-memory-info', ...(process.env.E2E_GPU === '1' ? ['--enable-gpu', '--use-angle=vulkan', '--enable-features=Vulkan'] : [])]
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
