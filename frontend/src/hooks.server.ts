import { dev } from '$app/environment';
import { assets } from '$app/paths';
import type { Handle } from '@sveltejs/kit';
import { ENGINE_KEY, WASM_FILE, engineFile } from '$lib/go/wasm_file';

// the engine Settings chose decides which wasm to fetch early, so a TinyGo page never downloads standard Go's
const preload = `<script>(() => {
  let tiny = false;
  try { tiny = localStorage.getItem(${JSON.stringify(ENGINE_KEY)}) === 'tinygo'; } catch {}
  const l = document.createElement('link');
  Object.assign(l, { rel: 'preload', as: 'fetch', crossOrigin: 'anonymous', href: tiny ? ${JSON.stringify(assets + engineFile(WASM_FILE, 'tinygo'))} : ${JSON.stringify(assets + WASM_FILE)} });
  document.head.append(l);
})();</script>`;

export const handle: Handle = ({ event, resolve }) =>
  resolve(event, {
    transformPageChunk: ({ html }) => (dev ? html : html.replace('</head>', `${preload}\n</head>`))
  });
