import { execFileSync } from 'node:child_process';
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineWorkspace } from 'vitest/config';

const chromium = (() => {
  if (process.env.E2E_CHROMIUM) return process.env.E2E_CHROMIUM;
  try {
    return execFileSync('which', ['chromium'], { encoding: 'utf8' }).trim() || undefined;
  } catch {
    return undefined;
  }
})();

export default defineWorkspace([
  'vite.config.js',
  {
    plugins: [tailwindcss(), sveltekit(), storybookTest({ configDir: '.storybook' })],
    test: {
      name: 'storybook',
      testTimeout: 60_000,
      browser: {
        enabled: true,
        headless: true,
        provider: 'playwright',
        instances: [{ browser: 'chromium', launch: { executablePath: chromium } }]
      }
    }
  }
]);
