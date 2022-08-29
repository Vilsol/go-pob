import { sveltekit } from '@sveltejs/kit/vite';
// import { viteStaticCopy } from 'vite-plugin-static-copy'
import copy from 'rollup-plugin-copy';

/** @type {import('vite').UserConfig} */
const config = {
  plugins: [
    sveltekit(),
    // viteStaticCopy({
    //   targets: [
    //     {
    //       src: 'node_modules/tinymce/**',
    //       dest: 'tinymce'
    //     }
    //   ]
    // }),
    copy({
      targets: [{ src: 'node_modules/tinymce/*', dest: 'static/tinymce' }]
    })
  ]
};

export default config;
