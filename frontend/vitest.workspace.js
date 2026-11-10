import { execFileSync } from 'node:child_process';
import { realpathSync } from 'node:fs';
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

// a checkout whose node_modules links elsewhere serves its modules from their real path, outside SvelteKit's allow list
const linkedModules = {
  name: 'allow-linked-node-modules',
  configResolved(config) {
    config.server.fs.allow.push(realpathSync('node_modules'));
  }
};

export default defineWorkspace([
  'vite.config.js',
  {
    // a cold dependency cache (a new checkout, or another checkout sharing node_modules) otherwise finds these mid-run,
    // and the reload that follows fails the story files then loading
    optimizeDeps: {
      include: ['storybook/theming', '@floating-ui/dom', '@tiptap/core', '@tiptap/extension-list', '@tiptap/starter-kit', 'fast-printf', 'runed', 'comlink']
    },
    plugins: [tailwindcss(), sveltekit(), storybookTest({ configDir: '.storybook' }), linkedModules],
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
