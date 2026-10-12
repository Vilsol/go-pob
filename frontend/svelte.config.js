import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

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
      files: (file) => !/(^|\/)(go-pob\.wasm.*|_headers|\.DS_Store)$/.test(file)
    }
  },

  compilerOptions: {
    enableSourcemap: true
  }
};

export default config;
