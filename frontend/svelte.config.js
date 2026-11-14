import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';
import { keepInPrecache } from './precache.js';

/** @type {import('@sveltejs/kit').Config} */
const config = {
  preprocess: vitePreprocess({
    sourceMap: true
  }),

  kit: {
    adapter: adapter({
      pages: 'build',
      assets: 'build',
      fallback: 'index.html'
    }),
    paths: {
      base: process.env.PATH_BASE || undefined
    },
    serviceWorker: {
      register: false,
      files: keepInPrecache
    }
  },

  compilerOptions: {
    enableSourcemap: true
  }
};

export default config;
