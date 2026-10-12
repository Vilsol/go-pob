import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { availableParallelism } from 'node:os';

/** @type {import('vite').UserConfig} */
const config = {
  plugins: [tailwindcss(), sveltekit()],

  resolve: process.env.VITEST ? { conditions: ['browser'] } : undefined,

  test: {
    environment: 'node',
    testTimeout: 20_000,
    poolOptions: { forks: { maxForks: Math.max(1, Math.floor(availableParallelism() / 2)), minForks: 1 } },
    setupFiles: ['./harness/component/setup.ts']
  }
};

export default config;
